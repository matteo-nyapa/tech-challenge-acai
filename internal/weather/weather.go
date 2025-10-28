package weather

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Geocode struct {
	Name    string
	Country string
	Lat     float64
	Lon     float64
}

type Current struct {
	TemperatureC float64
	WindSpeedKmh float64
	WindDirDeg   float64
	Condition    string
}

type DailyForecast struct {
	Date       time.Time
	MinTempC   float64
	MaxTempC   float64
	Condition  string
	WindMaxKmh float64
}

type Result struct {
	Place    Geocode
	Current  Current
	Forecast []DailyForecast
}

var HttpClient = &http.Client{Timeout: 8 * time.Second}
var BaseURL = "https://api.weatherapi.com/v1"

func SetBaseURL(url string) {
	if url == "" {
		BaseURL = "https://api.weatherapi.com/v1"
		return
	}
	BaseURL = url
}

func SetHTTPClient(client *http.Client) {
	if client == nil {
		HttpClient = &http.Client{Timeout: 8 * time.Second}
		return
	}
	HttpClient = client
}

func apiKey() (string, error) {
	k := os.Getenv("WEATHER_API_KEY")
	if k == "" {
		return "", errors.New("WEATHER_API_KEY is not set")
	}
	return k, nil
}

func Fetch(ctx context.Context, location string, days int) (Result, error) {
	key, err := apiKey()
	if err != nil {
		return Result{}, err
	}

	q := url.QueryEscape(location)
	base := BaseURL
	var endpoint string
	if days > 0 {
		if days > 10 {
			days = 10
		}
		endpoint = fmt.Sprintf("%s/forecast.json?key=%s&q=%s&days=%d&aqi=no&alerts=no", base, key, q, days)
	} else {
		endpoint = fmt.Sprintf("%s/current.json?key=%s&q=%s&aqi=no", base, key, q)
	}

	slog.InfoContext(ctx, "Fetching real weather from WeatherAPI...",
		"location", location,
		"days", days,
		"url", endpoint,
	)

	req, _ := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	resp, err := HttpClient.Do(req)
	if err != nil {
		slog.ErrorContext(ctx, "WeatherAPI request failed", "error", err)
		return Result{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		slog.WarnContext(ctx, "WeatherAPI non-200 status", "status", resp.Status)
		return Result{}, fmt.Errorf("weatherapi error: %s", resp.Status)
	}

	var data struct {
		Location struct {
			Name    string  `json:"name"`
			Country string  `json:"country"`
			Lat     float64 `json:"lat"`
			Lon     float64 `json:"lon"`
		} `json:"location"`
		Current struct {
			TempC      float64 `json:"temp_c"`
			WindKph    float64 `json:"wind_kph"`
			WindDegree float64 `json:"wind_degree"`
			Condition  struct {
				Text string `json:"text"`
			} `json:"condition"`
		} `json:"current"`
		Forecast struct {
			Forecastday []struct {
				Date string `json:"date"`
				Day  struct {
					MaxtempC   float64 `json:"maxtemp_c"`
					MintempC   float64 `json:"mintemp_c"`
					MaxwindKph float64 `json:"maxwind_kph"`
					Condition  struct {
						Text string `json:"text"`
					} `json:"condition"`
				} `json:"day"`
			} `json:"forecastday"`
		} `json:"forecast"`
		Error *struct {
			Code int    `json:"code"`
			Msg  string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		slog.ErrorContext(ctx, "Failed to decode WeatherAPI response", "error", err)
		return Result{}, err
	}
	if data.Error != nil {
		slog.ErrorContext(ctx, "WeatherAPI returned an error", "code", data.Error.Code, "msg", data.Error.Msg)
		return Result{}, fmt.Errorf("weatherapi: %s (code %d)", data.Error.Msg, data.Error.Code)
	}

	slog.InfoContext(ctx, "WeatherAPI data parsed successfully",
		"city", data.Location.Name,
		"country", data.Location.Country,
		"temp_c", data.Current.TempC,
		"condition", data.Current.Condition.Text,
	)

	res := Result{
		Place: Geocode{
			Name:    data.Location.Name,
			Country: data.Location.Country,
			Lat:     data.Location.Lat,
			Lon:     data.Location.Lon,
		},
		Current: Current{
			TemperatureC: data.Current.TempC,
			WindSpeedKmh: data.Current.WindKph,
			WindDirDeg:   data.Current.WindDegree,
			Condition:    data.Current.Condition.Text,
		},
	}

	if days > 0 && len(data.Forecast.Forecastday) > 0 {
		for _, d := range data.Forecast.Forecastday {
			dt, _ := time.Parse("2006-01-02", d.Date)
			res.Forecast = append(res.Forecast, DailyForecast{
				Date:       dt,
				MinTempC:   d.Day.MintempC,
				MaxTempC:   d.Day.MaxtempC,
				WindMaxKmh: d.Day.MaxwindKph,
				Condition:  d.Day.Condition.Text,
			})
		}

		slog.InfoContext(ctx, "Forecast parsed", "days", len(res.Forecast))
	}

	return res, nil
}
