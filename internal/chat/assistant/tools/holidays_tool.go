package tools

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
	"github.com/matteo-nyapa/tech-challenge-acai/internal/chat/assistant/calendar"

	"github.com/openai/openai-go/v2"
)

type HolidaysTool struct{}

func (HolidaysTool) Name() string { return "get_holidays" }
func (HolidaysTool) Description() string {
	return "Gets local bank and public holidays. Each line is 'YYYY-MM-DD: Holiday Name'."
}
func (HolidaysTool) Parameters() openai.FunctionParameters {
	return openai.FunctionParameters{
		"type": "object",
		"properties": map[string]any{
			"before_date": map[string]string{
				"type":        "string",
				"description": "Optional RFC3339 date, return holidays before this date.",
			},
			"after_date": map[string]string{
				"type":        "string",
				"description": "Optional RFC3339 date, return holidays after this date.",
			},
			"max_count": map[string]string{
				"type":        "integer",
				"description": "Optional limit of holidays to return.",
			},
		},
	}
}

func (HolidaysTool) Call(ctx context.Context, raw string) (string, error) {
	link := "https://www.officeholidays.com/ics/spain/catalonia"
	if v := os.Getenv("HOLIDAY_CALENDAR_LINK"); v != "" {
		link = v
	}

	events, err := calendar.LoadCalendar(ctx, link)

	if err != nil {
		return "failed to load holiday events", nil // devolvemos texto de error “suave”
	}

	var payload struct {
		BeforeDate time.Time `json:"before_date,omitempty"`
		AfterDate  time.Time `json:"after_date,omitempty"`
		MaxCount   int       `json:"max_count,omitempty"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "failed to parse tool call arguments: " + err.Error(), nil
	}

	var holidays []string
	for _, event := range events {
		date, err := event.GetAllDayStartAt()
		if err != nil {
			continue
		}

		if payload.MaxCount > 0 && len(holidays) >= payload.MaxCount {
			break
		}
		if !payload.BeforeDate.IsZero() && date.After(payload.BeforeDate) {
			continue
		}
		if !payload.AfterDate.IsZero() && date.Before(payload.AfterDate) {
			continue
		}

		holidays = append(holidays, date.Format(time.DateOnly)+": "+event.GetProperty(ics.ComponentPropertySummary).Value)
	}

	return strings.Join(holidays, "\n"), nil
}
