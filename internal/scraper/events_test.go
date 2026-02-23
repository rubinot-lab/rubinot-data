package scraper

import (
	"context"
	"strings"
	"testing"
)

func TestParseEventsHTMLScheduleFixture(t *testing.T) {
	html := readFixture(t, "events", "schedule.html")
	result, err := parseEventsHTML(html)
	if err != nil {
		t.Fatalf("expected schedule fixture to parse, got error: %v", err)
	}

	if result.Month == "" || result.Year != 2026 {
		t.Fatalf("expected month/year in fixture, got month=%q year=%d", result.Month, result.Year)
	}
	if result.LastUpdate == "" || !strings.HasSuffix(result.LastUpdate, "Z") {
		t.Fatalf("expected UTC RFC3339 last_update, got %q", result.LastUpdate)
	}
	if len(result.Days) == 0 {
		t.Fatal("expected at least one day with events")
	}
	if len(result.AllEvents) == 0 {
		t.Fatal("expected non-empty all_events")
	}

	firstDay := result.Days[0]
	if firstDay.Day <= 0 || len(firstDay.Events) == 0 {
		t.Fatalf("unexpected first day payload: %+v", firstDay)
	}
}

func TestParseEventsHTMLEmptyFixture(t *testing.T) {
	html := readFixture(t, "events", "empty.html")
	result, err := parseEventsHTML(html)
	if err != nil {
		t.Fatalf("expected empty fixture to parse, got error: %v", err)
	}
	if result.Year != 2027 {
		t.Fatalf("expected year=2027 in empty fixture, got %d", result.Year)
	}
	if len(result.Days) != 0 {
		t.Fatalf("expected no days with events, got %d", len(result.Days))
	}
	if len(result.AllEvents) != 0 {
		t.Fatalf("expected no all_events entries, got %d", len(result.AllEvents))
	}
}

func TestParseEventsHTMLEndingEventsFixture(t *testing.T) {
	html := readFixture(t, "events", "ending_events.html")
	result, err := parseEventsHTML(html)
	if err != nil {
		t.Fatalf("expected ending_events fixture to parse, got error: %v", err)
	}

	if len(result.Days) == 0 {
		t.Fatal("expected non-empty days in ending_events fixture")
	}

	hasEnding := false
	for _, day := range result.Days {
		if len(day.EndingEvents) > 0 {
			hasEnding = true
			break
		}
	}
	if !hasEnding {
		t.Fatal("expected at least one day with ending events")
	}
}

func TestFetchEventsScheduleHappy(t *testing.T) {
	fixture := readFixture(t, "events", "schedule.html")
	server := newFakeFlareSolverrServer(t, func(_ string) string {
		return fixture
	})
	defer server.Close()

	result, sourceURL, err := FetchEventsSchedule(
		context.Background(),
		"https://www.rubinot.com.br",
		3,
		2026,
		FetchOptions{FlareSolverrURL: server.URL, MaxTimeoutMs: 120000},
	)
	if err != nil {
		t.Fatalf("expected FetchEventsSchedule to succeed, got error: %v", err)
	}
	if !strings.Contains(sourceURL, "subtopic=eventcalendar") {
		t.Fatalf("unexpected source URL: %s", sourceURL)
	}
	if !strings.Contains(sourceURL, "calendarmonth=3") || !strings.Contains(sourceURL, "calendaryear=2026") {
		t.Fatalf("expected month/year params in source URL, got %s", sourceURL)
	}
	if len(result.Days) == 0 {
		t.Fatal("expected non-empty events result")
	}
}

func TestBuildEventsURL(t *testing.T) {
	base := buildEventsURL("https://www.rubinot.com.br", 0, 0)
	if base != "https://www.rubinot.com.br/?subtopic=eventcalendar" {
		t.Fatalf("unexpected base events URL: %s", base)
	}

	filtered := buildEventsURL("https://www.rubinot.com.br", 2, 2026)
	if !strings.Contains(filtered, "calendarmonth=2") || !strings.Contains(filtered, "calendaryear=2026") {
		t.Fatalf("expected month/year params in URL, got %s", filtered)
	}
}
