package scraper

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCachedFetcherRubinidataFetchJSON(t *testing.T) {
	worldsResp := map[string]any{
		"worlds": map[string]any{
			"overview": map[string]any{
				"total_players_online": 100,
				"overall_maximum":      500,
				"maximum_date":         "",
			},
			"regular_worlds": []map[string]any{
				{"id": 1, "name": "Elysian", "players_online": 50, "pvp_type": "pvp", "pvp_type_label": "Open PvP", "world_type": "regular", "locked": false},
			},
		},
	}
	worldsJSON, _ := json.Marshal(worldsResp)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/worlds" {
			w.Header().Set("Content-Type", "application/json")
			w.Write(worldsJSON)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	client := NewRubinidataClient(srv.URL, "test-key")
	fetcher := NewCachedFetcher(nil, 5*time.Second)
	fetcher.rubinidata = client

	body, err := fetcher.FetchJSON(context.Background(), "https://rubinot.com.br/api/worlds")
	if err != nil {
		t.Fatalf("FetchJSON failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	worlds, ok := result["worlds"]
	if !ok {
		t.Fatal("expected 'worlds' key in response")
	}
	worldSlice, ok := worlds.([]any)
	if !ok {
		t.Fatal("expected worlds to be an array")
	}
	if len(worldSlice) != 1 {
		t.Fatalf("expected 1 world, got %d", len(worldSlice))
	}
}

func TestCachedFetcherRubinidataCachesResult(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"worlds":{"overview":{"total_players_online":0,"overall_maximum":0,"maximum_date":""},"regular_worlds":[]}}`))
	}))
	defer srv.Close()

	client := NewRubinidataClient(srv.URL, "test-key")
	fetcher := NewCachedFetcher(nil, 5*time.Second)
	fetcher.rubinidata = client

	ctx := context.Background()
	_, err := fetcher.FetchJSON(ctx, "https://rubinot.com.br/api/worlds")
	if err != nil {
		t.Fatalf("first FetchJSON failed: %v", err)
	}

	_, err = fetcher.FetchJSON(ctx, "https://rubinot.com.br/api/worlds")
	if err != nil {
		t.Fatalf("second FetchJSON failed: %v", err)
	}

	if callCount != 1 {
		t.Fatalf("expected 1 upstream call (cache hit on second), got %d", callCount)
	}
}

func TestCachedFetcherRubinidataIsReady(t *testing.T) {
	fetcher := NewCachedFetcher(nil, 5*time.Second)
	fetcher.rubinidata = NewRubinidataClient("http://localhost", "key")

	if !fetcher.IsReady() {
		t.Fatal("expected IsReady()=true when rubinidata client is set")
	}
}

func TestCachedFetcherIsReadyFalseWithoutRubinidata(t *testing.T) {
	fetcher := NewCachedFetcher(nil, 5*time.Second)
	fetcher.cfBlocked.Store(true)

	if fetcher.IsReady() {
		t.Fatal("expected IsReady()=false when cfBlocked and no rubinidata")
	}
}

func TestCachedFetcherSetRubinidataWorldMapping(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	worldMu.RLock()
	savedIDToName := make(map[int]string, len(worldIDToName))
	for k, v := range worldIDToName {
		savedIDToName[k] = v
	}
	savedNameToID := make(map[string]int, len(worldNameToID))
	for k, v := range worldNameToID {
		savedNameToID[k] = v
	}
	worldMu.RUnlock()
	defer func() {
		worldMu.Lock()
		worldIDToName = savedIDToName
		worldNameToID = savedNameToID
		worldMu.Unlock()
	}()

	client := NewRubinidataClient(srv.URL, "key")
	fetcher := NewCachedFetcher(nil, 5*time.Second)
	fetcher.rubinidata = client

	mapping := map[int]string{1: "Elysian", 11: "Auroria"}
	fetcher.SetRubinidataWorldMapping(mapping)

	name := worldNameByID(1)
	if name != "Elysian" {
		t.Fatalf("expected worldNameByID(1)=Elysian, got %q", name)
	}
	name = worldNameByID(11)
	if name != "Auroria" {
		t.Fatalf("expected worldNameByID(11)=Auroria, got %q", name)
	}
}
