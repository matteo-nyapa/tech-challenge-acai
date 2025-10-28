package tools_test

import (
	"context"
	"testing"
	"time"

	"github.com/matteo-nyapa/tech-challenge-acai/internal/chat/assistant/tools"
	"github.com/stretchr/testify/require"
)

func TestTodayTool_ReturnsRFC3339Now(t *testing.T) {
	var tt tools.TodayTool

	out, err := tt.Call(context.Background(), `{}`)
	require.NoError(t, err)

	parsed, err := time.Parse(time.RFC3339, out)
	require.NoError(t, err, "should be RFC3339")
	now := time.Now()
	require.WithinDuration(t, now, parsed, 5*time.Second)
}

func TestTodayTool_IgnoresArgs(t *testing.T) {
	var tt tools.TodayTool
	out, err := tt.Call(context.Background(), `{"anything":"goes"}`)
	require.NoError(t, err)
	_, err = time.Parse(time.RFC3339, out)
	require.NoError(t, err)
}
