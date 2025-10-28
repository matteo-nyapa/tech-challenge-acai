package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTimeInTool_ValidZone(t *testing.T) {
	ctx := context.Background()
	tool := TimeInTool{}

	out, err := tool.Call(ctx, `{"zone":"Europe/Madrid"}`)
	require.NoError(t, err)
	require.Contains(t, out, "Europe/Madrid")
}

func TestTimeInTool_InvalidZone(t *testing.T) {
	ctx := context.Background()
	tool := TimeInTool{}

	_, err := tool.Call(ctx, `{"zone":"Mars/Phobos"}`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid time zone")
}
