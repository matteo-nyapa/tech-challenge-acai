package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/matteo-nyapa/tech-challenge-acai/internal/weather"
	"github.com/openai/openai-go/v2"
)

type WeatherTool struct{}

func (WeatherTool) Name() string { return "get_weather" }
func (WeatherTool) Description() string {
	return "Get weather at the given location (and optional forecast)"
}
func (WeatherTool) Parameters() openai.FunctionParameters {
	return openai.FunctionParameters{
		"type": "object",
		"properties": map[string]any{
			"location": map[string]string{
				"type": "string",
			},
			"days": map[string]any{
				"type":        "integer",
				"description": "Optional: number of forecast days (1-10)",
				"minimum":     0,
				"maximum":     10,
			},
		},
		"required": []string{"location"},
	}
}

func (WeatherTool) Call(ctx context.Context, rawArgs string) (string, error) {
	var args struct {
		Location string `json:"location"`
		Days     int    `json:"days,omitempty"`
	}
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if strings.TrimSpace(args.Location) == "" {
		return "", fmt.Errorf(`invalid arguments: provide {"location":"<city>", "days":<optional int>}`)
	}

	res, err := weather.Fetch(ctx, args.Location, args.Days)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Location: %s, %s\n", res.Place.Name, res.Place.Country)
	fmt.Fprintf(&b, "Current: %.1f°C, %s, wind %.0f km/h (dir %.0f°)\n",
		res.Current.TemperatureC, res.Current.Condition, res.Current.WindSpeedKmh, res.Current.WindDirDeg)

	if len(res.Forecast) > 0 {
		fmt.Fprintf(&b, "Forecast (%d days):\n", len(res.Forecast))
		for _, d := range res.Forecast {
			fmt.Fprintf(&b, "- %s: %s, min %.1f°C / max %.1f°C, wind max %.0f km/h\n",
				d.Date.Format("2006-01-02"), d.Condition, d.MinTempC, d.MaxTempC, d.WindMaxKmh)
		}
	}

	return b.String(), nil
}
