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

func TestRubinidataClientBatchFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/worlds":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"worlds":{"overview":{"total_players_online":0,"overall_maximum":0,"maximum_date":""},"regular_worlds":[]}}`))
		case "/v1/boosted":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"boosted":{"creature":{"name":"Rat","id":1,"looktype":21,"image_url":""},"boss":{"name":"Boss","id":2,"looktype":22,"image_url":""}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := NewRubinidataClient(srv.URL, "key")
	paths := []string{"/api/worlds", "/api/boosted"}
	results, err := client.BatchFetch(context.Background(), paths)
	if err != nil {
		t.Fatalf("BatchFetch failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if _, ok := results["/api/worlds"]; !ok {
		t.Fatal("missing /api/worlds in results")
	}
	if _, ok := results["/api/boosted"]; !ok {
		t.Fatal("missing /api/boosted in results")
	}
}

func TestRubinidataClientBatchFetchPartialError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/worlds" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"worlds":{"overview":{"total_players_online":0,"overall_maximum":0,"maximum_date":""},"regular_worlds":[]}}`))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("fail"))
	}))
	defer srv.Close()

	client := NewRubinidataClient(srv.URL, "key")
	_, err := client.BatchFetch(context.Background(), []string{"/api/worlds", "/api/unknown"})
	if err == nil {
		t.Fatal("expected error for batch with failing path, got nil")
	}
}

func TestRubinidataClientFetchBinary(t *testing.T) {
	pngHeader := []byte{0x89, 0x50, 0x4E, 0x47}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/outfit" {
			t.Fatalf("expected /v1/outfit, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("type") != "128" {
			t.Fatalf("expected type=128, got %s", r.URL.Query().Get("type"))
		}
		w.Header().Set("Content-Type", "image/png")
		w.Write(pngHeader)
	}))
	defer srv.Close()

	client := NewRubinidataClient(srv.URL, "key")
	body, ct, err := client.FetchBinary(context.Background(), "/api/outfit?type=128")
	if err != nil {
		t.Fatalf("FetchBinary failed: %v", err)
	}
	if ct != "image/png" {
		t.Fatalf("expected content-type image/png, got %q", ct)
	}
	if len(body) != 4 {
		t.Fatalf("expected 4 bytes, got %d", len(body))
	}
}

func TestCachedFetcherRubinidataBatchFetchJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/worlds":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"worlds":{"overview":{"total_players_online":0,"overall_maximum":0,"maximum_date":""},"regular_worlds":[]}}`))
		case "/v1/boosted":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"boosted":{"creature":{"name":"Rat","id":1,"looktype":21,"image_url":""},"boss":{"name":"Boss","id":2,"looktype":22,"image_url":""}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := NewRubinidataClient(srv.URL, "key")
	fetcher := NewCachedFetcher(nil, 5*time.Second)
	fetcher.rubinidata = client

	urls := []string{
		"https://rubinot.com.br/api/worlds",
		"https://rubinot.com.br/api/boosted",
	}
	results, err := fetcher.BatchFetchJSON(context.Background(), urls)
	if err != nil {
		t.Fatalf("BatchFetchJSON failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestCachedFetcherRubinidataFetchBinary(t *testing.T) {
	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(pngData)
	}))
	defer srv.Close()

	client := NewRubinidataClient(srv.URL, "key")
	fetcher := NewCachedFetcher(nil, 5*time.Second)
	fetcher.rubinidata = client

	body, ct, err := fetcher.FetchBinary(context.Background(), "/api/outfit?type=128")
	if err != nil {
		t.Fatalf("FetchBinary failed: %v", err)
	}
	if ct != "image/png" {
		t.Fatalf("expected content-type image/png, got %q", ct)
	}
	if len(body) != len(pngData) {
		t.Fatalf("expected %d bytes, got %d", len(pngData), len(body))
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
