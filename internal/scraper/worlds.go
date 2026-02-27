package scraper

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/giovannirco/rubinot-data/internal/domain"
	"go.opentelemetry.io/otel/attribute"
)

type worldsAPIResponse struct {
	Worlds []struct {
		ID            int    `json:"id"`
		Name          string `json:"name"`
		PVPType       string `json:"pvpType"`
		PVPTypeLabel  string `json:"pvpTypeLabel"`
		WorldType     string `json:"worldType"`
		Locked        bool   `json:"locked"`
		PlayersOnline int    `json:"playersOnline"`
	} `json:"worlds"`
	TotalOnline       int   `json:"totalOnline"`
	OverallRecord     int   `json:"overallRecord"`
	OverallRecordTime int64 `json:"overallRecordTime"`
}

func FetchWorlds(ctx context.Context, baseURL string, opts FetchOptions) (domain.WorldsResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchWorlds")
	defer span.End()

	sourceURL := fmt.Sprintf("%s/api/worlds", strings.TrimRight(baseURL, "/"))
	client := NewClient(opts)

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "worlds"),
		attribute.String("rubinot.source_url", sourceURL),
	)

	started := time.Now()
	var payload worldsAPIResponse
	err := client.FetchJSON(ctx, sourceURL, &payload)
	scrapeDuration.WithLabelValues("worlds").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("worlds", "error").Inc()
		return domain.WorldsResult{}, sourceURL, err
	}
	scrapeRequests.WithLabelValues("worlds", "ok").Inc()

	parseStarted := time.Now()
	result := mapWorldsResponse(payload)
	parseDuration.WithLabelValues("worlds").Observe(time.Since(parseStarted).Seconds())

	WorldsTotalPlayersOnline.Set(float64(result.TotalPlayersOnline))
	for _, world := range result.Worlds {
		WorldPlayersOnline.WithLabelValues(world.Name).Set(float64(world.PlayersOnline))
	}
	ParseItems.WithLabelValues("worlds").Set(float64(len(result.Worlds)))

	return result, sourceURL, nil
}

func mapWorldsResponse(payload worldsAPIResponse) domain.WorldsResult {
	worlds := make([]domain.WorldOverview, 0, len(payload.Worlds))
	for _, world := range payload.Worlds {
		overview := domain.WorldOverview{
			ID:            world.ID,
			Name:          strings.TrimSpace(world.Name),
			Status:        "online",
			PlayersOnline: world.PlayersOnline,
			PVPType:       strings.TrimSpace(world.PVPTypeLabel),
			WorldType:     strings.TrimSpace(world.WorldType),
			Locked:        world.Locked,
		}
		worlds = append(worlds, overview)
	}

	return domain.WorldsResult{
		TotalPlayersOnline: payload.TotalOnline,
		OverallRecord:      payload.OverallRecord,
		OverallRecordTime:  payload.OverallRecordTime,
		Worlds:             worlds,
	}
}
