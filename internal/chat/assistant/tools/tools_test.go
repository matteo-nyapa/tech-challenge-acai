package tools_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/matteo-nyapa/tech-challenge-acai/internal/chat/assistant/tools"
	"github.com/matteo-nyapa/tech-challenge-acai/internal/weather"
	"github.com/stretchr/testify/require"
)

func TestTodayTool_Call_ReturnsRFC3339Date(t *testing.T) {
	t.Parallel()
	tool := tools.TodayTool{}

	out, err := tool.Call(context.Background(), "")
	require.NoError(t, err)
	_, parseErr := time.Parse(time.RFC3339, out)
	require.NoError(t, parseErr, "output should be a valid RFC3339 date")
}

func TestTimeInTool_Call_ReturnsLocationTime(t *testing.T) {
	ctx := context.Background()
	reg := tools.NewRegistry(tools.TimeInTool{})

	tool, ok := reg.Get("time_in")
	require.True(t, ok, "time_in tool should exist")

	out, err := tool.Call(ctx, `{"zone":"Europe/Madrid"}`)
	require.NoError(t, err)
	require.Contains(t, out, "Europe/Madrid")
}

func TestHolidaysTool_Call_ParsesCalendar(t *testing.T) {
	t.Parallel()

	icsData := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
DTSTART;VALUE=DATE:20250106
SUMMARY:Epiphany
END:VEVENT
END:VCALENDAR`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(icsData))
	}))
	defer srv.Close()

	os.Setenv("HOLIDAY_CALENDAR_LINK", srv.URL)
	defer os.Unsetenv("HOLIDAY_CALENDAR_LINK")

	tool := tools.HolidaysTool{}
	out, err := tool.Call(context.Background(), "{}")
	require.NoError(t, err)
	require.Contains(t, out, "Epiphany")
	require.Contains(t, out, "2025-01-06")
}

func TestWeatherTool_Call_UsesMockWeatherAPI(t *testing.T) {
	t.Parallel()

	mockResp := map[string]any{
		"location": map[string]any{
			"name":    "Barcelona",
			"country": "Spain",
			"lat":     41.38,
			"lon":     2.17,
		},
		"current": map[string]any{
			"temp_c":      22.5,
			"wind_kph":    15.0,
			"wind_degree": 200,
			"condition":   map[string]any{"text": "Sunny"},
		},
		"forecast": map[string]any{
			"forecastday": []map[string]any{
				{
					"date": "2025-10-29",
					"day": map[string]any{
						"maxtemp_c":   25.0,
						"mintemp_c":   18.0,
						"maxwind_kph": 20.0,
						"condition":   map[string]any{"text": "Clear"},
					},
				},
			},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(mockResp)
	}))
	defer srv.Close()

	os.Setenv("WEATHER_API_KEY", "fake-key")
	defer os.Unsetenv("WEATHER_API_KEY")

	weather.SetBaseURL(srv.URL)
	weather.SetHTTPClient(srv.Client())

	tool := tools.WeatherTool{}
	args, _ := json.Marshal(map[string]any{"location": "Barcelona", "days": 1})
	out, err := tool.Call(context.Background(), string(args))
	require.NoError(t, err)
	require.Contains(t, out, "Barcelona")
	require.Contains(t, out, "Sunny")
	require.Contains(t, out, "Forecast")
}

func TestRegistry_BasicFunctions(t *testing.T) {
	reg := tools.NewRegistry(
		tools.TimeInTool{},
		tools.TodayTool{},
		tools.WeatherTool{},
		tools.HolidaysTool{},
	)

	_, ok := reg.Get("time_in")
	require.True(t, ok, "tool time_in should exist")
	_, ok = reg.Get("get_today_date")
	require.True(t, ok, "tool get_today_date should exist")
	_, ok = reg.Get("get_weather")
	require.True(t, ok, "tool get_weather should exist")
	_, ok = reg.Get("get_holidays")
	require.True(t, ok, "tool get_holidays should exist")

	oa := reg.AsOpenAITools()
	require.NotEmpty(t, oa)
}
