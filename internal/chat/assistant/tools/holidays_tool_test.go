package tools_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/matteo-nyapa/tech-challenge-acai/internal/chat/assistant/tools"
	"github.com/stretchr/testify/require"
)

const sampleICS = `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//EN
BEGIN:VEVENT
UID:1
DTSTART;VALUE=DATE:20250101
SUMMARY:New Year's Day
END:VEVENT
BEGIN:VEVENT
UID:2
DTSTART;VALUE=DATE:20250106
SUMMARY:Epiphany
END:VEVENT
END:VCALENDAR
`

func TestHolidaysTool_Basic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/calendar")
		_, _ = w.Write([]byte(sampleICS))
	}))
	defer srv.Close()

	// forzar el link al servidor de pruebas
	old := os.Getenv("HOLIDAY_CALENDAR_LINK")
	_ = os.Setenv("HOLIDAY_CALENDAR_LINK", srv.URL)
	defer os.Setenv("HOLIDAY_CALENDAR_LINK", old)

	var ht tools.HolidaysTool
	out, err := ht.Call(context.Background(), `{"max_count":2}`)
	require.NoError(t, err)

	// resultado “YYYY-MM-DD: Summary” por línea
	require.Contains(t, out, "2025-01-01")
	require.Contains(t, out, "New Year's Day")
	require.Contains(t, out, "2025-01-06")
	require.Contains(t, out, "Epiphany")

	// múltiples líneas
	require.True(t, strings.Contains(out, "\n") || strings.Contains(out, "\r\n"))
}

func TestHolidaysTool_AfterFilter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/calendar")
		_, _ = w.Write([]byte(sampleICS))
	}))
	defer srv.Close()

	old := os.Getenv("HOLIDAY_CALENDAR_LINK")
	_ = os.Setenv("HOLIDAY_CALENDAR_LINK", srv.URL)
	defer os.Setenv("HOLIDAY_CALENDAR_LINK", old)

	var ht tools.HolidaysTool
	out, err := ht.Call(context.Background(), `{"after_date":"2025-01-01T00:00:00Z","max_count":5}`)
	require.NoError(t, err)

	require.NotContains(t, out, "New Year's Day")
	require.Contains(t, out, "Epiphany")
}
