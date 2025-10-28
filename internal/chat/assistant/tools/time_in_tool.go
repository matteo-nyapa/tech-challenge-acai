package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/openai/openai-go/v2"
)

type TimeInTool struct{}

func (TimeInTool) Name() string { return "time_in" }

func (TimeInTool) Description() string {
	return "Get the current date/time for a given IANA time zone (e.g. Europe/Madrid)."
}

func (TimeInTool) Parameters() openai.FunctionParameters {
	return openai.FunctionParameters{
		"type": "object",
		"properties": map[string]any{
			"zone": map[string]any{
				"type":        "string",
				"description": "IANA time zone, e.g. Europe/Madrid, America/New_York",
			},
		},
		"required": []string{"zone"},
	}
}

func (TimeInTool) Call(ctx context.Context, rawArgs string) (string, error) {
	var args struct {
		Zone string `json:"zone"`
	}
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if args.Zone == "" {
		return "", fmt.Errorf("zone is required")
	}

	loc, err := time.LoadLocation(args.Zone)
	if err != nil {
		// <- asegura que zonas inválidas devuelven error (el test lo espera)
		return "", fmt.Errorf("invalid time zone %q: %w", args.Zone, err)
	}

	now := time.Now().In(loc)

	// Incluye explícitamente el nombre de la zona para que el test lo encuentre.
	// Ej: "2025-10-28T09:57:22Z (Europe/Madrid)"
	return fmt.Sprintf("%s (%s)", now.Format(time.RFC3339), loc.String()), nil
}
