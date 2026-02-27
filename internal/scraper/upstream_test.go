package scraper

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchBoostedFromAPI(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/boosted")
		writeJSON(w, map[string]any{
			"boss":    map[string]any{"id": 1226, "name": "Eradicator", "looktype": 875},
			"monster": map[string]any{"id": 1145, "name": "Vicious Squire", "looktype": 131},
		})
	}))
	defer api.Close()

	cdpSrv := newMockCDPProxyServer(t, api)
	defer cdpSrv.Close()

	result, sourceURL, err := FetchBoosted(context.Background(), baseURLOf(api), testFetchOptionsWithCDP("", cdpSrv.URL))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if sourceURL != baseURLOf(api)+"/api/boosted" {
		t.Fatalf("unexpected source URL: %s", sourceURL)
	}
	if result.Boss.Name != "Eradicator" || result.Monster.Name != "Vicious Squire" {
		t.Fatalf("unexpected boosted payload: %+v", result)
	}
}

func TestFetchEventsCalendarFromAPI(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/events/calendar")
		writeJSON(w, map[string]any{
			"events": []map[string]any{
				{
					"id":                 9,
					"name":               "Gaz'Haragoth",
					"description":        "Boss spawn",
					"colorDark":          "#735D10",
					"colorLight":         "#8B6D05",
					"displayPriority":    5,
					"specialEffect":      nil,
					"startDate":          nil,
					"endDate":            nil,
					"isRecurring":        true,
					"recurringWeekdays":  nil,
					"recurringMonthDays": []int{1, 15},
					"recurringStart":     "2026-02-01T16:00:00.000Z",
					"recurringEnd":       "2026-04-30T16:00:00.000Z",
					"tags":               []string{"boss"},
				},
			},
			"eventsByDay": map[string]any{
				"1": []map[string]any{
					{
						"id":                 9,
						"name":               "Gaz'Haragoth",
						"description":        "Boss spawn",
						"colorDark":          "#735D10",
						"colorLight":         "#8B6D05",
						"displayPriority":    5,
						"specialEffect":      nil,
						"startDate":          nil,
						"endDate":            nil,
						"isRecurring":        true,
						"recurringWeekdays":  nil,
						"recurringMonthDays": []int{1, 15},
						"recurringStart":     "2026-02-01T16:00:00.000Z",
						"recurringEnd":       "2026-04-30T16:00:00.000Z",
						"tags":               []string{"boss"},
					},
				},
			},
			"month": 2,
			"year":  2026,
		})
	}))
	defer api.Close()

	cdpSrv := newMockCDPProxyServer(t, api)
	defer cdpSrv.Close()

	result, sourceURL, err := FetchEventsCalendar(context.Background(), baseURLOf(api), testFetchOptionsWithCDP("", cdpSrv.URL))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if sourceURL != baseURLOf(api)+"/api/events/calendar" {
		t.Fatalf("unexpected source URL: %s", sourceURL)
	}
	if result.Month != 2 || result.Year != 2026 {
		t.Fatalf("unexpected month/year: %+v", result)
	}
	if len(result.Events) != 1 || len(result.EventsByDay["1"]) != 1 {
		t.Fatalf("unexpected events calendar payload: %+v", result)
	}
}

func TestFetchMaintenanceFromAPI(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/maintenance")
		writeJSON(w, map[string]any{
			"isClosed":     false,
			"closeMessage": "Server is under maintenance, please visit later.",
		})
	}))
	defer api.Close()

	cdpSrv := newMockCDPProxyServer(t, api)
	defer cdpSrv.Close()

	result, sourceURL, err := FetchMaintenance(context.Background(), baseURLOf(api), testFetchOptionsWithCDP("", cdpSrv.URL))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if sourceURL != baseURLOf(api)+"/api/maintenance" {
		t.Fatalf("unexpected source URL: %s", sourceURL)
	}
	if result.IsClosed {
		t.Fatalf("expected is_closed false, got true")
	}
}

func TestFetchGeoLanguageFromAPI(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertPath(t, r, "/api/geo-language")
		writeJSON(w, map[string]any{
			"language":    "pt",
			"countryCode": "BR",
		})
	}))
	defer api.Close()

	cdpSrv := newMockCDPProxyServer(t, api)
	defer cdpSrv.Close()

	result, sourceURL, err := FetchGeoLanguage(context.Background(), baseURLOf(api), testFetchOptionsWithCDP("", cdpSrv.URL))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if sourceURL != baseURLOf(api)+"/api/geo-language" {
		t.Fatalf("unexpected source URL: %s", sourceURL)
	}
	if result.Language != "pt" || result.CountryCode != "BR" {
		t.Fatalf("unexpected geo-language payload: %+v", result)
	}
}

func TestFetchOutfitImageFromAPI(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte{0x89, 0x50, 0x4E, 0x47})

	cdpSrv := newMockCDPServer(t, func(path string) string {
		if !strings.HasPrefix(path, "/api/outfit?looktype=131") {
			return "{}"
		}
		return fmt.Sprintf(`{"status":200,"contentType":"image/png","bodyBase64":"%s"}`, encoded)
	})
	defer cdpSrv.Close()

	body, contentType, sourceURL, err := FetchOutfitImage(
		context.Background(),
		"https://rubinot.com.br",
		"looktype=131&lookhead=0&lookbody=0&looklegs=0&lookfeet=0&lookaddons=0",
		testFetchOptionsWithCDP("", cdpSrv.URL),
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if sourceURL != "https://rubinot.com.br/api/outfit?looktype=131&lookhead=0&lookbody=0&looklegs=0&lookfeet=0&lookaddons=0" {
		t.Fatalf("unexpected source URL: %s", sourceURL)
	}
	if contentType != "image/png" {
		t.Fatalf("unexpected content type: %s", contentType)
	}
	if len(body) != 4 || body[0] != 0x89 {
		t.Fatalf("unexpected binary body: %v", body)
	}
}
