package chat

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/matteo-nyapa/tech-challenge-acai/internal/chat/assistant"
	"github.com/matteo-nyapa/tech-challenge-acai/internal/chat/model"
	. "github.com/matteo-nyapa/tech-challenge-acai/internal/chat/testing"
	"github.com/matteo-nyapa/tech-challenge-acai/internal/pb"
	"github.com/twitchtv/twirp"
	"google.golang.org/protobuf/testing/protocmp"
)

type assistantStub struct {
	title string
	reply string
}

func (s assistantStub) Title(ctx context.Context, conv *model.Conversation) (string, error) {
	if s.title != "" {
		return s.title, nil
	}
	return "Test Conversation Title", nil
}

func (s assistantStub) Reply(ctx context.Context, conv *model.Conversation) (string, error) {
	if s.reply != "" {
		return s.reply, nil
	}
	return "This is a stubbed assistant reply.", nil
}

func TestServer_DescribeConversation(t *testing.T) {
	ctx := context.Background()
	srv := NewServer(model.New(ConnectMongo()), nil)

	t.Run("describe existing conversation", WithFixture(func(t *testing.T, f *Fixture) {
		c := f.CreateConversation()

		out, err := srv.DescribeConversation(ctx, &pb.DescribeConversationRequest{ConversationId: c.ID.Hex()})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, want := out.GetConversation(), c.Proto()
		if !cmp.Equal(got, want, protocmp.Transform()) {
			t.Errorf("DescribeConversation() mismatch (-got +want):\n%s", cmp.Diff(got, want, protocmp.Transform()))
		}
	}))

	t.Run("describe non existing conversation should return 404", WithFixture(func(t *testing.T, f *Fixture) {
		_, err := srv.DescribeConversation(ctx, &pb.DescribeConversationRequest{ConversationId: "08a59244257c872c5943e2a2"})
		if err == nil {
			t.Fatal("expected error for non-existing conversation, got nil")
		}

		if te, ok := err.(twirp.Error); !ok || te.Code() != twirp.NotFound {
			t.Fatalf("expected twirp.NotFound error, got %v", err)
		}
	}))
}

func TestServer_StartConversation(t *testing.T) {
	ctx := context.Background()

	// Inyectamos el stub para NO llamar a OpenAI
	asst := assistantStub{
		title: "Barcelona weather forecast",
		reply: "Today in Barcelona: 20°C, sunny (stub).",
	}

	store := model.New(ConnectMongo())
	srv := NewServer(store, asst)

	// 1) Llamar a StartConversation
	start, err := srv.StartConversation(ctx, &pb.StartConversationRequest{
		Message: "What is the weather in Barcelona today?",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 2) Validar campos básicos
	if start.GetConversationId() == "" {
		t.Fatal("expected conversation_id to be set")
	}
	if got, want := start.GetTitle(), asst.title; got == "" || got != want {
		t.Fatalf("title mismatch: got %q, want %q", got, want)
	}
	if got, want := start.GetReply(), asst.reply; got == "" || got != want {
		t.Fatalf("reply mismatch: got %q, want %q", got, want)
	}

	// 3) BONUS de persistencia: recuperar y verificar mensajes
	desc, err := srv.DescribeConversation(ctx, &pb.DescribeConversationRequest{
		ConversationId: start.GetConversationId(),
	})
	if err != nil {
		t.Fatalf("DescribeConversation failed: %v", err)
	}
	c := desc.GetConversation()
	if c == nil {
		t.Fatal("conversation should not be nil")
	}
	if len(c.GetMessages()) != 2 {
		t.Fatalf("expected 2 messages (user + assistant), got %d", len(c.GetMessages()))
	}
	if got := c.GetMessages()[0].GetContent(); got != "What is the weather in Barcelona today?" {
		t.Fatalf("first message mismatch: got %q", got)
	}
	if got := c.GetMessages()[1].GetContent(); got != asst.reply {
		t.Fatalf("assistant message mismatch: got %q want %q", got, asst.reply)
	}
}

func TestAssistant_Title_GeneratesReadableTitle(t *testing.T) {
	ctx := context.Background()
	a := assistant.New()

	conv := &model.Conversation{
		Messages: []*model.Message{
			{Role: model.RoleUser, Content: "What is the weather like in Barcelona?"},
		},
	}

	title, err := a.Title(ctx, conv)
	if err != nil {
		t.Fatalf("unexpected error from Title: %v", err)
	}

	if title == "" {
		t.Fatal("expected a non-empty title")
	}
	if len(title) > 80 {
		t.Fatalf("title should be concise (<=80 chars), got length %d", len(title))
	}
	if strings.HasSuffix(title, ".") || strings.HasSuffix(title, "?") {
		t.Fatalf("title should not end with punctuation, got %q", title)
	}
}

func TestAssistant_Title_HandlesEmptyMessagesGracefully(t *testing.T) {
	ctx := context.Background()
	a := assistant.New()

	conv := &model.Conversation{
		Messages: []*model.Message{
			{Role: model.RoleUser, Content: "   "},
		},
	}

	title, err := a.Title(ctx, conv)
	if err != nil {
		t.Fatalf("Title() returned unexpected error: %v", err)
	}

	if title == "" {
		t.Fatal("expected a fallback title, got empty")
	}
}
