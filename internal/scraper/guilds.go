package scraper

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/giovannirco/rubinot-data/internal/domain"
	"go.opentelemetry.io/otel/attribute"
)

type guildsAPIResponse struct {
	Guilds []struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		WorldID     int    `json:"world_id"`
		LogoName    string `json:"logo_name"`
	} `json:"guilds"`
	TotalCount  int `json:"totalCount"`
	TotalPages  int `json:"totalPages"`
	CurrentPage int `json:"currentPage"`
}

func FetchGuilds(ctx context.Context, baseURL, worldName string, worldID int, opts FetchOptions) (domain.GuildsResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchGuilds")
	defer span.End()

	query := url.Values{}
	query.Set("world", strconv.Itoa(worldID))
	sourceURL := fmt.Sprintf("%s/api/guilds?%s", strings.TrimRight(baseURL, "/"), query.Encode())
	client := NewClient(opts)

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "guilds"),
		attribute.String("rubinot.source_url", sourceURL),
		attribute.String("rubinot.world", worldName),
	)

	started := time.Now()
	var payload guildsAPIResponse
	err := client.FetchJSON(ctx, sourceURL, &payload)
	scrapeDuration.WithLabelValues("guilds").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("guilds", "error").Inc()
		return domain.GuildsResult{}, sourceURL, err
	}
	scrapeRequests.WithLabelValues("guilds", "ok").Inc()

	parseStarted := time.Now()
	items := make([]domain.GuildListEntry, 0, len(payload.Guilds))
	for _, guild := range payload.Guilds {
		logoName := strings.TrimSpace(guild.LogoName)
		logoURL := ""
		if logoName != "" {
			logoURL = fmt.Sprintf("https://static.rubinot.com/guilds/%s", url.PathEscape(logoName))
		}

		items = append(items, domain.GuildListEntry{
			ID:          guild.ID,
			Name:        strings.TrimSpace(guild.Name),
			Description: strings.TrimSpace(guild.Description),
			WorldID:     guild.WorldID,
			LogoName:    logoName,
			LogoURL:     logoURL,
		})
	}

	result := domain.GuildsResult{
		World:  worldName,
		Guilds: items,
		Active: items,
	}
	parseDuration.WithLabelValues("guilds").Observe(time.Since(parseStarted).Seconds())
	ParseItems.WithLabelValues("guilds").Set(float64(len(items)))

	return result, sourceURL, nil
}
