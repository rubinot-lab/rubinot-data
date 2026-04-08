package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestTranslateVocationID(t *testing.T) {
	tests := []struct {
		name       string
		rubinotID  string
		expectedID string
	}{
		{"all vocations", "0", "0"},
		{"none", "1", "0"},
		{"sorcerer", "2", "4"},
		{"druid", "3", "2"},
		{"paladin", "4", "3"},
		{"knight", "5", "1"},
		{"monk", "9", "5"},
		{"unknown passes through", "99", "99"},
		{"empty passes through", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := translateVocationID(tt.rubinotID)
			if got != tt.expectedID {
				t.Fatalf("translateVocationID(%q) = %q, want %q", tt.rubinotID, got, tt.expectedID)
			}
		})
	}
}

func TestTranslatePath(t *testing.T) {
	UpdateWorldMappings([]WorldMapping{
		{ID: 1, Name: "Elysian"},
		{ID: 11, Name: "Auroria"},
		{ID: 15, Name: "Belaria"},
	})

	tests := []struct {
		name        string
		upstreamURL string
		wantPath    string
		wantErr     bool
	}{
		{
			"worlds list",
			"https://rubinot.com.br/api/worlds",
			"/v1/worlds",
			false,
		},
		{
			"single world",
			"https://rubinot.com.br/api/worlds/Auroria",
			"/v1/world/Auroria",
			false,
		},
		{
			"character search",
			"https://rubinot.com.br/api/characters/search?name=Bubble",
			"/v1/characters/Bubble",
			false,
		},
		{
			"character search with spaces",
			"https://rubinot.com.br/api/characters/search?name=Bubble+Gum",
			"/v1/characters/Bubble%20Gum",
			false,
		},
		{
			"character search url encoded",
			"https://rubinot.com.br/api/characters/search?name=Bubble%20Gum",
			"/v1/characters/Bubble%20Gum",
			false,
		},
		{
			"guilds list with world and page",
			"https://rubinot.com.br/api/guilds?world=11&page=2",
			"/v1/guilds/Auroria?page=2",
			false,
		},
		{
			"guilds list with world only",
			"https://rubinot.com.br/api/guilds?world=15",
			"/v1/guilds/Belaria",
			false,
		},
		{
			"guilds list unknown world id passes through",
			"https://rubinot.com.br/api/guilds?world=999",
			"/v1/guilds/999",
			false,
		},
		{
			"single guild",
			"https://rubinot.com.br/api/guilds/Red Rose",
			"/v1/guild/Red%20Rose",
			false,
		},
		{
			"highscores categories",
			"https://rubinot.com.br/api/highscores/categories",
			"/v1/highscores/categories",
			false,
		},
		{
			"highscores with params",
			"https://rubinot.com.br/api/highscores?world=11&category=experience&vocation=2",
			"/v1/highscores?category=experience&vocation=4&world=Auroria",
			false,
		},
		{
			"highscores unknown world passes through",
			"https://rubinot.com.br/api/highscores?world=999&category=experience&vocation=5",
			"/v1/highscores?category=experience&vocation=1&world=999",
			false,
		},
		{
			"killstats",
			"https://rubinot.com.br/api/killstats?world=1",
			"/v1/killstatistics/Elysian",
			false,
		},
		{
			"killstats unknown world",
			"https://rubinot.com.br/api/killstats?world=999",
			"/v1/killstatistics/999",
			false,
		},
		{
			"deaths with world and page",
			"https://rubinot.com.br/api/deaths?world=11&page=3",
			"/v1/deaths/Auroria?page=3",
			false,
		},
		{
			"deaths with world only",
			"https://rubinot.com.br/api/deaths?world=11",
			"/v1/deaths/Auroria",
			false,
		},
		{
			"bans with world and page",
			"https://rubinot.com.br/api/bans?world=15&page=2",
			"/v1/banishments/Belaria?page=2",
			false,
		},
		{
			"bans with world only",
			"https://rubinot.com.br/api/bans?world=15",
			"/v1/banishments/Belaria",
			false,
		},
		{
			"transfers with page",
			"https://rubinot.com.br/api/transfers?page=5",
			"/v1/transfers?page=5",
			false,
		},
		{
			"transfers without page",
			"https://rubinot.com.br/api/transfers",
			"/v1/transfers",
			false,
		},
		{
			"boosted",
			"https://rubinot.com.br/api/boosted",
			"/v1/boosted",
			false,
		},
		{
			"unrecognized path errors",
			"https://rubinot.com.br/api/unknown/endpoint",
			"",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := translatePath(tt.upstreamURL)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("translatePath(%q) expected error, got nil", tt.upstreamURL)
				}
				return
			}
			if err != nil {
				t.Fatalf("translatePath(%q) unexpected error: %v", tt.upstreamURL, err)
			}
			if got != tt.wantPath {
				t.Fatalf("translatePath(%q) = %q, want %q", tt.upstreamURL, got, tt.wantPath)
			}
		})
	}
}

func TestTranslatePathEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		upstreamURL string
		wantErr     bool
	}{
		{"empty url", "", true},
		{"just path /api/worlds", "/api/worlds", false},
		{"character search missing name", "https://rubinot.com.br/api/characters/search", true},
		{"guilds missing world param", "https://rubinot.com.br/api/guilds", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := translatePath(tt.upstreamURL)
			if tt.wantErr && err == nil {
				t.Fatalf("translatePath(%q) expected error, got nil", tt.upstreamURL)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("translatePath(%q) unexpected error: %v", tt.upstreamURL, err)
			}
		})
	}
}

func TestIsRubinidataProvider(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		want     bool
	}{
		{"empty env", "", false},
		{"rubinot value", "rubinot", false},
		{"rubinidata lowercase", "rubinidata", true},
		{"rubinidata uppercase", "RUBINIDATA", true},
		{"rubinidata mixed case", "RubiniData", true},
		{"other value", "other", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("UPSTREAM_PROVIDER", tt.envValue)
			if got := IsRubinidataProvider(); got != tt.want {
				t.Fatalf("IsRubinidataProvider() = %v, want %v (env=%q)", got, tt.want, tt.envValue)
			}
		})
	}
}

func TestRubinidataClientFetchSuccess(t *testing.T) {
	UpdateWorldMappings([]WorldMapping{{ID: 11, Name: "Auroria"}})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "test-key" {
			t.Fatalf("expected X-API-Key header, got %q", r.Header.Get("X-API-Key"))
		}
		if r.URL.Path != "/v1/worlds" {
			t.Fatalf("expected /v1/worlds, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"worlds":[]}`)
	}))
	defer server.Close()

	client := NewRubinidataClient(server.URL, "test-key")
	body, err := client.Fetch(context.Background(), "https://rubinot.com.br/api/worlds")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body != `{"worlds":[]}` {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestRubinidataClientFetchSendsAPIKeyHeader(t *testing.T) {
	var receivedKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedKey = r.Header.Get("X-API-Key")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{}`)
	}))
	defer server.Close()

	client := NewRubinidataClient(server.URL, "my-secret-key")
	_, _ = client.Fetch(context.Background(), "https://rubinot.com.br/api/worlds")

	if receivedKey != "my-secret-key" {
		t.Fatalf("expected API key %q, got %q", "my-secret-key", receivedKey)
	}
}

func TestRubinidataClientFetchRetries(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "server error")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer server.Close()

	client := NewRubinidataClient(server.URL, "key")
	body, err := client.Fetch(context.Background(), "https://rubinot.com.br/api/worlds")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body != `{"ok":true}` {
		t.Fatalf("unexpected body: %q", body)
	}
	if attempts.Load() != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts.Load())
	}
}

func TestRubinidataClientFetchExhaustedRetries(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "server error")
	}))
	defer server.Close()

	client := NewRubinidataClient(server.URL, "key")
	_, err := client.Fetch(context.Background(), "https://rubinot.com.br/api/worlds")
	if err == nil {
		t.Fatal("expected error after exhausted retries")
	}
	if attempts.Load() != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts.Load())
	}
}

func TestRubinidataClientFetchInvalidPath(t *testing.T) {
	client := NewRubinidataClient("http://localhost", "key")
	_, err := client.Fetch(context.Background(), "https://rubinot.com.br/api/unknown/xyz")
	if err == nil {
		t.Fatal("expected error for unrecognized path")
	}
}

func TestRubinidataClientFetchCallsAdaptResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"raw":"data"}`)
	}))
	defer server.Close()

	client := NewRubinidataClient(server.URL, "key")
	body, err := client.Fetch(context.Background(), "https://rubinot.com.br/api/worlds")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body != `{"raw":"data"}` {
		t.Fatalf("expected adaptResponse to pass through body, got %q", body)
	}
}

func TestRubinidataClientFetchContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{}`)
	}))
	defer server.Close()

	client := NewRubinidataClient(server.URL, "key")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Fetch(ctx, "https://rubinot.com.br/api/worlds")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestAdaptWorldsResponse(t *testing.T) {
	input := `{"worlds":{"overview":{"total_players_online":12902,"overall_maximum":27884,"maximum_date":"Feb 08 2026, 17:50:57 BRT"},"regular_worlds":[{"name":"Elysian","players_online":1313,"pvp_type":"no-pvp","pvp_type_label":"Optional PvP","world_type":"yellow","locked":false,"id":0}]}}`

	result, err := adaptResponse("/api/worlds", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed worldsAPIResponse
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("failed to unmarshal adapted response: %v", err)
	}

	if parsed.TotalOnline != 12902 {
		t.Fatalf("expected totalOnline=12902, got %d", parsed.TotalOnline)
	}
	if parsed.OverallRecord != 27884 {
		t.Fatalf("expected overallRecord=27884, got %d", parsed.OverallRecord)
	}
	if parsed.OverallRecordTime <= 0 {
		t.Fatalf("expected overallRecordTime > 0, got %d", parsed.OverallRecordTime)
	}
	if len(parsed.Worlds) != 1 {
		t.Fatalf("expected 1 world, got %d", len(parsed.Worlds))
	}
	w := parsed.Worlds[0]
	if w.Name != "Elysian" {
		t.Fatalf("expected world name Elysian, got %q", w.Name)
	}
	if w.PlayersOnline != 1313 {
		t.Fatalf("expected playersOnline=1313, got %d", w.PlayersOnline)
	}
	if w.PVPType != "no-pvp" {
		t.Fatalf("expected pvpType=no-pvp, got %q", w.PVPType)
	}
	if w.PVPTypeLabel != "Optional PvP" {
		t.Fatalf("expected pvpTypeLabel=Optional PvP, got %q", w.PVPTypeLabel)
	}
	if w.WorldType != "yellow" {
		t.Fatalf("expected worldType=yellow, got %q", w.WorldType)
	}
}

func TestAdaptWorldDetailResponse(t *testing.T) {
	input := `{"world":{"name":"Elysian","status":"online","players_online":1339,"online_record":{"players":3816,"date":"May 30 2024, 19:54:36 BRT"},"creation_date":"Aug 17 2023","pvp_type":"no-pvp","pvp_type_label":"Optional PvP","world_type":"yellow","locked":false,"id":0},"players":[{"name":"Bubble","level":8,"vocation":"Knight","vocation_id":0}]}`

	result, err := adaptResponse("/api/worlds/Elysian", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed worldDetailAPIResponse
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("failed to unmarshal adapted response: %v", err)
	}

	if parsed.World.Name != "Elysian" {
		t.Fatalf("expected world name Elysian, got %q", parsed.World.Name)
	}
	if parsed.PlayersOnline != 1339 {
		t.Fatalf("expected playersOnline=1339, got %d", parsed.PlayersOnline)
	}
	if parsed.Record != 3816 {
		t.Fatalf("expected record=3816, got %d", parsed.Record)
	}
	if parsed.RecordTime <= 0 {
		t.Fatalf("expected recordTime > 0, got %d", parsed.RecordTime)
	}
	if parsed.World.CreationDate <= 0 {
		t.Fatalf("expected creationDate > 0, got %d", parsed.World.CreationDate)
	}
	if len(parsed.Players) != 1 {
		t.Fatalf("expected 1 player, got %d", len(parsed.Players))
	}
	if parsed.Players[0].Name != "Bubble" {
		t.Fatalf("expected player name Bubble, got %q", parsed.Players[0].Name)
	}
	if parsed.Players[0].Level != 8 {
		t.Fatalf("expected player level 8, got %d", parsed.Players[0].Level)
	}
	if parsed.Players[0].Vocation != "Knight" {
		t.Fatalf("expected player vocation Knight, got %q", parsed.Players[0].Vocation)
	}
}

func TestAdaptCharacterResponse(t *testing.T) {
	input := `{"characters":{"character":{"id":0,"name":"Icraozinho","traded":false,"level":1160,"vocation":"Exalted Monk","world_name":"Belaria","world_id":2,"sex":"male","achievement_points":255,"residence":"Thais","last_login":"2026-04-08T19:13:45-03:00","account_status":"Premium Account","house":"Main Street 9b","former_names":["Blindao Full Rage"],"found_by_old_name":true,"outfit_url":"/v1/outfit?type=128&head=79&body=49&legs=49&feet=79&addons=1&direction=3&animated=0&walk=0&size=0","loyalty_points":6,"created":"2025-06-14T18:48:57-03:00"},"other_characters":[{"id":0,"name":"Alt","level":23,"vocation":"Knight","world_id":1,"world_name":"Elysian"}]}}`

	result, err := adaptResponse("/api/characters/search?name=Icraozinho", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed characterAPIResponse
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("failed to unmarshal adapted response: %v", err)
	}

	if parsed.Player == nil {
		t.Fatal("expected player to be non-nil")
	}
	if parsed.Player.Name != "Icraozinho" {
		t.Fatalf("expected name Icraozinho, got %q", parsed.Player.Name)
	}
	if parsed.Player.Level != 1160 {
		t.Fatalf("expected level=1160, got %d", parsed.Player.Level)
	}
	if parsed.Player.Vocation != "Exalted Monk" {
		t.Fatalf("expected vocation Exalted Monk, got %q", parsed.Player.Vocation)
	}
	if parsed.Player.VocationID != 9 {
		t.Fatalf("expected vocationId=9, got %d", parsed.Player.VocationID)
	}
	if parsed.Player.WorldID != 2 {
		t.Fatalf("expected world_id=2, got %d", parsed.Player.WorldID)
	}
	if parsed.Player.LookType != 128 {
		t.Fatalf("expected looktype=128, got %d", parsed.Player.LookType)
	}
	if parsed.Player.LookHead != 79 {
		t.Fatalf("expected lookhead=79, got %d", parsed.Player.LookHead)
	}
	if parsed.Player.LookBody != 49 {
		t.Fatalf("expected lookbody=49, got %d", parsed.Player.LookBody)
	}
	if parsed.Player.LookAddons != 1 {
		t.Fatalf("expected lookaddons=1, got %d", parsed.Player.LookAddons)
	}
	if !parsed.FoundByOldName {
		t.Fatal("expected foundByOldName=true")
	}
	if len(parsed.Player.FormerNames) != 1 || parsed.Player.FormerNames[0] != "Blindao Full Rage" {
		t.Fatalf("expected formerNames=[Blindao Full Rage], got %v", parsed.Player.FormerNames)
	}
	if len(parsed.OtherCharacters) != 1 {
		t.Fatalf("expected 1 other character, got %d", len(parsed.OtherCharacters))
	}
	if parsed.OtherCharacters[0].World != "Elysian" {
		t.Fatalf("expected other character world Elysian, got %q", parsed.OtherCharacters[0].World)
	}
	if parsed.Deaths == nil {
		t.Fatal("expected deaths to be non-nil")
	}
	if len(parsed.Deaths) != 0 {
		t.Fatalf("expected 0 deaths, got %d", len(parsed.Deaths))
	}
}

func TestAdaptDeathsResponse(t *testing.T) {
	input := `{"deaths":{"entries":[{"name":"Player","level":62,"killers":[{"name":"monster","player":false},{"name":"monster2","player":false}],"time":"08.04.2026, 18:31:48","datetime":"2026-04-08T18:31:48-03:00","world_id":0,"player_id":0}]}}`

	result, err := adaptResponse("/api/deaths?world=0&page=1", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed deathsAPIResponse
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("failed to unmarshal adapted response: %v", err)
	}

	if len(parsed.Deaths) != 1 {
		t.Fatalf("expected 1 death, got %d", len(parsed.Deaths))
	}
	d := parsed.Deaths[0]
	if d.Victim != "Player" {
		t.Fatalf("expected victim=Player, got %q", d.Victim)
	}
	if d.Level != 62 {
		t.Fatalf("expected level=62, got %d", d.Level)
	}
	if d.KilledBy != "monster" {
		t.Fatalf("expected killed_by=monster, got %q", d.KilledBy)
	}
	if d.IsPlayer != 0 {
		t.Fatalf("expected is_player=0, got %d", d.IsPlayer)
	}
	if d.MostDamageBy != "monster2" {
		t.Fatalf("expected mostdamage_by=monster2, got %q", d.MostDamageBy)
	}
	if d.MostDamageIsPlayer != 0 {
		t.Fatalf("expected mostdamage_is_player=0, got %d", d.MostDamageIsPlayer)
	}
	if parsed.Pagination.CurrentPage != 1 {
		t.Fatalf("expected currentPage=1, got %d", parsed.Pagination.CurrentPage)
	}
	if parsed.Pagination.TotalCount != 1 {
		t.Fatalf("expected totalCount=1, got %d", parsed.Pagination.TotalCount)
	}
}

func TestAdaptHighscoresResponse(t *testing.T) {
	input := `{"highscores":{"category":"experience","world":"elysian","highscore_list":[{"rank":1,"id":0,"name":"Player","vocation":"Elite Knight","world_id":1,"world_name":"Elysian","level":2503,"value":260936620480}]}}`

	result, err := adaptResponse("/api/highscores?world=1&category=experience&vocation=5", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed highscoresAPIResponse
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("failed to unmarshal adapted response: %v", err)
	}

	if len(parsed.Players) != 1 {
		t.Fatalf("expected 1 player, got %d", len(parsed.Players))
	}
	p := parsed.Players[0]
	if p.Rank != 1 {
		t.Fatalf("expected rank=1, got %d", p.Rank)
	}
	if p.Name != "Player" {
		t.Fatalf("expected name=Player, got %q", p.Name)
	}
	if p.Vocation != 5 {
		t.Fatalf("expected vocation=5 (Elite Knight), got %d", p.Vocation)
	}
	if p.WorldName != "Elysian" {
		t.Fatalf("expected worldName=Elysian, got %q", p.WorldName)
	}
	if p.Level != 2503 {
		t.Fatalf("expected level=2503, got %d", p.Level)
	}
	if parsed.TotalCount != 1 {
		t.Fatalf("expected totalCount=1, got %d", parsed.TotalCount)
	}
	if parsed.CachedAt <= 0 {
		t.Fatalf("expected cachedAt > 0, got %d", parsed.CachedAt)
	}
}

func TestAdaptGuildsListResponse(t *testing.T) {
	input := `{"guilds":{"guilds":[{"name":"Guild","logo_url":"https://static.rubinot.com/guilds/guild_123.gif","description":"desc","id":0,"world_id":0}]}}`

	result, err := adaptResponse("/api/guilds?world=1", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed guildsAPIResponse
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("failed to unmarshal adapted response: %v", err)
	}

	if len(parsed.Guilds) != 1 {
		t.Fatalf("expected 1 guild, got %d", len(parsed.Guilds))
	}
	g := parsed.Guilds[0]
	if g.Name != "Guild" {
		t.Fatalf("expected name=Guild, got %q", g.Name)
	}
	if g.LogoName != "guild_123.gif" {
		t.Fatalf("expected logo_name=guild_123.gif, got %q", g.LogoName)
	}
	if g.Description != "desc" {
		t.Fatalf("expected description=desc, got %q", g.Description)
	}
	if parsed.TotalCount != 1 {
		t.Fatalf("expected totalCount=1, got %d", parsed.TotalCount)
	}
	if parsed.TotalPages != 1 {
		t.Fatalf("expected totalPages=1, got %d", parsed.TotalPages)
	}
	if parsed.CurrentPage != 1 {
		t.Fatalf("expected currentPage=1, got %d", parsed.CurrentPage)
	}
}

func TestAdaptGuildDetailResponse(t *testing.T) {
	input := `{"guild":{"name":"Name","world_id":0,"logo_url":"https://static.rubinot.com/guilds/guild_456.gif","description":"desc","founded":"Dec 29 2025","active":true,"guild_bank_balance":"0","members_total":2,"members_online":1,"members":[{"name":"Leader","title":"","rank":"Leader","vocation":"Elite Knight","level":1423,"joining_date":"Feb 10 2026","is_online":true,"id":0}]}}`

	result, err := adaptResponse("/api/guilds/Name", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed guildAPIResponse
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("failed to unmarshal adapted response: %v", err)
	}

	if parsed.Guild.Name != "Name" {
		t.Fatalf("expected guild name=Name, got %q", parsed.Guild.Name)
	}
	if parsed.Guild.Description != "desc" {
		t.Fatalf("expected description=desc, got %q", parsed.Guild.Description)
	}
	if parsed.Guild.LogoName != "guild_456.gif" {
		t.Fatalf("expected logo_name=guild_456.gif, got %q", parsed.Guild.LogoName)
	}
	if parsed.Guild.CreationData <= 0 {
		t.Fatalf("expected creationdata > 0, got %d", parsed.Guild.CreationData)
	}
	if parsed.Guild.Owner == nil {
		t.Fatal("expected owner to be non-nil")
	}
	if parsed.Guild.Owner.Name != "Leader" {
		t.Fatalf("expected owner name=Leader, got %q", parsed.Guild.Owner.Name)
	}
	if parsed.Guild.Owner.Vocation != 5 {
		t.Fatalf("expected owner vocation=5, got %d", parsed.Guild.Owner.Vocation)
	}
	if len(parsed.Guild.Members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(parsed.Guild.Members))
	}
	m := parsed.Guild.Members[0]
	if m.Name != "Leader" {
		t.Fatalf("expected member name=Leader, got %q", m.Name)
	}
	if m.Vocation != 5 {
		t.Fatalf("expected member vocation=5, got %d", m.Vocation)
	}
	if m.JoinDate <= 0 {
		t.Fatalf("expected joinDate > 0, got %d", m.JoinDate)
	}
	if !m.IsOnline {
		t.Fatal("expected member isOnline=true")
	}
	if len(parsed.Guild.Ranks) != 1 {
		t.Fatalf("expected 1 rank, got %d", len(parsed.Guild.Ranks))
	}
}

func TestAdaptBanishmentsResponse(t *testing.T) {
	input := `{"banishments":{"entries":[{"account_id":0,"account_name":"Acc","character":"Char","reason":"Cheating","banned_at":"1712580000","expires_at":"1712666400","banned_by":"GM","is_permanent":false}]}}`

	result, err := adaptResponse("/api/bans?world=1&page=1", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed banishmentsAPIResponse
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("failed to unmarshal adapted response: %v", err)
	}

	if len(parsed.Bans) != 1 {
		t.Fatalf("expected 1 ban, got %d", len(parsed.Bans))
	}
	b := parsed.Bans[0]
	if b.AccountName != "Acc" {
		t.Fatalf("expected account_name=Acc, got %q", b.AccountName)
	}
	if b.MainCharacter != "Char" {
		t.Fatalf("expected main_character=Char, got %q", b.MainCharacter)
	}
	if b.Reason != "Cheating" {
		t.Fatalf("expected reason=Cheating, got %q", b.Reason)
	}
	if b.BannedBy != "GM" {
		t.Fatalf("expected banned_by=GM, got %q", b.BannedBy)
	}
	if parsed.TotalCount != 1 {
		t.Fatalf("expected totalCount=1, got %d", parsed.TotalCount)
	}
	if parsed.TotalPages != 1 {
		t.Fatalf("expected totalPages=1, got %d", parsed.TotalPages)
	}
}

func TestAdaptTransfersResponse(t *testing.T) {
	input := `{"transfers":{"entries":[{"name":"Player","level":100,"from_world":"Elysian","to_world":"Bellum","transfer_date":"2026-04-08T15:30:21-03:00"}]}}`

	result, err := adaptResponse("/api/transfers?page=1", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed transfersAPIResponse
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("failed to unmarshal adapted response: %v", err)
	}

	if len(parsed.Transfers) != 1 {
		t.Fatalf("expected 1 transfer, got %d", len(parsed.Transfers))
	}
	tr := parsed.Transfers[0]
	if tr.PlayerName != "Player" {
		t.Fatalf("expected player_name=Player, got %q", tr.PlayerName)
	}
	if tr.PlayerLevel != 100 {
		t.Fatalf("expected player_level=100, got %d", tr.PlayerLevel)
	}
	if tr.FromWorld != "Elysian" {
		t.Fatalf("expected from_world=Elysian, got %q", tr.FromWorld)
	}
	if tr.ToWorld != "Bellum" {
		t.Fatalf("expected to_world=Bellum, got %q", tr.ToWorld)
	}
	if parsed.TotalResults != 1 {
		t.Fatalf("expected totalResults=1, got %d", parsed.TotalResults)
	}
	if parsed.TotalPages != 1 {
		t.Fatalf("expected totalPages=1, got %d", parsed.TotalPages)
	}
	if parsed.CurrentPage != 1 {
		t.Fatalf("expected currentPage=1, got %d", parsed.CurrentPage)
	}
}

func TestAdaptBoostedResponse(t *testing.T) {
	input := `{"boosted":{"creature":{"name":"Dragon","id":1,"looktype":932,"image_url":"https://example.com/dragon.gif"},"boss":{"name":"Ferumbras","id":2,"looktype":1589,"image_url":"https://example.com/ferumbras.gif"}}}`

	result, err := adaptResponse("/api/boosted", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed boostedAPIResponse
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("failed to unmarshal adapted response: %v", err)
	}

	if parsed.Monster.Name != "Dragon" {
		t.Fatalf("expected monster name=Dragon, got %q", parsed.Monster.Name)
	}
	if parsed.Monster.ID != 1 {
		t.Fatalf("expected monster id=1, got %d", parsed.Monster.ID)
	}
	if parsed.Monster.LookType != 932 {
		t.Fatalf("expected monster looktype=932, got %d", parsed.Monster.LookType)
	}
	if parsed.Boss.Name != "Ferumbras" {
		t.Fatalf("expected boss name=Ferumbras, got %q", parsed.Boss.Name)
	}
	if parsed.Boss.ID != 2 {
		t.Fatalf("expected boss id=2, got %d", parsed.Boss.ID)
	}
	if parsed.Boss.LookType != 1589 {
		t.Fatalf("expected boss looktype=1589, got %d", parsed.Boss.LookType)
	}
}

func TestAdaptKillstatisticsPassthrough(t *testing.T) {
	input := `{"entries":[{"race_name":"Dragon","players_killed_24h":5,"creatures_killed_24h":100,"players_killed_7d":30,"creatures_killed_7d":700}],"totals":{"players_killed_24h":5,"creatures_killed_24h":100,"players_killed_7d":30,"creatures_killed_7d":700}}`

	result, err := adaptResponse("/api/killstats?world=1", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed killstatisticsAPIResponse
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("failed to unmarshal adapted response: %v", err)
	}

	if len(parsed.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(parsed.Entries))
	}
	if parsed.Entries[0].RaceName != "Dragon" {
		t.Fatalf("expected race_name=Dragon, got %q", parsed.Entries[0].RaceName)
	}
	if parsed.Totals.PlayersKilled24h != 5 {
		t.Fatalf("expected totals.players_killed_24h=5, got %d", parsed.Totals.PlayersKilled24h)
	}
}

func TestAdaptResponseUnknownPathPassthrough(t *testing.T) {
	input := `{"some":"data"}`
	result, err := adaptResponse("/api/unknown/endpoint", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != input {
		t.Fatalf("expected passthrough, got %q", result)
	}
}

func TestAdaptDeathsWithPlayerKillers(t *testing.T) {
	input := `{"deaths":{"entries":[{"name":"Victim","level":100,"killers":[{"name":"PlayerKiller","player":true},{"name":"monster","player":false}],"time":"08.04.2026, 18:31:48","datetime":"2026-04-08T18:31:48-03:00","world_id":0,"player_id":0}]}}`

	result, err := adaptResponse("/api/deaths?world=0&page=1", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed deathsAPIResponse
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("failed to unmarshal adapted response: %v", err)
	}

	d := parsed.Deaths[0]
	if d.KilledBy != "PlayerKiller" {
		t.Fatalf("expected killed_by=PlayerKiller, got %q", d.KilledBy)
	}
	if d.IsPlayer != 1 {
		t.Fatalf("expected is_player=1, got %d", d.IsPlayer)
	}
	if d.MostDamageBy != "monster" {
		t.Fatalf("expected mostdamage_by=monster, got %q", d.MostDamageBy)
	}
	if d.MostDamageIsPlayer != 0 {
		t.Fatalf("expected mostdamage_is_player=0, got %d", d.MostDamageIsPlayer)
	}
}

func TestAdaptHighscoresVocationMapping(t *testing.T) {
	tests := []struct {
		vocation string
		expected int
	}{
		{"Knight", 5},
		{"Elite Knight", 5},
		{"Paladin", 4},
		{"Royal Paladin", 4},
		{"Sorcerer", 2},
		{"Master Sorcerer", 2},
		{"Druid", 3},
		{"Elder Druid", 3},
		{"Monk", 9},
		{"Exalted Monk", 9},
		{"None", 1},
	}

	for _, tt := range tests {
		t.Run(tt.vocation, func(t *testing.T) {
			got := vocationNameToUpstreamID(tt.vocation)
			if got != tt.expected {
				t.Fatalf("vocationNameToUpstreamID(%q) = %d, want %d", tt.vocation, got, tt.expected)
			}
		})
	}
}

func TestParseBRTDate(t *testing.T) {
	ts := parseBRTDate("Feb 08 2026, 17:50:57 BRT")
	if ts <= 0 {
		t.Fatalf("expected positive timestamp, got %d", ts)
	}
	expected := time.Date(2026, 2, 8, 17, 50, 57, 0, brtLocation).Unix()
	if ts != expected {
		t.Fatalf("expected %d, got %d", expected, ts)
	}
}

func TestParseSimpleDate(t *testing.T) {
	ts := parseSimpleDate("Aug 17 2023")
	if ts <= 0 {
		t.Fatalf("expected positive timestamp, got %d", ts)
	}
	expected := time.Date(2023, 8, 17, 0, 0, 0, 0, time.UTC).Unix()
	if ts != expected {
		t.Fatalf("expected %d, got %d", expected, ts)
	}
}

func TestParseOutfitURL(t *testing.T) {
	lt, h, b, l, f, a := parseOutfitURL("/v1/outfit?type=128&head=79&body=49&legs=49&feet=79&addons=1&direction=3&animated=0&walk=0&size=0")
	if lt != 128 {
		t.Fatalf("expected looktype=128, got %d", lt)
	}
	if h != 79 {
		t.Fatalf("expected head=79, got %d", h)
	}
	if b != 49 {
		t.Fatalf("expected body=49, got %d", b)
	}
	if l != 49 {
		t.Fatalf("expected legs=49, got %d", l)
	}
	if f != 79 {
		t.Fatalf("expected feet=79, got %d", f)
	}
	if a != 1 {
		t.Fatalf("expected addons=1, got %d", a)
	}
}

func TestExtractFilename(t *testing.T) {
	got := extractFilename("https://static.rubinot.com/guilds/guild_123.gif")
	if got != "guild_123.gif" {
		t.Fatalf("expected guild_123.gif, got %q", got)
	}
}

func TestRubinidataClientFetchNotFoundReturnsBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error":"not found"}`)
	}))
	defer server.Close()

	client := NewRubinidataClient(server.URL, "key")
	body, err := client.Fetch(context.Background(), "https://rubinot.com.br/api/worlds")
	if err != nil {
		t.Fatalf("unexpected error for 404: %v", err)
	}
	if body != `{"error":"not found"}` {
		t.Fatalf("expected 404 body to be returned, got %q", body)
	}
}
