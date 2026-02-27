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
	"go.opentelemetry.io/otel/attribute"
)

const auctionListLimit = 50

type auctionListAPIResponse struct {
	Auctions []struct {
		ID                int    `json:"id"`
		State             int    `json:"state"`
		StateName         string `json:"stateName"`
		PlayerID          int    `json:"playerId"`
		Owner             int    `json:"owner"`
		StartingValue     int    `json:"startingValue"`
		CurrentValue      int    `json:"currentValue"`
		AuctionStart      int64  `json:"auctionStart"`
		AuctionEnd        int64  `json:"auctionEnd"`
		Name              string `json:"name"`
		Level             int    `json:"level"`
		Vocation          int    `json:"vocation"`
		VocationName      string `json:"vocationName"`
		Sex               int    `json:"sex"`
		WorldID           int    `json:"worldId"`
		WorldName         string `json:"worldName"`
		LookType          int    `json:"lookType"`
		LookHead          int    `json:"lookHead"`
		LookBody          int    `json:"lookBody"`
		LookLegs          int    `json:"lookLegs"`
		LookFeet          int    `json:"lookFeet"`
		LookAddons        int    `json:"lookAddons"`
		CharmPoints       int    `json:"charmPoints"`
		AchievementPoints int    `json:"achievementPoints"`
		MagLevel          int    `json:"magLevel"`
		Skills            struct {
			Axe       int `json:"axe"`
			Club      int `json:"club"`
			Sword     int `json:"sword"`
			Distance  int `json:"distance"`
			Dist      int `json:"dist"`
			Shielding int `json:"shielding"`
			Fishing   int `json:"fishing"`
			Fist      int `json:"fist"`
			Magic     int `json:"magic"`
		} `json:"skills"`
		HighlightItems []struct {
			ItemID int    `json:"itemId"`
			ID     int    `json:"id"`
			Name   string `json:"name"`
		} `json:"highlightItems"`
		HighlightAugments []struct {
			ArgType int    `json:"argType"`
			Text    string `json:"text"`
			Name    string `json:"name"`
		} `json:"highlightAugments"`
	} `json:"auctions"`
	Pagination struct {
		Page       int `json:"page"`
		Limit      int `json:"limit"`
		Total      int `json:"total"`
		TotalPages int `json:"totalPages"`
	} `json:"pagination"`
}

type auctionDetailAPIResponse struct {
	Auction struct {
		ID              int    `json:"id"`
		State           int    `json:"state"`
		StateName       string `json:"stateName"`
		PlayerID        int    `json:"playerId"`
		Owner           int    `json:"owner"`
		StartingValue   int    `json:"startingValue"`
		CurrentValue    int    `json:"currentValue"`
		WinningBid      int    `json:"winningBid"`
		HighestBidderID int    `json:"highestBidderId"`
		AuctionStart    int64  `json:"auctionStart"`
		AuctionEnd      int64  `json:"auctionEnd"`
	} `json:"auction"`
	Player struct {
		Name         string `json:"name"`
		Level        int    `json:"level"`
		Vocation     int    `json:"vocation"`
		VocationName string `json:"vocationName"`
		Sex          int    `json:"sex"`
		WorldID      int    `json:"worldId"`
		WorldName    string `json:"worldName"`
		LookType     int    `json:"lookType"`
		LookHead     int    `json:"lookHead"`
		LookBody     int    `json:"lookBody"`
		LookLegs     int    `json:"lookLegs"`
		LookFeet     int    `json:"lookFeet"`
		LookAddons   int    `json:"lookAddons"`
		LookMount    int    `json:"lookMount"`
	} `json:"player"`
	General struct {
		AchievementPoints int `json:"achievementPoints"`
		CharmPoints       int `json:"charmPoints"`
		MagLevel          int `json:"magLevel"`
		Skills            struct {
			Axe       int `json:"axe"`
			Club      int `json:"club"`
			Sword     int `json:"sword"`
			Distance  int `json:"distance"`
			Dist      int `json:"dist"`
			Shielding int `json:"shielding"`
			Fishing   int `json:"fishing"`
			Fist      int `json:"fist"`
			Magic     int `json:"magic"`
		} `json:"skills"`
	} `json:"general"`
	HighlightItems []struct {
		ItemID int    `json:"itemId"`
		ID     int    `json:"id"`
		Name   string `json:"name"`
	} `json:"highlightItems"`
	HighlightAugments []struct {
		ArgType int    `json:"argType"`
		Text    string `json:"text"`
		Name    string `json:"name"`
	} `json:"highlightAugments"`
}

func FetchCurrentAuctions(
	ctx context.Context,
	baseURL string,
	page int,
	opts FetchOptions,
) (domain.AuctionsResult, string, error) {
	return fetchAuctionsList(ctx, baseURL, "current", page, opts)
}

func FetchAuctionHistory(
	ctx context.Context,
	baseURL string,
	page int,
	opts FetchOptions,
) (domain.AuctionsResult, string, error) {
	return fetchAuctionsList(ctx, baseURL, "history", page, opts)
}

func FetchAllCurrentAuctions(
	ctx context.Context,
	baseURL string,
	opts FetchOptions,
) (domain.AuctionsResult, []string, error) {
	return fetchAllAuctionsList(ctx, baseURL, "current", opts)
}

func FetchAllAuctionHistory(
	ctx context.Context,
	baseURL string,
	opts FetchOptions,
) (domain.AuctionsResult, []string, error) {
	return fetchAllAuctionsList(ctx, baseURL, "history", opts)
}

func FetchCurrentAuctionsDetails(
	ctx context.Context,
	baseURL string,
	page int,
	opts FetchOptions,
) (domain.AuctionsDetailsResult, []string, error) {
	auctions, sourceURL, err := fetchAuctionsList(ctx, baseURL, "current", page, opts)
	if err != nil {
		return domain.AuctionsDetailsResult{}, []string{sourceURL}, err
	}
	return fetchAuctionDetailsForEntries(ctx, baseURL, auctions, []string{sourceURL}, opts)
}

func FetchAllCurrentAuctionsDetails(
	ctx context.Context,
	baseURL string,
	opts FetchOptions,
) (domain.AuctionsDetailsResult, []string, error) {
	auctions, sources, err := fetchAllAuctionsList(ctx, baseURL, "current", opts)
	if err != nil {
		return domain.AuctionsDetailsResult{}, sources, err
	}
	return fetchAuctionDetailsForEntries(ctx, baseURL, auctions, sources, opts)
}

func FetchAuctionHistoryDetails(
	ctx context.Context,
	baseURL string,
	page int,
	opts FetchOptions,
) (domain.AuctionsDetailsResult, []string, error) {
	auctions, sourceURL, err := fetchAuctionsList(ctx, baseURL, "history", page, opts)
	if err != nil {
		return domain.AuctionsDetailsResult{}, []string{sourceURL}, err
	}
	return fetchAuctionDetailsForEntries(ctx, baseURL, auctions, []string{sourceURL}, opts)
}

func FetchAllAuctionHistoryDetails(
	ctx context.Context,
	baseURL string,
	opts FetchOptions,
) (domain.AuctionsDetailsResult, []string, error) {
	auctions, sources, err := fetchAllAuctionsList(ctx, baseURL, "history", opts)
	if err != nil {
		return domain.AuctionsDetailsResult{}, sources, err
	}
	return fetchAuctionDetailsForEntries(ctx, baseURL, auctions, sources, opts)
}

func fetchAuctionsList(
	ctx context.Context,
	baseURL string,
	auctionType string,
	page int,
	opts FetchOptions,
) (domain.AuctionsResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.fetchAuctionsList")
	defer span.End()

	if page <= 0 {
		page = 1
	}

	sourceURL := buildAuctionListURL(baseURL, auctionType, page)
	client := NewClient(opts)

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "auctions"),
		attribute.String("rubinot.source_url", sourceURL),
		attribute.String("rubinot.type", auctionType),
		attribute.Int("rubinot.page", page),
	)

	started := time.Now()
	var payload auctionListAPIResponse
	err := client.FetchJSON(ctx, sourceURL, &payload)
	scrapeDuration.WithLabelValues("auctions").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("auctions", "error").Inc()
		return domain.AuctionsResult{}, sourceURL, err
	}
	scrapeRequests.WithLabelValues("auctions", "ok").Inc()

	parseStarted := time.Now()
	entries := make([]domain.AuctionEntry, 0, len(payload.Auctions))
	for _, row := range payload.Auctions {
		entries = append(entries, mapAuctionListEntry(row))
	}

	result := domain.AuctionsResult{
		Type:         auctionType,
		Page:         payload.Pagination.Page,
		TotalResults: payload.Pagination.Total,
		TotalPages:   payload.Pagination.TotalPages,
		Entries:      entries,
		Pagination: &domain.AuctionsPagination{
			Page:       payload.Pagination.Page,
			Limit:      payload.Pagination.Limit,
			Total:      payload.Pagination.Total,
			TotalPages: payload.Pagination.TotalPages,
		},
	}
	if result.Page == 0 {
		result.Page = page
	}

	parseDuration.WithLabelValues("auctions").Observe(time.Since(parseStarted).Seconds())
	ParseItems.WithLabelValues("auctions").Set(float64(len(result.Entries)))
	return result, sourceURL, nil
}

func fetchAllAuctionsList(
	ctx context.Context,
	baseURL string,
	auctionType string,
	opts FetchOptions,
) (domain.AuctionsResult, []string, error) {
	ctx, span := tracer.Start(ctx, "scraper.fetchAllAuctionsList")
	defer span.End()

	client := NewClient(opts)
	buildURL := func(page int) string {
		return buildAuctionListURL(baseURL, auctionType, page)
	}

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "auctions"),
		attribute.String("rubinot.type", auctionType),
	)

	started := time.Now()
	bodies, sources, err := client.FetchAllPages(
		ctx,
		buildURL(1),
		buildURL,
		func(body string) (int, error) {
			return auctionsTotalPagesFromBody(body)
		},
	)
	scrapeDuration.WithLabelValues("auctions").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("auctions", "error").Inc()
		return domain.AuctionsResult{}, sources, err
	}
	scrapeRequests.WithLabelValues("auctions", "ok").Inc()

	parseStarted := time.Now()
	entries := make([]domain.AuctionEntry, 0)
	totalResults := 0
	for idx, body := range bodies {
		var payload auctionListAPIResponse
		if parseErr := parseJSONBody(body, &payload); parseErr != nil {
			ParseErrors.WithLabelValues("auctions", "decode_error").Inc()
			return domain.AuctionsResult{}, sources, parseErr
		}
		if idx == 0 {
			totalResults = payload.Pagination.Total
		}

		for _, row := range payload.Auctions {
			entries = append(entries, mapAuctionListEntry(row))
		}
	}
	if totalResults <= 0 {
		totalResults = len(entries)
	}

	result := domain.AuctionsResult{
		Type:         auctionType,
		Page:         1,
		TotalResults: totalResults,
		TotalPages:   1,
		Entries:      entries,
		Pagination: &domain.AuctionsPagination{
			Page:       1,
			Limit:      auctionListLimit,
			Total:      totalResults,
			TotalPages: 1,
		},
	}

	parseDuration.WithLabelValues("auctions").Observe(time.Since(parseStarted).Seconds())
	ParseItems.WithLabelValues("auctions").Set(float64(len(result.Entries)))
	return result, sources, nil
}

func buildAuctionListURL(baseURL string, auctionType string, page int) string {
	path := "/api/bazaar"
	if auctionType == "history" {
		path = "/api/bazaar/history"
	}

	query := url.Values{}
	query.Set("page", strconv.Itoa(page))
	query.Set("limit", strconv.Itoa(auctionListLimit))
	return fmt.Sprintf("%s%s?%s", strings.TrimRight(baseURL, "/"), path, query.Encode())
}

func auctionsTotalPagesFromBody(body string) (int, error) {
	var payload struct {
		Pagination struct {
			TotalPages int `json:"totalPages"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return 0, fmt.Errorf("decode auctions pagination: %w", err)
	}
	if payload.Pagination.TotalPages <= 0 {
		return 1, nil
	}
	return payload.Pagination.TotalPages, nil
}

func FetchAuctionDetail(
	ctx context.Context,
	baseURL string,
	auctionID int,
	opts FetchOptions,
) (domain.AuctionDetail, []string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchAuctionDetail")
	defer span.End()

	sourceURL := fmt.Sprintf("%s/api/bazaar/%d", strings.TrimRight(baseURL, "/"), auctionID)
	client := NewClient(opts)

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "auction"),
		attribute.String("rubinot.source_url", sourceURL),
		attribute.Int("rubinot.auction_id", auctionID),
	)

	started := time.Now()
	var payload auctionDetailAPIResponse
	err := client.FetchJSON(ctx, sourceURL, &payload)
	scrapeDuration.WithLabelValues("auctions").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("auctions", "error").Inc()
		return domain.AuctionDetail{}, []string{sourceURL}, err
	}
	scrapeRequests.WithLabelValues("auctions", "ok").Inc()

	parseStarted := time.Now()
	result := mapAuctionDetailResponse(payload)
	parseDuration.WithLabelValues("auctions").Observe(time.Since(parseStarted).Seconds())

	return result, []string{sourceURL}, nil
}

func fetchAuctionDetailsForEntries(
	ctx context.Context,
	baseURL string,
	auctions domain.AuctionsResult,
	baseSources []string,
	opts FetchOptions,
) (domain.AuctionsDetailsResult, []string, error) {
	if len(auctions.Entries) == 0 {
		return domain.AuctionsDetailsResult{
			Type:         auctions.Type,
			Page:         auctions.Page,
			TotalResults: auctions.TotalResults,
			TotalPages:   auctions.TotalPages,
			Entries:      []domain.AuctionDetail{},
			Pagination:   auctions.Pagination,
		}, append([]string{}, baseSources...), nil
	}

	detailURLs := make([]string, 0, len(auctions.Entries))
	for _, entry := range auctions.Entries {
		detailURLs = append(detailURLs, fmt.Sprintf("%s/api/bazaar/%d", strings.TrimRight(baseURL, "/"), entry.AuctionID))
	}

	client := NewClient(opts)
	started := time.Now()
	bodies, err := fetchBatchJSONBodies(ctx, client, detailURLs)
	scrapeDuration.WithLabelValues("auctions").Observe(time.Since(started).Seconds())
	sources := append(append([]string{}, baseSources...), detailURLs...)
	if err != nil {
		scrapeRequests.WithLabelValues("auctions", "error").Inc()
		return domain.AuctionsDetailsResult{}, sources, err
	}
	scrapeRequests.WithLabelValues("auctions", "ok").Inc()

	parseStarted := time.Now()
	details := make([]domain.AuctionDetail, 0, len(bodies))
	for _, body := range bodies {
		var payload auctionDetailAPIResponse
		if parseErr := parseJSONBody(body, &payload); parseErr != nil {
			ParseErrors.WithLabelValues("auctions", "decode_error").Inc()
			return domain.AuctionsDetailsResult{}, sources, parseErr
		}
		details = append(details, mapAuctionDetailResponse(payload))
	}

	result := domain.AuctionsDetailsResult{
		Type:         auctions.Type,
		Page:         auctions.Page,
		TotalResults: auctions.TotalResults,
		TotalPages:   auctions.TotalPages,
		Entries:      details,
		Pagination:   auctions.Pagination,
	}
	parseDuration.WithLabelValues("auctions").Observe(time.Since(parseStarted).Seconds())
	ParseItems.WithLabelValues("auctions").Set(float64(len(result.Entries)))
	return result, sources, nil
}

func mapAuctionDetailResponse(payload auctionDetailAPIResponse) domain.AuctionDetail {
	return domain.AuctionDetail{
		AuctionID:         payload.Auction.ID,
		State:             payload.Auction.State,
		StateName:         strings.TrimSpace(payload.Auction.StateName),
		PlayerID:          payload.Auction.PlayerID,
		OwnerAccountID:    payload.Auction.Owner,
		CharacterName:     strings.TrimSpace(payload.Player.Name),
		Level:             payload.Player.Level,
		Vocation:          strings.TrimSpace(payload.Player.VocationName),
		VocationID:        payload.Player.Vocation,
		Sex:               sexName(payload.Player.Sex),
		World:             fallbackString(strings.TrimSpace(payload.Player.WorldName), worldNameByID(payload.Player.WorldID)),
		WorldID:           payload.Player.WorldID,
		BidType:           "bid",
		BidValue:          payload.Auction.CurrentValue,
		StartingValue:     payload.Auction.StartingValue,
		CurrentValue:      payload.Auction.CurrentValue,
		WinningBid:        payload.Auction.WinningBid,
		HighestBidderID:   payload.Auction.HighestBidderID,
		AuctionStart:      unixSecondsToRFC3339(payload.Auction.AuctionStart),
		AuctionEnd:        unixSecondsToRFC3339(payload.Auction.AuctionEnd),
		Status:            strings.TrimSpace(payload.Auction.StateName),
		CharmPoints:       payload.General.CharmPoints,
		AchievementPoints: payload.General.AchievementPoints,
		MagLevel:          payload.General.MagLevel,
		Skills:            toAuctionSkills(payload.General.Skills),
		Outfit: domain.AuctionOutfit{
			LookType:   payload.Player.LookType,
			LookHead:   payload.Player.LookHead,
			LookBody:   payload.Player.LookBody,
			LookLegs:   payload.Player.LookLegs,
			LookFeet:   payload.Player.LookFeet,
			LookAddons: payload.Player.LookAddons,
			LookMount:  payload.Player.LookMount,
		},
		HighlightItems:    toAuctionItems(payload.HighlightItems),
		HighlightAugments: toAuctionAugments(payload.HighlightAugments),
	}
}

func mapAuctionListEntry(row struct {
	ID                int    `json:"id"`
	State             int    `json:"state"`
	StateName         string `json:"stateName"`
	PlayerID          int    `json:"playerId"`
	Owner             int    `json:"owner"`
	StartingValue     int    `json:"startingValue"`
	CurrentValue      int    `json:"currentValue"`
	AuctionStart      int64  `json:"auctionStart"`
	AuctionEnd        int64  `json:"auctionEnd"`
	Name              string `json:"name"`
	Level             int    `json:"level"`
	Vocation          int    `json:"vocation"`
	VocationName      string `json:"vocationName"`
	Sex               int    `json:"sex"`
	WorldID           int    `json:"worldId"`
	WorldName         string `json:"worldName"`
	LookType          int    `json:"lookType"`
	LookHead          int    `json:"lookHead"`
	LookBody          int    `json:"lookBody"`
	LookLegs          int    `json:"lookLegs"`
	LookFeet          int    `json:"lookFeet"`
	LookAddons        int    `json:"lookAddons"`
	CharmPoints       int    `json:"charmPoints"`
	AchievementPoints int    `json:"achievementPoints"`
	MagLevel          int    `json:"magLevel"`
	Skills            struct {
		Axe       int `json:"axe"`
		Club      int `json:"club"`
		Sword     int `json:"sword"`
		Distance  int `json:"distance"`
		Dist      int `json:"dist"`
		Shielding int `json:"shielding"`
		Fishing   int `json:"fishing"`
		Fist      int `json:"fist"`
		Magic     int `json:"magic"`
	} `json:"skills"`
	HighlightItems []struct {
		ItemID int    `json:"itemId"`
		ID     int    `json:"id"`
		Name   string `json:"name"`
	} `json:"highlightItems"`
	HighlightAugments []struct {
		ArgType int    `json:"argType"`
		Text    string `json:"text"`
		Name    string `json:"name"`
	} `json:"highlightAugments"`
}) domain.AuctionEntry {
	return domain.AuctionEntry{
		AuctionID:         row.ID,
		State:             row.State,
		StateName:         strings.TrimSpace(row.StateName),
		PlayerID:          row.PlayerID,
		OwnerAccountID:    row.Owner,
		CharacterName:     strings.TrimSpace(row.Name),
		Level:             row.Level,
		Vocation:          strings.TrimSpace(row.VocationName),
		VocationID:        row.Vocation,
		Sex:               sexName(row.Sex),
		World:             fallbackString(strings.TrimSpace(row.WorldName), worldNameByID(row.WorldID)),
		WorldID:           row.WorldID,
		BidType:           "bid",
		BidValue:          row.CurrentValue,
		StartingValue:     row.StartingValue,
		CurrentValue:      row.CurrentValue,
		AuctionStart:      unixSecondsToRFC3339(row.AuctionStart),
		AuctionEnd:        unixSecondsToRFC3339(row.AuctionEnd),
		Status:            strings.TrimSpace(row.StateName),
		CharmPoints:       row.CharmPoints,
		AchievementPoints: row.AchievementPoints,
		MagLevel:          row.MagLevel,
		Skills:            toAuctionSkills(row.Skills),
		Outfit: domain.AuctionOutfit{
			LookType:   row.LookType,
			LookHead:   row.LookHead,
			LookBody:   row.LookBody,
			LookLegs:   row.LookLegs,
			LookFeet:   row.LookFeet,
			LookAddons: row.LookAddons,
		},
		HighlightItems:    toAuctionItems(row.HighlightItems),
		HighlightAugments: toAuctionAugments(row.HighlightAugments),
	}
}

func toAuctionSkills(skills struct {
	Axe       int `json:"axe"`
	Club      int `json:"club"`
	Sword     int `json:"sword"`
	Distance  int `json:"distance"`
	Dist      int `json:"dist"`
	Shielding int `json:"shielding"`
	Fishing   int `json:"fishing"`
	Fist      int `json:"fist"`
	Magic     int `json:"magic"`
}) domain.AuctionSkills {
	distance := skills.Distance
	if distance == 0 {
		distance = skills.Dist
	}
	return domain.AuctionSkills{
		Axe:       skills.Axe,
		Club:      skills.Club,
		Sword:     skills.Sword,
		Distance:  distance,
		Fishing:   skills.Fishing,
		Fist:      skills.Fist,
		Magic:     skills.Magic,
		Shielding: skills.Shielding,
	}
}

func toAuctionItems(rows []struct {
	ItemID int    `json:"itemId"`
	ID     int    `json:"id"`
	Name   string `json:"name"`
}) []domain.AuctionItem {
	items := make([]domain.AuctionItem, 0, len(rows))
	for _, row := range rows {
		id := row.ItemID
		if id == 0 {
			id = row.ID
		}
		items = append(items, domain.AuctionItem{ID: id, Name: strings.TrimSpace(row.Name)})
	}
	return items
}

func toAuctionAugments(rows []struct {
	ArgType int    `json:"argType"`
	Text    string `json:"text"`
	Name    string `json:"name"`
}) []domain.AuctionAugment {
	items := make([]domain.AuctionAugment, 0, len(rows))
	for _, row := range rows {
		name := strings.TrimSpace(row.Name)
		if name == "" {
			name = strings.TrimSpace(row.Text)
		}
		items = append(items, domain.AuctionAugment{ID: row.ArgType, Name: name})
	}
	return items
}

func sexName(raw int) string {
	if raw == 1 {
		return "Male"
	}
	if raw == 0 {
		return "Female"
	}
	return "Unknown"
}
