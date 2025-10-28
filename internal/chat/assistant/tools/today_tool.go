package tools

import (
	"context"
	"time"

	"github.com/openai/openai-go/v2"
)

type TodayTool struct{}

func (TodayTool) Name() string        { return "get_today_date" }
func (TodayTool) Description() string { return "Get today's date and time in RFC3339 format" }

func (TodayTool) Parameters() openai.FunctionParameters {
	return openai.FunctionParameters{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func (TodayTool) Call(ctx context.Context, _ string) (string, error) {
	return time.Now().Format(time.RFC3339), nil
}
