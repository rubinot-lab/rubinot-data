package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/giovannirco/rubinot-data/internal/scraper"
)

func TestInitOptimizedClientRubinidataSkipsCDP(t *testing.T) {
	rubinidataSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer rubinidataSrv.Close()

	t.Setenv("UPSTREAM_PROVIDER", "rubinidata")
	t.Setenv("RUBINIDATA_URL", rubinidataSrv.URL)
	t.Setenv("RUBINIDATA_API_KEY", "test-key")

	oc, err := initOptimizedClient(context.Background())
	if err != nil {
		t.Fatalf("initOptimizedClient failed: %v", err)
	}
	if oc == nil {
		t.Fatal("expected non-nil OptimizedClient")
	}
	if oc.Fetcher == nil {
		t.Fatal("expected non-nil Fetcher")
	}
	if !oc.Fetcher.IsReady() {
		t.Fatal("expected IsReady()=true for rubinidata mode")
	}
}

func TestInitOptimizedClientCDPRequiredWithoutRubinidata(t *testing.T) {
	t.Setenv("UPSTREAM_PROVIDER", "")
	t.Setenv("CDP_URL", "")

	_, err := initOptimizedClient(context.Background())
	if err == nil {
		t.Fatal("expected error when CDP_URL is not set and rubinidata is not active")
	}
}

func TestRouterRubinidataWorldMappingWiredAfterBootstrap(t *testing.T) {
	worldsResp := map[string]any{
		"worlds": map[string]any{
			"overview": map[string]any{
				"total_players_online": 100,
				"overall_maximum":      500,
				"maximum_date":         "",
			},
			"regular_worlds": []any{
				map[string]any{"id": 15, "name": "Belaria", "players_online": 50, "pvp_type": "pvp", "pvp_type_label": "Open PvP", "world_type": "regular", "locked": false},
				map[string]any{"id": 22, "name": "Serenian", "players_online": 30, "pvp_type": "pvp", "pvp_type_label": "Open PvP", "world_type": "regular", "locked": false},
			},
		},
	}
	worldsJSON, _ := json.Marshal(worldsResp)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v1/worlds" {
			w.Write(worldsJSON)
			return
		}
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	t.Setenv("UPSTREAM_PROVIDER", "rubinidata")
	t.Setenv("RUBINIDATA_URL", srv.URL)
	t.Setenv("RUBINIDATA_API_KEY", "key")
	t.Setenv("RUBINOT_BASE_URL", srv.URL)

	oc, err := initOptimizedClient(context.Background())
	if err != nil {
		t.Fatalf("initOptimizedClient failed: %v", err)
	}

	resolvedBaseURL = srv.URL
	validator, valErr := bootstrapValidatorV2(context.Background(), oc)
	if valErr != nil {
		t.Fatalf("bootstrapValidatorV2 failed: %v", valErr)
	}

	if scraper.IsRubinidataProvider() && oc.Fetcher != nil {
		oc.Fetcher.SetRubinidataWorldMapping(validator.WorldIDToName())
	}

	name, id, ok := validator.WorldExists("Belaria")
	if !ok || name != "Belaria" || id != 15 {
		t.Fatalf("expected Belaria world, got name=%q id=%d ok=%v", name, id, ok)
	}
}
