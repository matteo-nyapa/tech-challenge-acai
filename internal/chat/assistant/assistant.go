package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
	"github.com/matteo-nyapa/tech-challenge-acai/internal/chat/model"
	"github.com/matteo-nyapa/tech-challenge-acai/internal/weather"
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
			Tools: []openai.ChatCompletionToolUnionParam{
				openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
					Name:        "get_weather",
					Description: openai.String("Get weather at the given location (and optional forecast)"),
					Parameters: openai.FunctionParameters{
						"type": "object",
						"properties": map[string]any{
							"location": map[string]string{
								"type": "string",
							},
							"days": map[string]string{
								"type":        "integer",
								"description": "Optional: number of forecast days (1-10)",
							},
						},
						"required": []string{"location"},
					},
				}),
				openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
					Name:        "get_today_date",
					Description: openai.String("Get today's date and time in RFC3339 format"),
				}),
				openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
					Name:        "get_holidays",
					Description: openai.String("Gets local bank and public holidays. Each line is a single holiday in the format 'YYYY-MM-DD: Holiday Name'."),
					Parameters: openai.FunctionParameters{
						"type": "object",
						"properties": map[string]any{
							"before_date": map[string]string{
								"type":        "string",
								"description": "Optional date in RFC3339 format to get holidays before this date. If not provided, all holidays will be returned.",
							},
							"after_date": map[string]string{
								"type":        "string",
								"description": "Optional date in RFC3339 format to get holidays after this date. If not provided, all holidays will be returned.",
							},
							"max_count": map[string]string{
								"type":        "integer",
								"description": "Optional maximum number of holidays to return. If not provided, all holidays will be returned.",
							},
						},
					},
				}),
			},
		})

		if err != nil {
			return "", err
		}

		if len(resp.Choices) == 0 {
			return "", errors.New("no choices returned by OpenAI")
		}

		if message := resp.Choices[0].Message; len(message.ToolCalls) > 0 {
			msgs = append(msgs, message.ToParam())

			for _, call := range message.ToolCalls {
				slog.InfoContext(ctx, "Tool call received", "name", call.Function.Name, "args", call.Function.Arguments)

				switch call.Function.Name {
				case "get_weather":
					var args struct {
						Location string `json:"location"`
						Days     int    `json:"days,omitempty"`
					}
					if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil || strings.TrimSpace(args.Location) == "" {
						msgs = append(msgs, openai.ToolMessage("invalid arguments: provide {\"location\":\"<city>\", \"days\":<optional int>}", call.ID))
						break
					}

					res, werr := weather.Fetch(ctx, args.Location, args.Days)
					if werr != nil {
						msgs = append(msgs, openai.ToolMessage("failed to fetch weather: "+werr.Error(), call.ID))
						break
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

					msgs = append(msgs, openai.ToolMessage(b.String(), call.ID))

				case "get_today_date":
					msgs = append(msgs, openai.ToolMessage(time.Now().Format(time.RFC3339), call.ID))
				case "get_holidays":
					link := "https://www.officeholidays.com/ics/spain/catalonia"
					if v := os.Getenv("HOLIDAY_CALENDAR_LINK"); v != "" {
						link = v
					}

					events, err := LoadCalendar(ctx, link)
					if err != nil {
						msgs = append(msgs, openai.ToolMessage("failed to load holiday events", call.ID))
						break
					}

					var payload struct {
						BeforeDate time.Time `json:"before_date,omitempty"`
						AfterDate  time.Time `json:"after_date,omitempty"`
						MaxCount   int       `json:"max_count,omitempty"`
					}

					if err := json.Unmarshal([]byte(call.Function.Arguments), &payload); err != nil {
						msgs = append(msgs, openai.ToolMessage("failed to parse tool call arguments: "+err.Error(), call.ID))
						break
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

					msgs = append(msgs, openai.ToolMessage(strings.Join(holidays, "\n"), call.ID))
				default:
					return "", errors.New("unknown tool call: " + call.Function.Name)
				}
			}

			continue
		}

		return resp.Choices[0].Message.Content, nil
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
