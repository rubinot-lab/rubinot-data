package scraper

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/giovannirco/rubinot-data/internal/validation"
)

func newTestOC(t *testing.T, responses map[string]string) *OptimizedClient {
	t.Helper()
	cdpServer := newMockCDPServer(t, func(path string) string {
		body, ok := responses[path]
		if !ok {
			return `{"error":"not found"}`
		}
		return body
	})
	t.Cleanup(cdpServer.Close)

	cdpURL := strings.Replace(cdpServer.URL, "http://", "ws://", 1)
	pool := NewCDPPool(cdpURL, "http://test.local", 1)
	if err := pool.Init(context.Background()); err != nil {
		t.Fatalf("pool init: %v", err)
	}
	t.Cleanup(pool.Close)

	fetcher := NewCachedFetcher(pool, 5*time.Minute)
	return NewOptimizedClient(fetcher)
}

func TestV2FetchWorlds(t *testing.T) {
	payload := worldsAPIResponse{
		Worlds: []struct {
			ID            int    `json:"id"`
			Name          string `json:"name"`
			PVPType       string `json:"pvpType"`
			PVPTypeLabel  string `json:"pvpTypeLabel"`
			WorldType     string `json:"worldType"`
			Locked        bool   `json:"locked"`
			PlayersOnline int    `json:"playersOnline"`
		}{
			{ID: 1, Name: "Elysian", PVPTypeLabel: "Open PvP", WorldType: "regular", PlayersOnline: 42},
		},
		TotalOnline:       42,
		OverallRecord:     100,
		OverallRecordTime: 1000,
	}
	body, _ := json.Marshal(payload)

	oc := newTestOC(t, map[string]string{
		"/api/worlds": string(body),
	})

	result, sourceURL, err := V2FetchWorlds(context.Background(), oc, "http://test.local")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sourceURL, "/api/worlds") {
		t.Errorf("expected sourceURL to contain /api/worlds, got %s", sourceURL)
	}
	if result.TotalPlayersOnline != 42 {
		t.Errorf("expected 42 players, got %d", result.TotalPlayersOnline)
	}
	if len(result.Worlds) != 1 {
		t.Fatalf("expected 1 world, got %d", len(result.Worlds))
	}
	if result.Worlds[0].Name != "Elysian" {
		t.Errorf("expected Elysian, got %s", result.Worlds[0].Name)
	}
}

func TestV2FetchWorld(t *testing.T) {
	payload := worldDetailAPIResponse{}
	payload.World.ID = 1
	payload.World.Name = "Elysian"
	payload.World.PVPTypeLabel = "Open PvP"
	payload.PlayersOnline = 10
	payload.Players = []struct {
		Name       string `json:"name"`
		Level      int    `json:"level"`
		Vocation   string `json:"vocation"`
		VocationID int    `json:"vocationId"`
	}{
		{Name: "TestPlayer", Level: 100, Vocation: "Knight", VocationID: 4},
	}
	body, _ := json.Marshal(payload)

	oc := newTestOC(t, map[string]string{
		"/api/worlds/Elysian": string(body),
	})

	result, sourceURL, err := V2FetchWorld(context.Background(), oc, "http://test.local", "Elysian")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sourceURL, "/api/worlds/Elysian") {
		t.Errorf("expected sourceURL to contain /api/worlds/Elysian, got %s", sourceURL)
	}
	if result.Name != "Elysian" {
		t.Errorf("expected Elysian, got %s", result.Name)
	}
	if len(result.PlayersOnline) != 1 {
		t.Fatalf("expected 1 player, got %d", len(result.PlayersOnline))
	}
}

func TestV2FetchCharacter(t *testing.T) {
	id := 1
	payload := characterAPIResponse{
		Player: &struct {
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
			FormerNames       []string        `json:"formerNames"`
			Title             *string         `json:"title"`
			Auction           any             `json:"auction"`
			LookType          int             `json:"looktype"`
			LookHead          int             `json:"lookhead"`
			LookBody          int             `json:"lookbody"`
			LookLegs          int             `json:"looklegs"`
			LookFeet          int             `json:"lookfeet"`
			LookAddons        int             `json:"lookaddons"`
			VIPTime           int64           `json:"vip_time"`
			AchievementPoints int             `json:"achievementPoints"`
		}{
			ID:    &id,
			Name:  "TestChar",
			Level: 200,
		},
	}
	body, _ := json.Marshal(payload)

	query := url.Values{}
	query.Set("name", "TestChar")
	path := "/api/characters/search?" + query.Encode()

	oc := newTestOC(t, map[string]string{
		path: string(body),
	})

	result, sourceURL, err := V2FetchCharacter(context.Background(), oc, "http://test.local", "TestChar")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sourceURL, "/api/characters/search") {
		t.Errorf("expected sourceURL to contain /api/characters/search, got %s", sourceURL)
	}
	if result.CharacterInfo.Name != "TestChar" {
		t.Errorf("expected TestChar, got %s", result.CharacterInfo.Name)
	}
}

func TestV2FetchCharacter_NotFound(t *testing.T) {
	payload := characterAPIResponse{Player: nil}
	body, _ := json.Marshal(payload)

	query := url.Values{}
	query.Set("name", "Nobody")
	path := "/api/characters/search?" + query.Encode()

	oc := newTestOC(t, map[string]string{
		path: string(body),
	})

	_, _, err := V2FetchCharacter(context.Background(), oc, "http://test.local", "Nobody")
	if err == nil {
		t.Fatal("expected error for not-found character")
	}
	valErr, ok := err.(validation.Error)
	if !ok {
		t.Fatalf("expected validation.Error, got %T", err)
	}
	if valErr.Code() != validation.ErrorEntityNotFound {
		t.Errorf("expected code %d, got %d", validation.ErrorEntityNotFound, valErr.Code())
	}
}

func TestV2FetchGuild(t *testing.T) {
	payload := guildAPIResponse{}
	payload.Guild.ID = 1
	payload.Guild.Name = "TestGuild"
	payload.Guild.WorldID = 1
	body, _ := json.Marshal(payload)

	oc := newTestOC(t, map[string]string{
		"/api/guilds/TestGuild": string(body),
	})

	result, sourceURL, err := V2FetchGuild(context.Background(), oc, "http://test.local", "TestGuild")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sourceURL, "/api/guilds/TestGuild") {
		t.Errorf("expected sourceURL to contain /api/guilds/TestGuild, got %s", sourceURL)
	}
	if result.ID != 1 {
		t.Errorf("expected guild ID 1, got %d", result.ID)
	}
}

func TestV2FetchGuilds(t *testing.T) {
	payload := guildsAPIResponse{
		Guilds: []struct {
			ID          int    `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			WorldID     int    `json:"world_id"`
			LogoName    string `json:"logo_name"`
		}{
			{ID: 1, Name: "Guild A", WorldID: 1},
		},
		TotalCount:  1,
		TotalPages:  1,
		CurrentPage: 1,
	}
	body, _ := json.Marshal(payload)

	q := url.Values{}
	q.Set("world", "1")
	q.Set("page", "1")
	path := "/api/guilds?" + q.Encode()

	oc := newTestOC(t, map[string]string{
		path: string(body),
	})

	result, sourceURL, err := V2FetchGuilds(context.Background(), oc, "http://test.local", "Elysian", 1, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sourceURL, "/api/guilds") {
		t.Errorf("expected sourceURL to contain /api/guilds, got %s", sourceURL)
	}
	if len(result.Guilds) != 1 {
		t.Fatalf("expected 1 guild, got %d", len(result.Guilds))
	}
	if result.World != "Elysian" {
		t.Errorf("expected Elysian, got %s", result.World)
	}
}

func TestV2FetchDeaths(t *testing.T) {
	payload := deathsAPIResponse{
		Deaths: []struct {
			PlayerID           int    `json:"player_id"`
			Time               string `json:"time"`
			Level              int    `json:"level"`
			KilledBy           string `json:"killed_by"`
			IsPlayer           int    `json:"is_player"`
			MostDamageBy       string `json:"mostdamage_by"`
			MostDamageIsPlayer int    `json:"mostdamage_is_player"`
			Victim             string `json:"victim"`
			WorldID            int    `json:"world_id"`
		}{
			{PlayerID: 1, Level: 100, KilledBy: "Dragon", Victim: "TestPlayer", WorldID: 1},
		},
		Pagination: struct {
			CurrentPage  int `json:"currentPage"`
			TotalPages   int `json:"totalPages"`
			TotalCount   int `json:"totalCount"`
			ItemsPerPage int `json:"itemsPerPage"`
		}{CurrentPage: 1, TotalPages: 1, TotalCount: 1, ItemsPerPage: 50},
	}
	body, _ := json.Marshal(payload)

	q := url.Values{}
	q.Set("world", "1")
	q.Set("page", "1")
	path := "/api/deaths?" + q.Encode()

	oc := newTestOC(t, map[string]string{
		path: string(body),
	})

	result, sourceURL, err := V2FetchDeaths(context.Background(), oc, "http://test.local", "Elysian", 1, DeathsFilters{Page: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sourceURL, "/api/deaths") {
		t.Errorf("expected sourceURL to contain /api/deaths, got %s", sourceURL)
	}
	if len(result.Entries) != 1 {
		t.Fatalf("expected 1 death, got %d", len(result.Entries))
	}
}

func TestV2FetchBanishments(t *testing.T) {
	payload := banishmentsAPIResponse{
		Bans: []struct {
			AccountID     int    `json:"account_id"`
			AccountName   string `json:"account_name"`
			MainCharacter string `json:"main_character"`
			Reason        string `json:"reason"`
			BannedAt      string `json:"banned_at"`
			ExpiresAt     string `json:"expires_at"`
			BannedBy      string `json:"banned_by"`
			IsPermanent   bool   `json:"is_permanent"`
		}{
			{AccountID: 1, MainCharacter: "BadPlayer", Reason: "bug abuse", IsPermanent: true},
		},
		TotalCount:  1,
		TotalPages:  1,
		CurrentPage: 1,
	}
	body, _ := json.Marshal(payload)

	q := url.Values{}
	q.Set("world", "1")
	q.Set("page", "1")
	path := "/api/bans?" + q.Encode()

	oc := newTestOC(t, map[string]string{
		path: string(body),
	})

	result, sourceURL, err := V2FetchBanishments(context.Background(), oc, "http://test.local", "Elysian", 1, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sourceURL, "/api/bans") {
		t.Errorf("expected sourceURL to contain /api/bans, got %s", sourceURL)
	}
	if len(result.Entries) != 1 {
		t.Fatalf("expected 1 ban, got %d", len(result.Entries))
	}
}

func TestV2FetchTransfers(t *testing.T) {
	payload := transfersAPIResponse{
		Transfers: []struct {
			ID            int         `json:"id"`
			PlayerID      int         `json:"player_id"`
			PlayerName    string      `json:"player_name"`
			PlayerLevel   int         `json:"player_level"`
			FromWorldID   int         `json:"from_world_id"`
			ToWorldID     int         `json:"to_world_id"`
			FromWorld     string      `json:"from_world"`
			ToWorld       string      `json:"to_world"`
			TransferredAt interface{} `json:"transferred_at"`
		}{
			{ID: 1, PlayerName: "Mover", FromWorld: "Elysian", ToWorld: "Lunarian"},
		},
		TotalResults: 1,
		TotalPages:   1,
		CurrentPage:  1,
	}
	body, _ := json.Marshal(payload)

	q := url.Values{}
	q.Set("page", "1")
	q.Set("world", "1")
	path := "/api/transfers?" + q.Encode()

	oc := newTestOC(t, map[string]string{
		path: string(body),
	})

	result, sourceURL, err := V2FetchTransfers(context.Background(), oc, "http://test.local", TransfersFilters{WorldID: 1, WorldName: "Elysian", Page: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sourceURL, "/api/transfers") {
		t.Errorf("expected sourceURL to contain /api/transfers, got %s", sourceURL)
	}
	if len(result.Entries) != 1 {
		t.Fatalf("expected 1 transfer, got %d", len(result.Entries))
	}
}

func TestV2FetchHighscores(t *testing.T) {
	payload := highscoresAPIResponse{
		Players: []struct {
			Rank      int         `json:"rank"`
			ID        int         `json:"id"`
			Name      string      `json:"name"`
			Level     int         `json:"level"`
			Vocation  int         `json:"vocation"`
			WorldID   int         `json:"world_id"`
			WorldName string      `json:"worldName"`
			Value     interface{} `json:"value"`
		}{
			{Rank: 1, ID: 1, Name: "TopPlayer", Level: 500, Vocation: 4, WorldID: 1, WorldName: "Elysian", Value: 1000000},
		},
		TotalCount: 1,
	}
	body, _ := json.Marshal(payload)

	q := url.Values{}
	q.Set("world", "1")
	q.Set("category", "experience")
	q.Set("vocation", "0")
	path := "/api/highscores?" + q.Encode()

	oc := newTestOC(t, map[string]string{
		path: string(body),
	})

	category := validation.HighscoreCategory{ID: 1, Name: "Experience", Slug: "experience"}
	vocation := validation.HighscoreVocation{Name: "(all)", ProfessionID: 0}

	result, sourceURL, err := V2FetchHighscores(context.Background(), oc, "http://test.local", "1", category, vocation)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sourceURL, "/api/highscores") {
		t.Errorf("expected sourceURL to contain /api/highscores, got %s", sourceURL)
	}
	if len(result.HighscoreList) != 1 {
		t.Fatalf("expected 1 highscore, got %d", len(result.HighscoreList))
	}
	if result.HighscorePage.TotalPages != 1 {
		t.Errorf("expected 1 total page, got %d", result.HighscorePage.TotalPages)
	}
}

func TestV2FetchKillstatistics(t *testing.T) {
	payload := killstatisticsAPIResponse{
		Entries: []struct {
			RaceName           string `json:"race_name"`
			PlayersKilled24h   int    `json:"players_killed_24h"`
			CreaturesKilled24h int    `json:"creatures_killed_24h"`
			PlayersKilled7d    int    `json:"players_killed_7d"`
			CreaturesKilled7d  int    `json:"creatures_killed_7d"`
		}{
			{RaceName: "Dragon", PlayersKilled24h: 5, CreaturesKilled24h: 100},
		},
	}
	body, _ := json.Marshal(payload)

	q := url.Values{}
	q.Set("world", "1")
	path := "/api/killstats?" + q.Encode()

	oc := newTestOC(t, map[string]string{
		path: string(body),
	})

	result, sourceURL, err := V2FetchKillstatistics(context.Background(), oc, "http://test.local", "Elysian", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sourceURL, "/api/killstats") {
		t.Errorf("expected sourceURL to contain /api/killstats, got %s", sourceURL)
	}
	if len(result.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result.Entries))
	}
}

func TestV2FetchBoosted(t *testing.T) {
	payload := boostedAPIResponse{}
	payload.Boss.ID = 1
	payload.Boss.Name = "TestBoss"
	payload.Monster.ID = 2
	payload.Monster.Name = "TestMonster"
	body, _ := json.Marshal(payload)

	oc := newTestOC(t, map[string]string{
		"/api/boosted": string(body),
	})

	result, sourceURL, err := V2FetchBoosted(context.Background(), oc, "http://test.local")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sourceURL, "/api/boosted") {
		t.Errorf("expected sourceURL to contain /api/boosted, got %s", sourceURL)
	}
	if result.Boss.Name != "TestBoss" {
		t.Errorf("expected TestBoss, got %s", result.Boss.Name)
	}
	if result.Monster.Name != "TestMonster" {
		t.Errorf("expected TestMonster, got %s", result.Monster.Name)
	}
}

func TestV2FetchMaintenance(t *testing.T) {
	payload := maintenanceAPIResponse{IsClosed: true, CloseMessage: "Server down"}
	body, _ := json.Marshal(payload)

	oc := newTestOC(t, map[string]string{
		"/api/maintenance": string(body),
	})

	result, sourceURL, err := V2FetchMaintenance(context.Background(), oc, "http://test.local")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sourceURL, "/api/maintenance") {
		t.Errorf("expected sourceURL to contain /api/maintenance, got %s", sourceURL)
	}
	if !result.IsClosed {
		t.Error("expected IsClosed true")
	}
}

func TestV2FetchCurrentAuctions(t *testing.T) {
	payload := auctionListAPIResponse{
		Auctions: []struct {
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
		}{
			{ID: 100, Name: "AuctionChar", Level: 300},
		},
		Pagination: struct {
			Page       int `json:"page"`
			Limit      int `json:"limit"`
			Total      int `json:"total"`
			TotalPages int `json:"totalPages"`
		}{Page: 1, Limit: 50, Total: 1, TotalPages: 1},
	}
	body, _ := json.Marshal(payload)

	q := url.Values{}
	q.Set("page", "1")
	q.Set("limit", "50")
	path := "/api/bazaar?" + q.Encode()

	oc := newTestOC(t, map[string]string{
		path: string(body),
	})

	result, sourceURL, err := V2FetchCurrentAuctions(context.Background(), oc, "http://test.local", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sourceURL, "/api/bazaar") {
		t.Errorf("expected sourceURL to contain /api/bazaar, got %s", sourceURL)
	}
	if len(result.Entries) != 1 {
		t.Fatalf("expected 1 auction, got %d", len(result.Entries))
	}
	if result.Type != "current" {
		t.Errorf("expected type current, got %s", result.Type)
	}
}

func TestV2FetchAuctionDetail(t *testing.T) {
	payload := auctionDetailAPIResponse{}
	payload.Auction.ID = 42
	payload.Auction.StateName = "finished"
	payload.Player.Name = "DetailChar"
	payload.Player.Level = 500
	body, _ := json.Marshal(payload)

	oc := newTestOC(t, map[string]string{
		"/api/bazaar/42": string(body),
	})

	result, sources, err := V2FetchAuctionDetail(context.Background(), oc, "http://test.local", 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}
	if result.AuctionID != 42 {
		t.Errorf("expected auction ID 42, got %d", result.AuctionID)
	}
}

func TestV2FetchNewsByID_Article(t *testing.T) {
	payload := newsAPIResponse{
		Articles: []struct {
			ID         int    `json:"id"`
			Title      string `json:"title"`
			Slug       string `json:"slug"`
			Summary    string `json:"summary"`
			Content    string `json:"content"`
			CoverImage string `json:"cover_image"`
			Author     string `json:"author"`
			Category   struct {
				ID      int    `json:"id"`
				Name    string `json:"name"`
				Slug    string `json:"slug"`
				Color   string `json:"color"`
				Icon    string `json:"icon"`
				IconURL string `json:"icon_url"`
			} `json:"category"`
			PublishedAt string `json:"published_at"`
		}{
			{ID: 99, Title: "Test Article", Content: "Some content"},
		},
	}
	body, _ := json.Marshal(payload)

	oc := newTestOC(t, map[string]string{
		"/api/news": string(body),
	})

	result, sources, err := V2FetchNewsByID(context.Background(), oc, "http://test.local", 99)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}
	if result.ID != 99 {
		t.Errorf("expected news ID 99, got %d", result.ID)
	}
	if result.Type != "article" {
		t.Errorf("expected type article, got %s", result.Type)
	}
}

func TestV2FetchNewsByID_NotFound(t *testing.T) {
	payload := newsAPIResponse{}
	body, _ := json.Marshal(payload)

	oc := newTestOC(t, map[string]string{
		"/api/news": string(body),
	})

	_, _, err := V2FetchNewsByID(context.Background(), oc, "http://test.local", 999)
	if err == nil {
		t.Fatal("expected error for not-found news")
	}
}

func TestV2FetchNewsArchive(t *testing.T) {
	payload := newsAPIResponse{
		Articles: []struct {
			ID         int    `json:"id"`
			Title      string `json:"title"`
			Slug       string `json:"slug"`
			Summary    string `json:"summary"`
			Content    string `json:"content"`
			CoverImage string `json:"cover_image"`
			Author     string `json:"author"`
			Category   struct {
				ID      int    `json:"id"`
				Name    string `json:"name"`
				Slug    string `json:"slug"`
				Color   string `json:"color"`
				Icon    string `json:"icon"`
				IconURL string `json:"icon_url"`
			} `json:"category"`
			PublishedAt string `json:"published_at"`
		}{
			{ID: 1, Title: "Recent Article", PublishedAt: "2026-03-01T00:00:00Z"},
		},
	}
	body, _ := json.Marshal(payload)

	oc := newTestOC(t, map[string]string{
		"/api/news": string(body),
	})

	result, sourceURL, err := V2FetchNewsArchive(context.Background(), oc, "http://test.local", 90)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sourceURL, "/api/news") {
		t.Errorf("expected sourceURL to contain /api/news, got %s", sourceURL)
	}
	if result.Mode != "archive" {
		t.Errorf("expected mode archive, got %s", result.Mode)
	}
}

func TestV2FetchNewsLatest(t *testing.T) {
	payload := newsAPIResponse{
		Articles: []struct {
			ID         int    `json:"id"`
			Title      string `json:"title"`
			Slug       string `json:"slug"`
			Summary    string `json:"summary"`
			Content    string `json:"content"`
			CoverImage string `json:"cover_image"`
			Author     string `json:"author"`
			Category   struct {
				ID      int    `json:"id"`
				Name    string `json:"name"`
				Slug    string `json:"slug"`
				Color   string `json:"color"`
				Icon    string `json:"icon"`
				IconURL string `json:"icon_url"`
			} `json:"category"`
			PublishedAt string `json:"published_at"`
		}{
			{ID: 1, Title: "Latest", PublishedAt: "2026-01-01T00:00:00Z"},
		},
	}
	body, _ := json.Marshal(payload)

	oc := newTestOC(t, map[string]string{
		"/api/news": string(body),
	})

	result, _, err := V2FetchNewsLatest(context.Background(), oc, "http://test.local")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Mode != "latest" {
		t.Errorf("expected mode latest, got %s", result.Mode)
	}
}

func TestV2FetchNewsTicker(t *testing.T) {
	payload := newsAPIResponse{
		Tickers: []struct {
			ID         int    `json:"id"`
			Message    string `json:"message"`
			CategoryID int    `json:"category_id"`
			Category   struct {
				ID      int    `json:"id"`
				Name    string `json:"name"`
				Slug    string `json:"slug"`
				Color   string `json:"color"`
				Icon    string `json:"icon"`
				IconURL string `json:"icon_url"`
			} `json:"category"`
			Author    string `json:"author"`
			CreatedAt string `json:"created_at"`
		}{
			{ID: 1, Message: "Ticker message", CreatedAt: "2026-01-01T00:00:00Z"},
		},
	}
	body, _ := json.Marshal(payload)

	oc := newTestOC(t, map[string]string{
		"/api/news": string(body),
	})

	result, _, err := V2FetchNewsTicker(context.Background(), oc, "http://test.local")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Mode != "newsticker" {
		t.Errorf("expected mode newsticker, got %s", result.Mode)
	}
}

func TestV2FetchWorldBatch(t *testing.T) {
	makeWorldBody := func(name string, id int) string {
		payload := worldDetailAPIResponse{}
		payload.World.ID = id
		payload.World.Name = name
		payload.PlayersOnline = 5
		b, _ := json.Marshal(payload)
		return string(b)
	}

	worlds := []validation.World{
		{ID: 1, Name: "Elysian"},
		{ID: 9, Name: "Lunarian"},
	}

	oc := newTestOC(t, map[string]string{
		"/api/worlds/Elysian":  makeWorldBody("Elysian", 1),
		"/api/worlds/Lunarian": makeWorldBody("Lunarian", 9),
	})

	results, sources, err := V2FetchWorldBatch(context.Background(), oc, "http://test.local", worlds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if len(sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(sources))
	}
}

func TestV2FetchWorldDashboard(t *testing.T) {
	worldPayload := worldDetailAPIResponse{}
	worldPayload.World.ID = 1
	worldPayload.World.Name = "Elysian"
	worldPayload.PlayersOnline = 5
	worldBody, _ := json.Marshal(worldPayload)

	deathsPayload := deathsAPIResponse{
		Pagination: struct {
			CurrentPage  int `json:"currentPage"`
			TotalPages   int `json:"totalPages"`
			TotalCount   int `json:"totalCount"`
			ItemsPerPage int `json:"itemsPerPage"`
		}{CurrentPage: 1, TotalPages: 1, TotalCount: 0, ItemsPerPage: 50},
	}
	deathsBody, _ := json.Marshal(deathsPayload)

	killstatsPayload := killstatisticsAPIResponse{}
	killstatsBody, _ := json.Marshal(killstatsPayload)

	oc := newTestOC(t, map[string]string{
		"/api/worlds/Elysian":        string(worldBody),
		"/api/deaths?page=1&world=1": string(deathsBody),
		"/api/killstats?world=1":     string(killstatsBody),
	})

	result, sources, err := V2FetchWorldDashboard(context.Background(), oc, "http://test.local", "Elysian", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sources) != 3 {
		t.Fatalf("expected 3 sources, got %d", len(sources))
	}
	if result.World.Name != "Elysian" {
		t.Errorf("expected Elysian, got %s", result.World.Name)
	}
}

func TestV2FetchKillstatisticsBatch(t *testing.T) {
	makeKillstatsBody := func() string {
		payload := killstatisticsAPIResponse{
			Entries: []struct {
				RaceName           string `json:"race_name"`
				PlayersKilled24h   int    `json:"players_killed_24h"`
				CreaturesKilled24h int    `json:"creatures_killed_24h"`
				PlayersKilled7d    int    `json:"players_killed_7d"`
				CreaturesKilled7d  int    `json:"creatures_killed_7d"`
			}{
				{RaceName: "Dragon"},
			},
		}
		b, _ := json.Marshal(payload)
		return string(b)
	}

	worlds := []validation.World{
		{ID: 1, Name: "Elysian"},
		{ID: 9, Name: "Lunarian"},
	}

	oc := newTestOC(t, map[string]string{
		"/api/killstats?world=1": makeKillstatsBody(),
		"/api/killstats?world=9": makeKillstatsBody(),
	})

	results, sources, err := V2FetchKillstatisticsBatch(context.Background(), oc, "http://test.local", worlds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if len(sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(sources))
	}
}

func TestV2FetchAllDeaths(t *testing.T) {
	payload := deathsAPIResponse{
		Deaths: []struct {
			PlayerID           int    `json:"player_id"`
			Time               string `json:"time"`
			Level              int    `json:"level"`
			KilledBy           string `json:"killed_by"`
			IsPlayer           int    `json:"is_player"`
			MostDamageBy       string `json:"mostdamage_by"`
			MostDamageIsPlayer int    `json:"mostdamage_is_player"`
			Victim             string `json:"victim"`
			WorldID            int    `json:"world_id"`
		}{
			{PlayerID: 1, Level: 100, KilledBy: "Dragon", Victim: "TestPlayer", WorldID: 1},
		},
		Pagination: struct {
			CurrentPage  int `json:"currentPage"`
			TotalPages   int `json:"totalPages"`
			TotalCount   int `json:"totalCount"`
			ItemsPerPage int `json:"itemsPerPage"`
		}{CurrentPage: 1, TotalPages: 1, TotalCount: 1, ItemsPerPage: 50},
	}
	body, _ := json.Marshal(payload)

	q := url.Values{}
	q.Set("world", "1")
	q.Set("page", "1")
	path := "/api/deaths?" + q.Encode()

	oc := newTestOC(t, map[string]string{
		path: string(body),
	})

	result, sources, err := V2FetchAllDeaths(context.Background(), oc, "http://test.local", "Elysian", 1, DeathsFilters{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sources) < 1 {
		t.Fatal("expected at least 1 source")
	}
	if len(result.Entries) != 1 {
		t.Fatalf("expected 1 death, got %d", len(result.Entries))
	}
}

func TestV2FetchAllBanishments(t *testing.T) {
	payload := banishmentsAPIResponse{
		Bans: []struct {
			AccountID     int    `json:"account_id"`
			AccountName   string `json:"account_name"`
			MainCharacter string `json:"main_character"`
			Reason        string `json:"reason"`
			BannedAt      string `json:"banned_at"`
			ExpiresAt     string `json:"expires_at"`
			BannedBy      string `json:"banned_by"`
			IsPermanent   bool   `json:"is_permanent"`
		}{
			{AccountID: 1, MainCharacter: "BadPlayer", Reason: "cheating"},
		},
		TotalCount:  1,
		TotalPages:  1,
		CurrentPage: 1,
	}
	body, _ := json.Marshal(payload)

	q := url.Values{}
	q.Set("world", "1")
	q.Set("page", "1")

	oc := newTestOC(t, map[string]string{
		"/api/bans?" + q.Encode(): string(body),
	})

	result, sources, err := V2FetchAllBanishments(context.Background(), oc, "http://test.local", "Elysian", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sources) < 1 {
		t.Fatal("expected at least 1 source")
	}
	if len(result.Entries) != 1 {
		t.Fatalf("expected 1 ban, got %d", len(result.Entries))
	}
}

func TestV2FetchAllTransfers(t *testing.T) {
	payload := transfersAPIResponse{
		Transfers: []struct {
			ID            int         `json:"id"`
			PlayerID      int         `json:"player_id"`
			PlayerName    string      `json:"player_name"`
			PlayerLevel   int         `json:"player_level"`
			FromWorldID   int         `json:"from_world_id"`
			ToWorldID     int         `json:"to_world_id"`
			FromWorld     string      `json:"from_world"`
			ToWorld       string      `json:"to_world"`
			TransferredAt interface{} `json:"transferred_at"`
		}{
			{ID: 1, PlayerName: "Mover", FromWorld: "A", ToWorld: "B"},
		},
		TotalResults: 1,
		TotalPages:   1,
		CurrentPage:  1,
	}
	body, _ := json.Marshal(payload)

	q := url.Values{}
	q.Set("page", "1")
	q.Set("world", "1")

	oc := newTestOC(t, map[string]string{
		"/api/transfers?" + q.Encode(): string(body),
	})

	result, sources, err := V2FetchAllTransfers(context.Background(), oc, "http://test.local", TransfersFilters{WorldID: 1, WorldName: "Elysian"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sources) < 1 {
		t.Fatal("expected at least 1 source")
	}
	if len(result.Entries) != 1 {
		t.Fatalf("expected 1 transfer, got %d", len(result.Entries))
	}
}

func TestV2FetchAllGuilds(t *testing.T) {
	payload := guildsAPIResponse{
		Guilds: []struct {
			ID          int    `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			WorldID     int    `json:"world_id"`
			LogoName    string `json:"logo_name"`
		}{
			{ID: 1, Name: "TestGuild", WorldID: 1},
		},
		TotalCount:  1,
		TotalPages:  1,
		CurrentPage: 1,
	}
	body, _ := json.Marshal(payload)

	q := url.Values{}
	q.Set("world", "1")
	q.Set("page", "1")

	oc := newTestOC(t, map[string]string{
		"/api/guilds?" + q.Encode(): string(body),
	})

	result, sources, err := V2FetchAllGuilds(context.Background(), oc, "http://test.local", "Elysian", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sources) < 1 {
		t.Fatal("expected at least 1 source")
	}
	if len(result.Guilds) != 1 {
		t.Fatalf("expected 1 guild, got %d", len(result.Guilds))
	}
}

func TestV2FetchAllCurrentAuctions(t *testing.T) {
	payload := auctionListAPIResponse{
		Auctions: []struct {
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
		}{
			{ID: 1, Name: "Auction1"},
		},
		Pagination: struct {
			Page       int `json:"page"`
			Limit      int `json:"limit"`
			Total      int `json:"total"`
			TotalPages int `json:"totalPages"`
		}{Page: 1, Limit: 50, Total: 1, TotalPages: 1},
	}
	body, _ := json.Marshal(payload)

	q := url.Values{}
	q.Set("page", "1")
	q.Set("limit", "50")

	oc := newTestOC(t, map[string]string{
		"/api/bazaar?" + q.Encode(): string(body),
	})

	result, sources, err := V2FetchAllCurrentAuctions(context.Background(), oc, "http://test.local")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sources) < 1 {
		t.Fatal("expected at least 1 source")
	}
	if result.Type != "current" {
		t.Errorf("expected type current, got %s", result.Type)
	}
}
