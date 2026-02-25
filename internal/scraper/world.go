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

type worldDetailAPIResponse struct {
	World struct {
		ID           int    `json:"id"`
		Name         string `json:"name"`
		PVPType      string `json:"pvpType"`
		PVPTypeLabel string `json:"pvpTypeLabel"`
		WorldType    string `json:"worldType"`
		Locked       bool   `json:"locked"`
		CreationDate int64  `json:"creationDate"`
	} `json:"world"`
	PlayersOnline int   `json:"playersOnline"`
	Record        int   `json:"record"`
	RecordTime    int64 `json:"recordTime"`
	Players       []struct {
		Name       string `json:"name"`
		Level      int    `json:"level"`
		Vocation   string `json:"vocation"`
		VocationID int    `json:"vocationId"`
	} `json:"players"`
}

func FetchWorld(ctx context.Context, baseURL, world string, opts FetchOptions) (domain.WorldResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchWorld")
	defer span.End()

	canonicalWorld := strings.TrimSpace(world)
	sourceURL := fmt.Sprintf("%s/api/worlds/%s", strings.TrimRight(baseURL, "/"), url.PathEscape(canonicalWorld))
	client := NewClient(opts)

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "world"),
		attribute.String("rubinot.source_url", sourceURL),
		attribute.String("rubinot.world", canonicalWorld),
	)

	started := time.Now()
	var payload worldDetailAPIResponse
	err := client.FetchJSON(ctx, sourceURL, &payload)
	scrapeDuration.WithLabelValues("world").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("world", "error").Inc()
		return domain.WorldResult{}, sourceURL, err
	}
	scrapeRequests.WithLabelValues("world", "ok").Inc()

	parseStarted := time.Now()
	result := mapWorldResponse(canonicalWorld, payload)
	parseDuration.WithLabelValues("world").Observe(time.Since(parseStarted).Seconds())
	ParseItems.WithLabelValues("world").Set(float64(len(result.PlayersOnline)))

	return result, sourceURL, nil
}

func mapWorldResponse(canonicalWorld string, payload worldDetailAPIResponse) domain.WorldResult {
	players := make([]domain.PlayerOnline, 0, len(payload.Players))
	for _, player := range payload.Players {
		players = append(players, domain.PlayerOnline{
			Name:       strings.TrimSpace(player.Name),
			Level:      player.Level,
			Vocation:   strings.TrimSpace(player.Vocation),
			VocationID: player.VocationID,
		})
	}

	name := strings.TrimSpace(payload.World.Name)
	if name == "" {
		name = canonicalWorld
	}

	return domain.WorldResult{
		Name: name,
		Info: domain.WorldInfo{
			ID:            payload.World.ID,
			Status:        "online",
			PlayersOnline: payload.PlayersOnline,
			PVPType:       strings.TrimSpace(payload.World.PVPTypeLabel),
			WorldType:     strings.TrimSpace(payload.World.WorldType),
			Locked:        payload.World.Locked,
			CreationDate:  unixSecondsToRFC3339(payload.World.CreationDate),
			Record:        payload.Record,
			RecordTime:    payload.RecordTime,
		},
		PlayersOnline: players,
	}
}
