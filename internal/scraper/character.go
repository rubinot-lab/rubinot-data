package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/giovannirco/rubinot-data/internal/domain"
	"github.com/giovannirco/rubinot-data/internal/validation"
	"go.opentelemetry.io/otel/attribute"
)

type characterAPIResponse struct {
	Player *struct {
		ID             *int   `json:"id"`
		AccountID      *int   `json:"account_id"`
		Name           string `json:"name"`
		Level          int    `json:"level"`
		Vocation       string `json:"vocation"`
		VocationID     int    `json:"vocationId"`
		WorldID        int    `json:"world_id"`
		Sex            string `json:"sex"`
		Residence      string `json:"residence"`
		LastLogin      string `json:"lastlogin"`
		Created        int64  `json:"created"`
		Comment        string `json:"comment"`
		AccountCreated int64  `json:"account_created"`
		LoyaltyPoints  int    `json:"loyalty_points"`
		IsHidden       bool   `json:"isHidden"`
		Guild          *struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
			Rank string `json:"rank"`
			Nick string `json:"nick"`
		} `json:"guild"`
		House *struct {
			ID     int    `json:"id"`
			Name   string `json:"name"`
			TownID int    `json:"town_id"`
			Rent   int    `json:"rent"`
			Size   int    `json:"size"`
		} `json:"house"`
		Partner           json.RawMessage `json:"partner"`
		FormerNames       []string `json:"formerNames"`
		Title             *string  `json:"title"`
		Auction           any      `json:"auction"`
		LookType          int      `json:"looktype"`
		LookHead          int      `json:"lookhead"`
		LookBody          int      `json:"lookbody"`
		LookLegs          int      `json:"looklegs"`
		LookFeet          int      `json:"lookfeet"`
		LookAddons        int      `json:"lookaddons"`
		VIPTime           int64    `json:"vip_time"`
		AchievementPoints int      `json:"achievementPoints"`
	} `json:"player"`
	Deaths []struct {
		Time               string `json:"time"`
		Level              int    `json:"level"`
		KilledBy           string `json:"killed_by"`
		IsPlayer           int    `json:"is_player"`
		MostDamageBy       string `json:"mostdamage_by"`
		MostDamageIsPlayer int    `json:"mostdamage_is_player"`
	} `json:"deaths"`
	OtherCharacters []struct {
		Name     string `json:"name"`
		World    string `json:"world"`
		WorldID  int    `json:"world_id"`
		Level    int    `json:"level"`
		Vocation string `json:"vocation"`
		IsOnline bool   `json:"isOnline"`
	} `json:"otherCharacters"`
	AccountBadges []struct {
		ID       int    `json:"id"`
		ClientID int    `json:"clientId"`
		Name     string `json:"name"`
		URL      string `json:"url"`
	} `json:"accountBadges"`
	DisplayedAchievements []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"displayedAchievements"`
	BanInfo                      map[string]any `json:"banInfo"`
	CanSeeCharacterIdentifiers   bool           `json:"canSeeCharacterIdentifiers"`
	CanSeeDeathDetails           bool           `json:"canSeeDeathDetails"`
	FoundByOldName               bool           `json:"foundByOldName"`
}

func FetchCharacter(ctx context.Context, baseURL, characterName string, opts FetchOptions) (domain.CharacterResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchCharacter")
	defer span.End()

	query := url.Values{}
	query.Set("name", strings.TrimSpace(characterName))
	sourceURL := fmt.Sprintf("%s/api/characters/search?%s", strings.TrimRight(baseURL, "/"), query.Encode())
	client := NewClient(opts)

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "character"),
		attribute.String("rubinot.source_url", sourceURL),
		attribute.String("rubinot.character", characterName),
	)

	started := time.Now()
	var payload characterAPIResponse
	err := client.FetchJSON(ctx, sourceURL, &payload)
	scrapeDuration.WithLabelValues("character").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("character", "error").Inc()
		return domain.CharacterResult{}, sourceURL, err
	}
	if payload.Player == nil {
		scrapeRequests.WithLabelValues("character", "error").Inc()
		return domain.CharacterResult{}, sourceURL, validation.NewError(validation.ErrorEntityNotFound, "character not found", nil)
	}
	scrapeRequests.WithLabelValues("character", "ok").Inc()

	parseStarted := time.Now()
	result := mapCharacterResponse(payload)
	parseDuration.WithLabelValues("character").Observe(time.Since(parseStarted).Seconds())
	ParseItems.WithLabelValues("character").Set(float64(len(result.Deaths)))

	return result, sourceURL, nil
}

func mapCharacterResponse(payload characterAPIResponse) domain.CharacterResult {
	player := payload.Player
	worldName := worldNameByID(player.WorldID)
	if worldName == "" {
		worldName = "Unknown"
	}

	var guild *domain.CharacterGuild
	if player.Guild != nil {
		guild = &domain.CharacterGuild{
			ID:   player.Guild.ID,
			Name: strings.TrimSpace(player.Guild.Name),
			Rank: strings.TrimSpace(player.Guild.Rank),
			Nick: strings.TrimSpace(player.Guild.Nick),
		}
	}

	var house *domain.CharacterHouse
	if player.House != nil {
		house = &domain.CharacterHouse{
			HouseID: player.House.ID,
			Name:    strings.TrimSpace(player.House.Name),
			World:   worldName,
			TownID:  player.House.TownID,
			Rent:    player.House.Rent,
			Size:    player.House.Size,
		}
	}

	outfit := &domain.CharacterOutfit{
		LookType:   player.LookType,
		LookHead:   player.LookHead,
		LookBody:   player.LookBody,
		LookLegs:   player.LookLegs,
		LookFeet:   player.LookFeet,
		LookAddons: player.LookAddons,
	}

	deaths := make([]domain.CharacterDeath, 0, len(payload.Deaths))
	for _, row := range payload.Deaths {
		deaths = append(deaths, domain.CharacterDeath{
			Time:               unixTextToRFC3339(row.Time),
			Level:              row.Level,
			KilledBy:           strings.TrimSpace(row.KilledBy),
			IsPlayerKill:       row.IsPlayer == 1,
			MostDamageBy:       strings.TrimSpace(row.MostDamageBy),
			MostDamageIsPlayer: row.MostDamageIsPlayer == 1,
		})
	}

	others := make([]domain.OtherCharacter, 0, len(payload.OtherCharacters))
	for _, row := range payload.OtherCharacters {
		otherWorld := strings.TrimSpace(row.World)
		if otherWorld == "" {
			otherWorld = worldNameByID(row.WorldID)
		}
		others = append(others, domain.OtherCharacter{
			Name:     strings.TrimSpace(row.Name),
			World:    otherWorld,
			WorldID:  row.WorldID,
			Level:    row.Level,
			Vocation: strings.TrimSpace(row.Vocation),
			IsOnline: row.IsOnline,
		})
	}

	badges := make([]domain.CharacterBadge, 0, len(payload.AccountBadges))
	for _, row := range payload.AccountBadges {
		badges = append(badges, domain.CharacterBadge{
			ID:       row.ID,
			ClientID: row.ClientID,
			Name:     strings.TrimSpace(row.Name),
			URL:      strings.TrimSpace(row.URL),
		})
	}

	displayed := make([]domain.DisplayedAchievement, 0, len(payload.DisplayedAchievements))
	for _, row := range payload.DisplayedAchievements {
		displayed = append(displayed, domain.DisplayedAchievement{ID: row.ID, Name: strings.TrimSpace(row.Name)})
	}

	marriedTo := ""
	if len(player.Partner) > 0 && string(player.Partner) != "null" {
		var partnerObj struct {
			Name string `json:"name"`
		}
		if json.Unmarshal(player.Partner, &partnerObj) == nil && partnerObj.Name != "" {
			marriedTo = strings.TrimSpace(partnerObj.Name)
		} else {
			var partnerStr string
			if json.Unmarshal(player.Partner, &partnerStr) == nil {
				marriedTo = strings.TrimSpace(partnerStr)
			}
		}
	}
	title := ""
	if player.Title != nil {
		title = strings.TrimSpace(*player.Title)
	}

	info := domain.CharacterInfo{
		ID:                player.ID,
		AccountID:         player.AccountID,

		Name:              strings.TrimSpace(player.Name),
		FormerNames:       player.FormerNames,
		Sex:               strings.TrimSpace(player.Sex),
		Title:             title,
		Vocation:          strings.TrimSpace(player.Vocation),
		VocationID:        player.VocationID,
		Level:             player.Level,
		AchievementPoints: player.AchievementPoints,
		World:             worldName,
		WorldID:           player.WorldID,
		Residence:         strings.TrimSpace(player.Residence),
		MarriedTo:         marriedTo,
		House:             house,
		Guild:             guild,
		LastLogin:         unixTextToRFC3339(player.LastLogin),
		Comment:           strings.TrimSpace(player.Comment),
		IsBanned:          payload.BanInfo != nil,
		BanReason:         stringFromAny(payload.BanInfo["reason"]),
		LoyaltyPoints:     player.LoyaltyPoints,
		IsHidden:          player.IsHidden,
		Created:           unixSecondsToRFC3339(player.Created),
		AccountCreated:    unixSecondsToRFC3339(player.AccountCreated),
		VIPTime:           player.VIPTime,
		Outfit:            outfit,
		Auction:           player.Auction,
		FoundByOldName:    payload.FoundByOldName,
	}
	if house != nil {
		info.Houses = []domain.CharacterHouse{*house}
	}

	account := &domain.AccountInformation{
		Created:       unixSecondsToRFC3339(player.AccountCreated),
		LoyaltyPoints: player.LoyaltyPoints,
	}

	return domain.CharacterResult{
		CharacterInfo:                info,
		Deaths:                       deaths,
		AccountInfo:                  account,
		OtherCharacters:              others,
		AccountBadges:                badges,
		DisplayedAchievements:        displayed,
		CanSeeCharacterIdentifiers:   payload.CanSeeCharacterIdentifiers,
	}
}

func stringFromAny(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	}
}
