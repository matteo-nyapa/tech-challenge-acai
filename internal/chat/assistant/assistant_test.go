package assistant_test

import (
	"context"
	"strings"
	"testing"

	"github.com/matteo-nyapa/tech-challenge-acai/internal/chat/assistant"
	"github.com/matteo-nyapa/tech-challenge-acai/internal/chat/model"
	"github.com/stretchr/testify/require"
)

func TestAssistant_Title_GeneratesConciseSummary(t *testing.T) {
	ctx := context.Background()
	a := assistant.New()

	conv := &model.Conversation{
		Messages: []*model.Message{
			{
				Role:    model.RoleUser,
				Content: "What is the weather like in Barcelona?",
			},
		},
	}

	title, err := a.Title(ctx, conv)
	require.NoError(t, err, "should not return an error")
	require.NotEmpty(t, title, "title should not be empty")

	require.LessOrEqual(t, len(title), 80, "title should be concise")
	require.NotContains(t, title, "?", "title should not contain a question mark")
	require.False(t, strings.HasSuffix(title, "."), "title should not end with punctuation")
}
