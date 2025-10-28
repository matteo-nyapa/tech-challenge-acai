package tools_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/matteo-nyapa/tech-challenge-acai/internal/chat/assistant/tools"
	"github.com/matteo-nyapa/tech-challenge-acai/internal/weather"
	"github.com/stretchr/testify/require"
)

func withWeatherServer(t *testing.T, handler http.HandlerFunc) (baseURL string, restore func()) {
	t.Helper()
	srv := httptest.NewServer(handler)

	weather.SetBaseURL(srv.URL)
	weather.SetHTTPClient(srv.Client())

	oldKey := os.Getenv("WEATHER_API_KEY")
	_ = os.Setenv("WEATHER_API_KEY", "fake-key")

	restore = func() {
		srv.Close()
		weather.SetBaseURL("https://api.weatherapi.com/v1")
		weather.SetHTTPClient(nil)
		_ = os.Setenv("WEATHER_API_KEY", oldKey)
	}
	return srv.URL, restore
}

func TestWeatherTool_Current(t *testing.T) {
	_, restore := withWeatherServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Contains(t, r.URL.Path, "/current.json")
		_ = json.NewEncoder(w).Encode(map[string]any{
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
		})
	})
	defer restore()

	var wt tools.WeatherTool
	out, err := wt.Call(context.Background(), `{"location":"Barcelona"}`)
	require.NoError(t, err)
	require.Contains(t, out, "Barcelona")
	require.Contains(t, out, "Current")
	require.Contains(t, out, "Sunny")
}

func TestWeatherTool_Forecast(t *testing.T) {
	_, restore := withWeatherServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Contains(t, r.URL.Path, "/forecast.json")
		require.Equal(t, "3", r.URL.Query().Get("days"))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"location": map[string]any{
				"name":    "Madrid",
				"country": "Spain",
				"lat":     40.41,
				"lon":     -3.7,
			},
			"current": map[string]any{
				"temp_c":      18.2,
				"wind_kph":    10.0,
				"wind_degree": 180,
				"condition":   map[string]any{"text": "Clear"},
			},
			"forecast": map[string]any{
				"forecastday": []any{
					map[string]any{
						"date": "2025-01-01",
						"day": map[string]any{
							"maxtemp_c":   20.0,
							"mintemp_c":   10.0,
							"maxwind_kph": 25.0,
							"condition":   map[string]any{"text": "Partly cloudy"},
						},
					},
					map[string]any{
						"date": "2025-01-02",
						"day": map[string]any{
							"maxtemp_c":   21.0,
							"mintemp_c":   11.0,
							"maxwind_kph": 23.0,
							"condition":   map[string]any{"text": "Sunny"},
						},
					},
				},
			},
		})
	})
	defer restore()

	var wt tools.WeatherTool
	out, err := wt.Call(context.Background(), `{"location":"Madrid","days":3}`)
	require.NoError(t, err)
	require.Contains(t, out, "Madrid")
	require.Contains(t, out, "Forecast")

	require.True(t, strings.Contains(out, "2025-01-01") || strings.Contains(out, "2025-01-02"))
}

func TestWeatherTool_InvalidArgs(t *testing.T) {
	var wt tools.WeatherTool
	_, err := wt.Call(context.Background(), `{}`)
	require.Error(t, err)
}
