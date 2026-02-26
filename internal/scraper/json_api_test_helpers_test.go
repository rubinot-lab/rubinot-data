package scraper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type fsRequestPayload struct {
	URL string `json:"url"`
}

func newFlareSolverrJSONServer(t *testing.T, contentForURL func(string) string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected flaresolverr method: %s", r.Method)
		}

		var payload fsRequestPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode flaresolverr request: %v", err)
		}

		targetURL := strings.TrimSpace(payload.URL)
		body := "<html><body>ok</body></html>"
		if contentForURL != nil {
			if custom := contentForURL(targetURL); custom != "" {
				body = custom
			}
		}

		resp := map[string]any{
			"status":  "ok",
			"message": "",
			"solution": map[string]any{
				"response": body,
				"status":   http.StatusOK,
				"url":      targetURL,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("failed to encode flaresolverr response: %v", err)
		}
	}))
}

func newFlareSolverrProxyServer(t *testing.T, targetServer *httptest.Server) *httptest.Server {
	t.Helper()

	return newFlareSolverrJSONServer(t, func(targetURL string) string {
		proxyResp, err := http.Get(targetURL)
		if err != nil {
			t.Fatalf("failed to proxy to target: %v", err)
		}
		defer proxyResp.Body.Close()
		raw, err := io.ReadAll(proxyResp.Body)
		if err != nil {
			t.Fatalf("failed to read proxy response: %v", err)
		}
		return string(raw)
	})
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}
	return string(raw)
}

func assertPath(t *testing.T, r *http.Request, expected string) {
	t.Helper()
	if r.URL.Path != expected {
		t.Fatalf("expected path %s, got %s", expected, r.URL.Path)
	}
}

func assertQuery(t *testing.T, r *http.Request, key, expected string) {
	t.Helper()
	if got := r.URL.Query().Get(key); got != expected {
		t.Fatalf("expected query %s=%s, got %s", key, expected, got)
	}
}

func testFetchOptions(fsURL string) FetchOptions {
	return FetchOptions{FlareSolverrURL: fsURL, MaxTimeoutMs: 120000}
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func failUnexpectedRequest(t *testing.T, r *http.Request) {
	t.Helper()
	t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
}

func baseURLOf(server *httptest.Server) string {
	return strings.TrimSuffix(server.URL, "/")
}

func formatErrBody(body string) string {
	return fmt.Sprintf("response body: %s", body)
}

func readFixture(t *testing.T, dir, name string) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve current file")
	}

	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	path := filepath.Join(repoRoot, "testdata", dir, name)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", path, err)
	}
	return string(raw)
}
