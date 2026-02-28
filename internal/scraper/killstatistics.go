package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/giovannirco/rubinot-data/internal/domain"
	"github.com/giovannirco/rubinot-data/internal/validation"
	"go.opentelemetry.io/otel/attribute"
)

type killstatisticsAPIResponse struct {
	Entries []struct {
		RaceName           string `json:"race_name"`
		PlayersKilled24h   int    `json:"players_killed_24h"`
		CreaturesKilled24h int    `json:"creatures_killed_24h"`
		PlayersKilled7d    int    `json:"players_killed_7d"`
		CreaturesKilled7d  int    `json:"creatures_killed_7d"`
	} `json:"entries"`
	Totals struct {
		PlayersKilled24h   int `json:"players_killed_24h"`
		CreaturesKilled24h int `json:"creatures_killed_24h"`
		PlayersKilled7d    int `json:"players_killed_7d"`
		CreaturesKilled7d  int `json:"creatures_killed_7d"`
	} `json:"totals"`
}

func FetchKillstatistics(
	ctx context.Context,
	baseURL,
	worldName string,
	worldID int,
	opts FetchOptions,
) (domain.KillstatisticsResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchKillstatistics")
	defer span.End()

	query := url.Values{}
	query.Set("world", strconv.Itoa(worldID))
	sourceURL := fmt.Sprintf("%s/api/killstats?%s", strings.TrimRight(baseURL, "/"), query.Encode())
	client := NewClient(opts)

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "killstatistics"),
		attribute.String("rubinot.source_url", sourceURL),
		attribute.String("rubinot.world", worldName),
	)

	started := time.Now()
	var payload killstatisticsAPIResponse
	err := client.FetchJSON(ctx, sourceURL, &payload)
	scrapeDuration.WithLabelValues("killstatistics").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("killstatistics", "error").Inc()
		return domain.KillstatisticsResult{}, sourceURL, err
	}
	scrapeRequests.WithLabelValues("killstatistics", "ok").Inc()

	parseStarted := time.Now()
	result := mapKillstatisticsResponse(worldName, payload)
	parseDuration.WithLabelValues("killstatistics").Observe(time.Since(parseStarted).Seconds())
	ParseItems.WithLabelValues("killstatistics").Set(float64(len(result.Entries)))

	return result, sourceURL, nil
}

func FetchAllWorldsKillstatistics(
	ctx context.Context,
	baseURL string,
	worlds []validation.World,
	opts FetchOptions,
) ([]domain.KillstatisticsResult, []string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchAllWorldsKillstatistics")
	defer span.End()

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "killstatistics_all"),
		attribute.Int("rubinot.world_count", len(worlds)),
	)

	apiURLs := make([]string, 0, len(worlds))
	for _, world := range worlds {
		query := url.Values{}
		query.Set("world", strconv.Itoa(world.ID))
		apiURLs = append(apiURLs, fmt.Sprintf("%s/api/killstats?%s", strings.TrimRight(baseURL, "/"), query.Encode()))
	}

	client := NewClient(opts)
	started := time.Now()
	bodies, err := fetchBatchJSONBodies(ctx, client, apiURLs)
	scrapeDuration.WithLabelValues("killstatistics_all").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("killstatistics_all", "error").Inc()
		return nil, apiURLs, err
	}
	scrapeRequests.WithLabelValues("killstatistics_all", "ok").Inc()

	results := make([]domain.KillstatisticsResult, 0, len(worlds))
	for i, body := range bodies {
		var payload killstatisticsAPIResponse
		if err := json.Unmarshal([]byte(body), &payload); err != nil {
			return nil, apiURLs, fmt.Errorf("parse killstatistics for %s: %w", worlds[i].Name, err)
		}
		results = append(results, mapKillstatisticsResponse(worlds[i].Name, payload))
	}
	ParseItems.WithLabelValues("killstatistics_all").Set(float64(len(results)))

	return results, apiURLs, nil
}

func mapKillstatisticsResponse(worldName string, payload killstatisticsAPIResponse) domain.KillstatisticsResult {
	entries := make([]domain.KillstatisticsEntry, 0, len(payload.Entries))
	for _, row := range payload.Entries {
		entries = append(entries, domain.KillstatisticsEntry{
			Race:                  strings.TrimSpace(row.RaceName),
			LastDayPlayersKilled:  row.PlayersKilled24h,
			LastDayKilled:         row.CreaturesKilled24h,
			LastWeekPlayersKilled: row.PlayersKilled7d,
			LastWeekKilled:        row.CreaturesKilled7d,
		})
	}

	return domain.KillstatisticsResult{
		World:   worldName,
		Entries: entries,
		Total: domain.KillstatisticsTotal{
			LastDayPlayersKilled:  payload.Totals.PlayersKilled24h,
			LastDayKilled:         payload.Totals.CreaturesKilled24h,
			LastWeekPlayersKilled: payload.Totals.PlayersKilled7d,
			LastWeekKilled:        payload.Totals.CreaturesKilled7d,
		},
	}
}
