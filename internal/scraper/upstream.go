package scraper

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/giovannirco/rubinot-data/internal/domain"
	"go.opentelemetry.io/otel/attribute"
)

type boostedAPIResponse struct {
	Boss struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		LookType int    `json:"looktype"`
	} `json:"boss"`
	Monster struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		LookType int    `json:"looktype"`
	} `json:"monster"`
}

type eventsCalendarAPIEvent struct {
	ID                 int      `json:"id"`
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	ColorDark          string   `json:"colorDark"`
	ColorLight         string   `json:"colorLight"`
	DisplayPriority    int      `json:"displayPriority"`
	SpecialEffect      *string  `json:"specialEffect"`
	StartDate          *string  `json:"startDate"`
	EndDate            *string  `json:"endDate"`
	IsRecurring        bool     `json:"isRecurring"`
	RecurringWeekdays  []int    `json:"recurringWeekdays"`
	RecurringMonthDays []int    `json:"recurringMonthDays"`
	RecurringStart     *string  `json:"recurringStart"`
	RecurringEnd       *string  `json:"recurringEnd"`
	Tags               []string `json:"tags"`
}

type eventsCalendarAPIResponse struct {
	Events      []eventsCalendarAPIEvent            `json:"events"`
	EventsByDay map[string][]eventsCalendarAPIEvent `json:"eventsByDay"`
	Month       int                                 `json:"month"`
	Year        int                                 `json:"year"`
}

type maintenanceAPIResponse struct {
	IsClosed     bool   `json:"isClosed"`
	CloseMessage string `json:"closeMessage"`
}

type geoLanguageAPIResponse struct {
	Language    string `json:"language"`
	CountryCode string `json:"countryCode"`
}

func FetchBoosted(ctx context.Context, baseURL string, opts FetchOptions) (domain.BoostedResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchBoosted")
	defer span.End()

	sourceURL := fmt.Sprintf("%s/api/boosted", strings.TrimRight(baseURL, "/"))
	span.SetAttributes(
		attribute.String("rubinot.endpoint", "boosted"),
		attribute.String("rubinot.source_url", sourceURL),
	)

	started := time.Now()
	client := NewClient(opts)
	var payload boostedAPIResponse
	err := client.FetchJSON(ctx, sourceURL, &payload)
	scrapeDuration.WithLabelValues("boosted").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("boosted", "error").Inc()
		return domain.BoostedResult{}, sourceURL, err
	}
	scrapeRequests.WithLabelValues("boosted", "ok").Inc()
	ParseItems.WithLabelValues("boosted").Set(2)

	return domain.BoostedResult{
		Boss: domain.BoostedEntity{
			ID:       payload.Boss.ID,
			Name:     strings.TrimSpace(payload.Boss.Name),
			LookType: payload.Boss.LookType,
		},
		Monster: domain.BoostedEntity{
			ID:       payload.Monster.ID,
			Name:     strings.TrimSpace(payload.Monster.Name),
			LookType: payload.Monster.LookType,
		},
	}, sourceURL, nil
}

func FetchEventsCalendar(ctx context.Context, baseURL string, opts FetchOptions) (domain.EventsCalendarResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchEventsCalendar")
	defer span.End()

	sourceURL := fmt.Sprintf("%s/api/events/calendar", strings.TrimRight(baseURL, "/"))
	span.SetAttributes(
		attribute.String("rubinot.endpoint", "events"),
		attribute.String("rubinot.source_url", sourceURL),
	)

	started := time.Now()
	client := NewClient(opts)
	var payload eventsCalendarAPIResponse
	err := client.FetchJSON(ctx, sourceURL, &payload)
	scrapeDuration.WithLabelValues("events").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("events", "error").Inc()
		return domain.EventsCalendarResult{}, sourceURL, err
	}
	scrapeRequests.WithLabelValues("events", "ok").Inc()

	result := domain.EventsCalendarResult{
		Month:       payload.Month,
		Year:        payload.Year,
		Events:      mapEventsCalendarEvents(payload.Events),
		EventsByDay: make(map[string][]domain.EventsCalendarEvent, len(payload.EventsByDay)),
	}
	for day, events := range payload.EventsByDay {
		result.EventsByDay[day] = mapEventsCalendarEvents(events)
	}
	ParseItems.WithLabelValues("events").Set(float64(len(result.Events)))
	return result, sourceURL, nil
}

func FetchMaintenance(ctx context.Context, baseURL string, opts FetchOptions) (domain.MaintenanceResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchMaintenance")
	defer span.End()

	sourceURL := fmt.Sprintf("%s/api/maintenance", strings.TrimRight(baseURL, "/"))
	span.SetAttributes(
		attribute.String("rubinot.endpoint", "maintenance"),
		attribute.String("rubinot.source_url", sourceURL),
	)

	started := time.Now()
	client := NewClient(opts)
	var payload maintenanceAPIResponse
	err := client.FetchJSON(ctx, sourceURL, &payload)
	scrapeDuration.WithLabelValues("maintenance").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("maintenance", "error").Inc()
		return domain.MaintenanceResult{}, sourceURL, err
	}
	scrapeRequests.WithLabelValues("maintenance", "ok").Inc()
	ParseItems.WithLabelValues("maintenance").Set(1)

	return domain.MaintenanceResult{
		IsClosed:     payload.IsClosed,
		CloseMessage: strings.TrimSpace(payload.CloseMessage),
	}, sourceURL, nil
}

func FetchGeoLanguage(ctx context.Context, baseURL string, opts FetchOptions) (domain.GeoLanguageResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchGeoLanguage")
	defer span.End()

	sourceURL := fmt.Sprintf("%s/api/geo-language", strings.TrimRight(baseURL, "/"))
	span.SetAttributes(
		attribute.String("rubinot.endpoint", "geo-language"),
		attribute.String("rubinot.source_url", sourceURL),
	)

	started := time.Now()
	client := NewClient(opts)
	var payload geoLanguageAPIResponse
	err := client.FetchJSON(ctx, sourceURL, &payload)
	scrapeDuration.WithLabelValues("geo-language").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("geo-language", "error").Inc()
		return domain.GeoLanguageResult{}, sourceURL, err
	}
	scrapeRequests.WithLabelValues("geo-language", "ok").Inc()
	ParseItems.WithLabelValues("geo-language").Set(1)

	return domain.GeoLanguageResult{
		Language:    strings.TrimSpace(payload.Language),
		CountryCode: strings.TrimSpace(payload.CountryCode),
	}, sourceURL, nil
}

func FetchOutfitImage(ctx context.Context, baseURL, rawQuery string, opts FetchOptions) ([]byte, string, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchOutfitImage")
	defer span.End()

	normalizedQuery := normalizeOutfitQuery(rawQuery)
	sourceURL := fmt.Sprintf("%s/api/outfit", strings.TrimRight(baseURL, "/"))
	if normalizedQuery != "" {
		sourceURL += "?" + normalizedQuery
	}
	span.SetAttributes(
		attribute.String("rubinot.endpoint", "outfit"),
		attribute.String("rubinot.source_url", sourceURL),
	)

	started := time.Now()
	client := NewClient(opts)
	body, contentType, err := client.FetchBinary(ctx, sourceURL)
	scrapeDuration.WithLabelValues("outfit").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("outfit", "error").Inc()
		return nil, "", sourceURL, err
	}
	scrapeRequests.WithLabelValues("outfit", "ok").Inc()
	ParseItems.WithLabelValues("outfit").Set(float64(len(body)))
	return body, contentType, sourceURL, nil
}

func normalizeOutfitQuery(rawQuery string) string {
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return strings.TrimSpace(rawQuery)
	}

	setFirstAvailable := func(target string, keys ...string) {
		for _, key := range keys {
			if v := strings.TrimSpace(values.Get(key)); v != "" {
				values.Set(target, v)
				return
			}
		}
	}

	setFirstAvailable("type", "type", "looktype")
	setFirstAvailable("head", "head", "lookhead")
	setFirstAvailable("body", "body", "lookbody")
	setFirstAvailable("legs", "legs", "looklegs")
	setFirstAvailable("feet", "feet", "lookfeet")
	setFirstAvailable("addons", "addons", "lookaddons")

	if strings.TrimSpace(values.Get("direction")) == "" {
		values.Set("direction", "3")
	}
	if strings.TrimSpace(values.Get("animated")) == "" {
		values.Set("animated", "1")
	}
	if strings.TrimSpace(values.Get("walk")) == "" {
		values.Set("walk", "1")
	}
	if strings.TrimSpace(values.Get("size")) == "" {
		values.Set("size", "0")
	}

	values.Del("looktype")
	values.Del("lookhead")
	values.Del("lookbody")
	values.Del("looklegs")
	values.Del("lookfeet")
	values.Del("lookaddons")
	values.Del("format")

	return values.Encode()
}

func mapEventsCalendarEvents(events []eventsCalendarAPIEvent) []domain.EventsCalendarEvent {
	mapped := make([]domain.EventsCalendarEvent, 0, len(events))
	for _, event := range events {
		mapped = append(mapped, domain.EventsCalendarEvent{
			ID:                 event.ID,
			Name:               strings.TrimSpace(event.Name),
			Description:        strings.TrimSpace(event.Description),
			ColorDark:          strings.TrimSpace(event.ColorDark),
			ColorLight:         strings.TrimSpace(event.ColorLight),
			DisplayPriority:    event.DisplayPriority,
			SpecialEffect:      event.SpecialEffect,
			StartDate:          event.StartDate,
			EndDate:            event.EndDate,
			IsRecurring:        event.IsRecurring,
			RecurringWeekdays:  event.RecurringWeekdays,
			RecurringMonthDays: event.RecurringMonthDays,
			RecurringStart:     event.RecurringStart,
			RecurringEnd:       event.RecurringEnd,
			Tags:               event.Tags,
		})
	}
	return mapped
}
