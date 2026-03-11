package scraper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
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

var cdpFetchPathRe = regexp.MustCompile(`fetch\('([^']+)'\)`)

var wsUpgrader = websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}

func resetGlobalCDPForTests() {
	globalCDPMu.Lock()
	if globalCDP != nil {
		globalCDP.Close()
	}
	globalCDP = nil
	globalCDPReady = false
	globalCDPMu.Unlock()
}

func newMockCDPServer(t *testing.T, contentForPath func(string) string) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/json/list", func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		targets := []map[string]string{{
			"id":                   "MOCK_PAGE_1",
			"type":                 "page",
			"url":                  "https://rubinot.com.br/news",
			"webSocketDebuggerUrl": fmt.Sprintf("ws://%s/devtools/page/MOCK_PAGE_1", host),
		}}
		writeJSON(w, targets)
	})

	mux.HandleFunc("/devtools/page/", func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("websocket upgrade: %v", err)
		}
		defer conn.Close()

		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var req struct {
				ID     int64 `json:"id"`
				Method string `json:"method"`
				Params struct {
					Expression string `json:"expression"`
				} `json:"params"`
			}
			if err := json.Unmarshal(data, &req); err != nil {
				continue
			}

			var value string
			if req.Method == "Runtime.evaluate" {
				matches := cdpFetchPathRe.FindAllStringSubmatch(req.Params.Expression, -1)
				if len(matches) > 0 && contentForPath != nil {
					isBatch := strings.Contains(req.Params.Expression, "Promise.allSettled")
					isBinary := strings.Contains(req.Params.Expression, "arrayBuffer")
					if len(matches) == 1 && !isBatch {
						body := contentForPath(matches[0][1])
						if isBinary {
							value = body
						} else {
							wrapper := map[string]any{"ok": true, "status": 200, "body": body}
							value = mustJSON(t, wrapper)
						}
					} else {
						results := make([]map[string]string, 0, len(matches))
						for _, match := range matches {
							body := contentForPath(match[1])
							if strings.HasPrefix(body, "__REJECT__:") {
								results = append(results, map[string]string{
									"status": "rejected",
									"value":  strings.TrimPrefix(body, "__REJECT__:"),
								})
								continue
							}
							results = append(results, map[string]string{
								"status": "fulfilled",
								"value":  body,
							})
						}
						value = mustJSON(t, results)
					}
				}
			}

			resp := map[string]any{
				"id": req.ID,
				"result": map[string]any{
					"result": map[string]any{
						"type":  "string",
						"value": value,
					},
				},
			}
			conn.WriteJSON(resp)
		}
	})

	return httptest.NewServer(mux)
}

func newMockCDPProxyServer(t *testing.T, targetServer *httptest.Server) *httptest.Server {
	t.Helper()
	resetGlobalCDPForTests()

	return newMockCDPServer(t, func(apiPath string) string {
		resp, err := http.Get(targetServer.URL + apiPath)
		if err != nil {
			t.Fatalf("mock CDP proxy failed: %v", err)
		}
		defer resp.Body.Close()
		raw, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("mock CDP proxy read: %v", err)
		}
		return string(raw)
	})
}

func testFetchOptionsWithCDP(fsURL, cdpURL string) FetchOptions {
	return FetchOptions{FlareSolverrURL: fsURL, MaxTimeoutMs: 120000, CDPURL: cdpURL}
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
