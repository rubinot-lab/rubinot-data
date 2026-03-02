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

type guildAPIResponse struct {
	Guild struct {
		ID           int         `json:"id"`
		Name         string      `json:"name"`
		MOTD         string      `json:"motd"`
		Description  string      `json:"description"`
		Homepage     string      `json:"homepage"`
		WorldID      int         `json:"world_id"`
		LogoName     string      `json:"logo_name"`
		Balance      interface{} `json:"balance"`
		CreationData int64       `json:"creationdata"`
		Owner        *struct {
			ID       int    `json:"id"`
			Name     string `json:"name"`
			Level    int    `json:"level"`
			Vocation int    `json:"vocation"`
		} `json:"owner"`
		Members []struct {
			ID        int    `json:"id"`
			Name      string `json:"name"`
			Level     int    `json:"level"`
			Vocation  int    `json:"vocation"`
			Rank      string `json:"rank"`
			RankLevel int    `json:"rankLevel"`
			Nick      string `json:"nick"`
			JoinDate  int64  `json:"joinDate"`
			IsOnline  bool   `json:"isOnline"`
		} `json:"members"`
		Ranks []struct {
			ID    int    `json:"id"`
			Name  string `json:"name"`
			Level int    `json:"level"`
		} `json:"ranks"`
		Residence *struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
			Town string `json:"town"`
		} `json:"residence"`
	} `json:"guild"`
}

func FetchGuildsBatch(ctx context.Context, baseURL string, names []string, opts FetchOptions) ([]domain.GuildResult, []string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchGuildsBatch")
	defer span.End()

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "guild_batch"),
		attribute.Int("rubinot.batch_size", len(names)),
	)

	apiURLs := make([]string, 0, len(names))
	for _, name := range names {
		apiURLs = append(apiURLs, fmt.Sprintf("%s/api/guilds/%s", strings.TrimRight(baseURL, "/"), url.PathEscape(strings.TrimSpace(name))))
	}

	client := NewClient(opts)
	bodies, err := fetchBatchJSONBodies(ctx, client, apiURLs)
	if err != nil {
		return nil, apiURLs, err
	}

	results := make([]domain.GuildResult, 0, len(bodies))
	for _, body := range bodies {
		var payload guildAPIResponse
		if parseErr := parseJSONBody(body, &payload); parseErr != nil {
			continue
		}
		results = append(results, mapGuildResponse(payload))
	}

	return results, apiURLs, nil
}

func FetchGuild(ctx context.Context, baseURL, guildName string, opts FetchOptions) (domain.GuildResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchGuild")
	defer span.End()

	sourceURL := fmt.Sprintf("%s/api/guilds/%s", strings.TrimRight(baseURL, "/"), url.PathEscape(strings.TrimSpace(guildName)))
	client := NewClient(opts)

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "guild"),
		attribute.String("rubinot.source_url", sourceURL),
		attribute.String("rubinot.guild", guildName),
	)

	started := time.Now()
	var payload guildAPIResponse
	err := client.FetchJSON(ctx, sourceURL, &payload)
	scrapeDuration.WithLabelValues("guild").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("guild", "error").Inc()
		return domain.GuildResult{}, sourceURL, err
	}
	scrapeRequests.WithLabelValues("guild", "ok").Inc()

	parseStarted := time.Now()
	result := mapGuildResponse(payload)
	parseDuration.WithLabelValues("guild").Observe(time.Since(parseStarted).Seconds())
	ParseItems.WithLabelValues("guild").Set(float64(len(result.Members)))

	return result, sourceURL, nil
}

func mapGuildResponse(payload guildAPIResponse) domain.GuildResult {
	members := make([]domain.GuildMember, 0, len(payload.Guild.Members))
	playersOnline := 0

	for _, row := range payload.Guild.Members {
		if row.IsOnline {
			playersOnline++
		}
		members = append(members, domain.GuildMember{
			ID:         row.ID,
			Name:       strings.TrimSpace(row.Name),
			Title:      strings.TrimSpace(row.Nick),
			Rank:       strings.TrimSpace(row.Rank),
			RankLevel:  row.RankLevel,
			Vocation:   fallbackString(vocationNameByID(row.Vocation), fmt.Sprintf("%d", row.Vocation)),
			VocationID: row.Vocation,
			Level:      row.Level,
			Joined:     unixSecondsToRFC3339(row.JoinDate),
			IsOnline:   row.IsOnline,
			Status:     mapOnlineStatus(row.IsOnline),
		})
	}

	ranks := make([]domain.GuildRank, 0, len(payload.Guild.Ranks))
	for _, row := range payload.Guild.Ranks {
		ranks = append(ranks, domain.GuildRank{ID: row.ID, Name: strings.TrimSpace(row.Name), Level: row.Level})
	}

	var owner *domain.GuildOwner
	if payload.Guild.Owner != nil {
		owner = &domain.GuildOwner{
			ID:       payload.Guild.Owner.ID,
			Name:     strings.TrimSpace(payload.Guild.Owner.Name),
			Level:    payload.Guild.Owner.Level,
			Vocation: payload.Guild.Owner.Vocation,
		}
	}

	var residence *domain.GuildResidence
	if payload.Guild.Residence != nil {
		residence = &domain.GuildResidence{
			ID:   payload.Guild.Residence.ID,
			Name: strings.TrimSpace(payload.Guild.Residence.Name),
			Town: strings.TrimSpace(payload.Guild.Residence.Town),
		}
	}

	membersTotal := len(members)
	return domain.GuildResult{
		ID:             payload.Guild.ID,
		Name:           strings.TrimSpace(payload.Guild.Name),
		MOTD:           strings.TrimSpace(payload.Guild.MOTD),
		World:          worldNameByID(payload.Guild.WorldID),
		WorldID:        payload.Guild.WorldID,
		Description:    strings.TrimSpace(payload.Guild.Description),
		LogoName:       strings.TrimSpace(payload.Guild.LogoName),
		Balance:        strings.TrimSpace(fmt.Sprintf("%v", payload.Guild.Balance)),
		Residence:      residence,
		Active:         true,
		Founded:        unixSecondsToRFC3339(payload.Guild.CreationData),
		Homepage:       strings.TrimSpace(payload.Guild.Homepage),
		Owner:          owner,
		Ranks:          ranks,
		PlayersOnline:  playersOnline,
		PlayersOffline: membersTotal - playersOnline,
		MembersTotal:   membersTotal,
		Members:        members,
	}
}

func mapOnlineStatus(isOnline bool) string {
	if isOnline {
		return "online"
	}
	return "offline"
}
