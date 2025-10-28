package assistant

import (
	"context"
	"errors"
	"log/slog"
	"regexp"
	"strings"

	"github.com/matteo-nyapa/tech-challenge-acai/internal/chat/assistant/tools"
	"github.com/matteo-nyapa/tech-challenge-acai/internal/chat/model"
	"github.com/openai/openai-go/v2"
)

type Assistant struct {
	cli openai.Client
}

func New() *Assistant {
	return &Assistant{cli: openai.NewClient()}
}

func (a *Assistant) Title(ctx context.Context, conv *model.Conversation) (string, error) {
	if len(conv.Messages) == 0 {
		return "Untitled conversation", nil
	}

	slog.InfoContext(ctx, "Generating title for conversation", "conversation_id", conv.ID)

	var firstUser string
	for _, m := range conv.Messages {
		if m.Role == model.RoleUser && strings.TrimSpace(m.Content) != "" {
			firstUser = m.Content
			break
		}
	}
	if firstUser == "" {
		firstUser = conv.Messages[0].Content
	}

	msgs := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(
			"You are a titling assistant. Generate a concise, neutral conversation TITLE (max 80 characters) summarizing the user's question. Do NOT answer the question. No quotes, no emojis, no trailing punctuation. Return ONLY the title.",
		),
		openai.UserMessage(firstUser),
	}

	resp, err := a.cli.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:    openai.ChatModelGPT4_1,
		Messages: msgs,
	})
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 || strings.TrimSpace(resp.Choices[0].Message.Content) == "" {
		return "", errors.New("empty response from OpenAI for title generation")
	}

	title := normalizeTitle(resp.Choices[0].Message.Content)
	if title == "" {
		title = normalizeTitle(firstUser)
		if title == "" {
			title = "Untitled conversation"
		}
	}
	return title, nil
}

func (a *Assistant) Reply(ctx context.Context, conv *model.Conversation) (string, error) {
	if len(conv.Messages) == 0 {
		return "", errors.New("conversation has no messages")
	}

	slog.InfoContext(ctx, "Generating reply for conversation", "conversation_id", conv.ID)

	reg := tools.NewRegistry(
		tools.WeatherTool{},
		tools.TodayTool{},
		tools.HolidaysTool{},
	)

	msgs := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage("You are a helpful, concise AI assistant. Provide accurate, safe, and clear responses."),
	}
	for _, m := range conv.Messages {
		switch m.Role {
		case model.RoleUser:
			msgs = append(msgs, openai.UserMessage(m.Content))
		case model.RoleAssistant:
			msgs = append(msgs, openai.AssistantMessage(m.Content))
		}
	}

	for i := 0; i < 15; i++ {
		resp, err := a.cli.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Model:    openai.ChatModelGPT4_1,
			Messages: msgs,
			Tools:    reg.AsOpenAITools(),
		})
		if err != nil {
			return "", err
		}
		if len(resp.Choices) == 0 {
			return "", errors.New("no choices returned by OpenAI")
		}

		message := resp.Choices[0].Message
		if len(message.ToolCalls) == 0 {
			return message.Content, nil
		}

		msgs = append(msgs, message.ToParam())
		for _, call := range message.ToolCalls {
			slog.InfoContext(ctx, "Tool call received", "name", call.Function.Name, "args", call.Function.Arguments)

			t, ok := reg.Get(call.Function.Name)
			if !ok {
				msgs = append(msgs, openai.ToolMessage("unknown tool: "+call.Function.Name, call.ID))
				continue
			}

			payload, err := t.Call(ctx, call.Function.Arguments)
			if err != nil {
				msgs = append(msgs, openai.ToolMessage("tool error: "+err.Error(), call.ID))
				continue
			}

			msgs = append(msgs, openai.ToolMessage(payload, call.ID))
		}
	}

	return "", errors.New("too many tool calls, unable to generate reply")
}

func normalizeTitle(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Trim(s, `"'`)

	re := regexp.MustCompile(`[^\p{L}\p{N}\s\-]`)
	s = re.ReplaceAllString(s, "")

	s = strings.TrimRight(s, ".!?… ")

	if len(s) > 80 {
		s = s[:80]
	}
	return strings.TrimSpace(s)
}
