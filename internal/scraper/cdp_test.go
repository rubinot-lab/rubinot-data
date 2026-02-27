package scraper

import (
	"context"
	"testing"
)

func TestCDPClientBatchFetch(t *testing.T) {
	cdpSrv := newMockCDPServer(t, func(path string) string {
		switch path {
		case "/api/worlds":
			return `{"worlds":[{"id":15,"name":"Belaria"}]}`
		case "/api/deaths?world=15&page=1":
			return `{"deaths":[]}`
		default:
			return `{}`
		}
	})
	defer cdpSrv.Close()

	client := NewCDPClient(cdpSrv.URL)
	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("connect cdp: %v", err)
	}
	defer client.Close()

	results, err := client.BatchFetch(context.Background(), []string{
		"/api/worlds",
		"/api/deaths?world=15&page=1",
	})
	if err != nil {
		t.Fatalf("batch fetch: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Status != "fulfilled" || results[1].Status != "fulfilled" {
		t.Fatalf("unexpected statuses: %+v", results)
	}
	if results[0].Value == "" || results[1].Value == "" {
		t.Fatalf("expected non-empty values: %+v", results)
	}
}

func TestCDPClientBatchFetchPartialFailure(t *testing.T) {
	cdpSrv := newMockCDPServer(t, func(path string) string {
		if path == "/api/characters/search?name=Failing" {
			return "__REJECT__:net::ERR_BLOCKED_BY_CLIENT"
		}
		return `{"ok":true}`
	})
	defer cdpSrv.Close()

	client := NewCDPClient(cdpSrv.URL)
	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("connect cdp: %v", err)
	}
	defer client.Close()

	results, err := client.BatchFetch(context.Background(), []string{
		"/api/characters/search?name=Working",
		"/api/characters/search?name=Failing",
	})
	if err != nil {
		t.Fatalf("batch fetch: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Status != "fulfilled" {
		t.Fatalf("expected first result fulfilled, got %+v", results[0])
	}
	if results[1].Status != "rejected" {
		t.Fatalf("expected second result rejected, got %+v", results[1])
	}
	if results[1].Value != "net::ERR_BLOCKED_BY_CLIENT" {
		t.Fatalf("unexpected rejected value: %+v", results[1])
	}
}

func TestCDPClientBatchFetchEmptyPaths(t *testing.T) {
	client := NewCDPClient("ws://localhost:9222")

	results, err := client.BatchFetch(context.Background(), nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected empty result set, got %d", len(results))
	}
}
