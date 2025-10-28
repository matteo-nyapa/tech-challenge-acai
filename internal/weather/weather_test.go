package weather_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/matteo-nyapa/tech-challenge-acai/internal/weather"
	"github.com/stretchr/testify/require"
)

func TestFetch_CurrentWeather(t *testing.T) {
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
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(mockResp)
	}))
	defer ts.Close()

	os.Setenv("WEATHER_API_KEY", "fake-key")

	oldClient := weather.HttpClient
	weather.HttpClient = ts.Client()
	weather.BaseURL = ts.URL
	defer func() { weather.HttpClient = oldClient }()

	oldBase := "https://api.weatherapi.com/v1"
	defer func() {
		_ = oldBase
	}()

	res, err := weather.Fetch(context.Background(), "Barcelona", 0)
	require.NoError(t, err)
	require.Equal(t, "Barcelona", res.Place.Name)
	require.Equal(t, "Spain", res.Place.Country)
	require.InDelta(t, 22.5, res.Current.TemperatureC, 0.1)
	require.Equal(t, "Sunny", res.Current.Condition)
	require.Empty(t, res.Forecast)
}

func TestFetch_ForecastWeather(t *testing.T) {
	mockResp := map[string]any{
		"location": map[string]any{
			"name":    "Madrid",
			"country": "Spain",
			"lat":     40.41,
			"lon":     -3.7,
		},
		"current": map[string]any{
			"temp_c":      18.2,
			"wind_kph":    10.0,
			"wind_degree": 90,
			"condition":   map[string]any{"text": "Clear"},
		},
		"forecast": map[string]any{
			"forecastday": []map[string]any{
				{
					"date": "2025-10-28",
					"day": map[string]any{
						"maxtemp_c":   24.0,
						"mintemp_c":   16.0,
						"maxwind_kph": 12.0,
						"condition":   map[string]any{"text": "Sunny"},
					},
				},
				{
					"date": "2025-10-29",
					"day": map[string]any{
						"maxtemp_c":   22.0,
						"mintemp_c":   14.0,
						"maxwind_kph": 10.0,
						"condition":   map[string]any{"text": "Cloudy"},
					},
				},
			},
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(mockResp)
	}))
	defer ts.Close()

	os.Setenv("WEATHER_API_KEY", "fake-key")

	oldClient := weather.HttpClient
	oldBase := weather.BaseURL
	weather.HttpClient = ts.Client()
	weather.BaseURL = ts.URL
	defer func() {
		weather.HttpClient = oldClient
		weather.BaseURL = oldBase
	}()

	res, err := weather.Fetch(context.Background(), "Madrid", 3)
	require.NoError(t, err)
	require.Equal(t, "Madrid", res.Place.Name)
	require.Equal(t, "Spain", res.Place.Country)
	require.Equal(t, "Clear", res.Current.Condition)
	require.Len(t, res.Forecast, 2)
	require.Equal(t, "Sunny", res.Forecast[0].Condition)
	require.Equal(t, "Cloudy", res.Forecast[1].Condition)
}

func TestFetch_ErrorResponse(t *testing.T) {
	mockResp := map[string]any{
		"error": map[string]any{
			"code":    1006,
			"message": "No matching location found.",
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(mockResp)
	}))
	defer ts.Close()

	os.Setenv("WEATHER_API_KEY", "fake-key")

	oldClient := weather.HttpClient
	weather.HttpClient = ts.Client()
	weather.BaseURL = ts.URL
	defer func() { weather.HttpClient = oldClient }()

	_, err := weather.Fetch(context.Background(), "UnknownCity", 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "No matching location")
}
