package scraper

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/giovannirco/rubinot-data/internal/domain"
	"github.com/giovannirco/rubinot-data/internal/validation"
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

func FetchWorldDetails(
	ctx context.Context,
	baseURL,
	worldName string,
	worldID int,
	opts FetchOptions,
) (domain.WorldDetailsResult, []string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchWorldDetails")
	defer span.End()

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "world"),
		attribute.String("rubinot.world", worldName),
		attribute.Int("rubinot.world_id", worldID),
	)

	worldResult, worldSource, err := FetchWorld(ctx, baseURL, worldName, opts)
	if err != nil {
		return domain.WorldDetailsResult{}, []string{worldSource}, err
	}

	characterURLs := make([]string, 0, len(worldResult.PlayersOnline))
	for _, player := range worldResult.PlayersOnline {
		query := url.Values{}
		query.Set("name", strings.TrimSpace(player.Name))
		characterURLs = append(characterURLs, fmt.Sprintf("%s/api/characters/search?%s", strings.TrimRight(baseURL, "/"), query.Encode()))
	}

	if len(characterURLs) == 0 {
		return domain.WorldDetailsResult{
			Name:       worldResult.Name,
			Info:       worldResult.Info,
			Characters: []domain.CharacterResult{},
		}, []string{worldSource}, nil
	}

	client := NewClient(opts)
	started := time.Now()
	bodies, batchErr := fetchBatchJSONBodies(ctx, client, characterURLs)
	scrapeDuration.WithLabelValues("world").Observe(time.Since(started).Seconds())
	sources := append([]string{worldSource}, characterURLs...)
	if batchErr != nil {
		scrapeRequests.WithLabelValues("world", "error").Inc()
		return domain.WorldDetailsResult{}, sources, batchErr
	}
	scrapeRequests.WithLabelValues("world", "ok").Inc()

	parseStarted := time.Now()
	characters := make([]domain.CharacterResult, 0, len(bodies))
	for _, body := range bodies {
		var payload characterAPIResponse
		if parseErr := parseJSONBody(body, &payload); parseErr != nil {
			ParseErrors.WithLabelValues("world", "decode_error").Inc()
			return domain.WorldDetailsResult{}, sources, parseErr
		}
		if payload.Player == nil {
			continue
		}
		characters = append(characters, mapCharacterResponse(payload))
	}

	result := domain.WorldDetailsResult{
		Name:       worldResult.Name,
		Info:       worldResult.Info,
		Characters: characters,
	}

	parseDuration.WithLabelValues("world").Observe(time.Since(parseStarted).Seconds())
	ParseItems.WithLabelValues("world").Set(float64(len(result.Characters)))
	return result, sources, nil
}

func FetchWorldDashboard(
	ctx context.Context,
	baseURL,
	worldName string,
	worldID int,
	opts FetchOptions,
) (domain.WorldDashboardResult, []string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchWorldDashboard")
	defer span.End()

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "world"),
		attribute.String("rubinot.world", worldName),
		attribute.Int("rubinot.world_id", worldID),
	)

	worldURL := fmt.Sprintf("%s/api/worlds/%s", strings.TrimRight(baseURL, "/"), url.PathEscape(strings.TrimSpace(worldName)))
	deathsURL := fmt.Sprintf("%s/api/deaths?world=%d&page=1", strings.TrimRight(baseURL, "/"), worldID)
	killstatsURL := fmt.Sprintf("%s/api/killstats?world=%d", strings.TrimRight(baseURL, "/"), worldID)
	sources := []string{worldURL, deathsURL, killstatsURL}

	client := NewClient(opts)
	started := time.Now()
	bodies, err := fetchBatchJSONBodies(ctx, client, sources)
	scrapeDuration.WithLabelValues("world").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("world", "error").Inc()
		return domain.WorldDashboardResult{}, sources, err
	}
	scrapeRequests.WithLabelValues("world", "ok").Inc()

	parseStarted := time.Now()
	var worldPayload worldDetailAPIResponse
	if parseErr := parseJSONBody(bodies[0], &worldPayload); parseErr != nil {
		ParseErrors.WithLabelValues("world", "decode_error").Inc()
		return domain.WorldDashboardResult{}, sources, parseErr
	}
	worldResult := mapWorldResponse(worldName, worldPayload)

	var deathsPayload deathsAPIResponse
	if parseErr := parseJSONBody(bodies[1], &deathsPayload); parseErr != nil {
		ParseErrors.WithLabelValues("world", "decode_error").Inc()
		return domain.WorldDashboardResult{}, sources, parseErr
	}
	deathsResult := mapDeathsResponse(worldName, DeathsFilters{Page: 1}, deathsPayload)

	var killstatsPayload killstatisticsAPIResponse
	if parseErr := parseJSONBody(bodies[2], &killstatsPayload); parseErr != nil {
		ParseErrors.WithLabelValues("world", "decode_error").Inc()
		return domain.WorldDashboardResult{}, sources, parseErr
	}
	killstatsResult := mapKillstatisticsResponse(worldName, killstatsPayload)

	result := domain.WorldDashboardResult{
		World:          worldResult,
		RecentDeaths:   deathsResult,
		KillStatistics: killstatsResult,
	}
	parseDuration.WithLabelValues("world").Observe(time.Since(parseStarted).Seconds())
	ParseItems.WithLabelValues("world").Set(float64(len(result.World.PlayersOnline)))
	return result, sources, nil
}

func fetchBatchJSONBodies(ctx context.Context, client *Client, apiURLs []string) ([]string, error) {
	if len(apiURLs) == 0 {
		return []string{}, nil
	}

	cdp, err := client.ensureCDP(ctx)
	if err != nil {
		return nil, validation.NewError(validation.ErrorFlareSolverrConnection, fmt.Sprintf("CDP init failed: %v", err), err)
	}
	if cdp == nil {
		return nil, validation.NewError(validation.ErrorFlareSolverrConnection, "CDP_URL not configured; JSON API requires CDP", nil)
	}

	type request struct {
		index int
		url   string
		path  string
	}

	requests := make([]request, 0, len(apiURLs))
	for idx, rawURL := range apiURLs {
		path, pathErr := apiPathFromURL(rawURL)
		if pathErr != nil {
			return nil, pathErr
		}
		requests = append(requests, request{
			index: idx,
			url:   rawURL,
			path:  path,
		})
	}

	bodies := make([]string, len(apiURLs))
	remaining := requests
	var lastErr error
	for attempt := 0; attempt < cdpPageFetchMaxRetries && len(remaining) > 0; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
		}

		failed := make([]request, 0)
		for start := 0; start < len(remaining); start += cdpBatchSize {
			end := start + cdpBatchSize
			if end > len(remaining) {
				end = len(remaining)
			}

			chunk := remaining[start:end]
			paths := make([]string, 0, len(chunk))
			for _, req := range chunk {
				paths = append(paths, req.path)
			}

			batchStarted := time.Now()
			results, batchErr := cdp.BatchFetch(ctx, paths)
			CDPFetchDuration.Observe(time.Since(batchStarted).Seconds())
			if batchErr != nil {
				CDPFetchRequests.WithLabelValues("error").Add(float64(len(chunk)))
				globalCDPMu.Lock()
				globalCDPReady = false
				globalCDPMu.Unlock()
				lastErr = validation.NewError(validation.ErrorFlareSolverrConnection, fmt.Sprintf("CDP batch fetch failed: %v", batchErr), batchErr)
				failed = append(failed, chunk...)
				continue
			}

			for idx, result := range results {
				req := chunk[idx]
				if result.Status != "fulfilled" {
					CDPFetchRequests.WithLabelValues("error").Inc()
					lastErr = validation.NewError(validation.ErrorUpstreamUnknown, fmt.Sprintf("CDP batch item failed: %s", strings.TrimSpace(result.Value)), nil)
					failed = append(failed, req)
					continue
				}

				trimmed := strings.TrimSpace(result.Value)
				if len(trimmed) == 0 || (trimmed[0] != '{' && trimmed[0] != '[') {
					CDPFetchRequests.WithLabelValues("non_json").Inc()
					lastErr = validation.NewError(validation.ErrorUpstreamUnknown, "CDP returned non-JSON response", nil)
					failed = append(failed, req)
					continue
				}

				CDPFetchRequests.WithLabelValues("ok").Inc()
				UpstreamStatus.WithLabelValues(endpointFromURL(req.url), "200").Inc()
				bodies[req.index] = result.Value
			}
		}

		remaining = failed
	}

	if len(remaining) > 0 {
		if lastErr == nil {
			lastErr = validation.NewError(validation.ErrorUpstreamUnknown, "CDP batch request failed", nil)
		}
		return nil, lastErr
	}

	return bodies, nil
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
