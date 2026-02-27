package scraper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/giovannirco/rubinot-data/internal/validation"
)

func TestClientFetchSuccess(t *testing.T) {
	server := newFlareSolverrServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, flaresolverrResponseJSON("ok", "", http.StatusOK, "<html><body>ok</body></html>"))
	})
	defer server.Close()

	client := newClientForTest(t, server.URL, 8)
	html, err := client.Fetch(context.Background(), "https://www.rubinot.com.br/?subtopic=worlds")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if html != "<html><body>ok</body></html>" {
		t.Fatalf("unexpected html response: %q", html)
	}
}

func TestClientFetchFlareSolverrDown(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to reserve local port: %v", err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()

	client := newClientForTest(t, "http://"+addr, 8)
	_, fetchErr := client.Fetch(context.Background(), "https://www.rubinot.com.br")
	assertValidationErrorCode(t, fetchErr, validation.ErrorFlareSolverrConnection)
}

func TestClientFetchFlareSolverrNon200(t *testing.T) {
	server := newFlareSolverrServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprint(w, `{"status":"error","message":"bad gateway"}`)
	})
	defer server.Close()

	client := newClientForTest(t, server.URL, 8)
	_, fetchErr := client.Fetch(context.Background(), "https://www.rubinot.com.br")
	assertValidationErrorCode(t, fetchErr, validation.ErrorFlareSolverrNon200)
}

func TestClientFetchTargetForbidden(t *testing.T) {
	server := newFlareSolverrServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, flaresolverrResponseJSON("ok", "", http.StatusForbidden, "<html></html>"))
	})
	defer server.Close()

	client := newClientForTest(t, server.URL, 8)
	_, fetchErr := client.Fetch(context.Background(), "https://www.rubinot.com.br")
	assertValidationErrorCode(t, fetchErr, validation.ErrorUpstreamForbidden)
}

func TestClientFetchTargetMaintenance(t *testing.T) {
	server := newFlareSolverrServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, flaresolverrResponseJSON("ok", "", http.StatusServiceUnavailable, "<html></html>"))
	})
	defer server.Close()

	client := newClientForTest(t, server.URL, 8)
	_, fetchErr := client.Fetch(context.Background(), "https://www.rubinot.com.br")
	assertValidationErrorCode(t, fetchErr, validation.ErrorUpstreamMaintenanceMode)
	if fetchErr.Error() != validation.UpstreamMaintenanceMessage {
		t.Fatalf("expected maintenance message %q, got %q", validation.UpstreamMaintenanceMessage, fetchErr.Error())
	}
}

func TestClientFetchTargetMaintenanceMessageInHTML(t *testing.T) {
	server := newFlareSolverrServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, flaresolverrResponseJSON("ok", "", http.StatusOK, "<html><body><p>Server is under maintenance, please visit later.</p></body></html>"))
	})
	defer server.Close()

	client := newClientForTest(t, server.URL, 8)
	_, fetchErr := client.Fetch(context.Background(), "https://www.rubinot.com.br")
	assertValidationErrorCode(t, fetchErr, validation.ErrorUpstreamMaintenanceMode)
	if fetchErr.Error() != validation.UpstreamMaintenanceMessage {
		t.Fatalf("expected maintenance message %q, got %q", validation.UpstreamMaintenanceMessage, fetchErr.Error())
	}
}

func TestClientFetchTargetMaintenanceURL(t *testing.T) {
	server := newFlareSolverrServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(
			w,
			`{"status":"ok","message":"","solution":{"response":"<html><body>temporary page</body></html>","status":200,"url":"https://www.rubinot.com.br/maintenance"}}`,
		)
	})
	defer server.Close()

	client := newClientForTest(t, server.URL, 8)
	_, fetchErr := client.Fetch(context.Background(), "https://www.rubinot.com.br")
	assertValidationErrorCode(t, fetchErr, validation.ErrorUpstreamMaintenanceMode)
	if fetchErr.Error() != validation.UpstreamMaintenanceMessage {
		t.Fatalf("expected maintenance message %q, got %q", validation.UpstreamMaintenanceMessage, fetchErr.Error())
	}
}

func TestClientFetchTargetUnknownError(t *testing.T) {
	server := newFlareSolverrServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, flaresolverrResponseJSON("ok", "", http.StatusInternalServerError, "<html></html>"))
	})
	defer server.Close()

	client := newClientForTest(t, server.URL, 8)
	_, fetchErr := client.Fetch(context.Background(), "https://www.rubinot.com.br")
	assertValidationErrorCode(t, fetchErr, validation.ErrorUpstreamUnknown)
}

func TestClientFetchCloudflareChallenge(t *testing.T) {
	server := newFlareSolverrServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, flaresolverrResponseJSON("ok", "", http.StatusOK, "<html><title>Just a moment...</title></html>"))
	})
	defer server.Close()

	client := newClientForTest(t, server.URL, 8)
	_, fetchErr := client.Fetch(context.Background(), "https://www.rubinot.com.br")
	assertValidationErrorCode(t, fetchErr, validation.ErrorCloudflareChallengePresent)
}

func TestClientFetchTimeoutMapping(t *testing.T) {
	server := newFlareSolverrServer(t, func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(150 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, flaresolverrResponseJSON("ok", "", http.StatusOK, "<html></html>"))
	})
	defer server.Close()

	client := newClientForTest(t, server.URL, 8)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	_, fetchErr := client.Fetch(ctx, "https://www.rubinot.com.br")
	assertValidationErrorCode(t, fetchErr, validation.ErrorFlareSolverrTimeout)
}

func TestClientFetchRespectsSemaphoreConcurrency(t *testing.T) {
	var inFlight atomic.Int32
	var maxInFlight atomic.Int32

	server := newFlareSolverrServer(t, func(w http.ResponseWriter, _ *http.Request) {
		current := inFlight.Add(1)
		defer inFlight.Add(-1)
		for {
			max := maxInFlight.Load()
			if current <= max || maxInFlight.CompareAndSwap(max, current) {
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, flaresolverrResponseJSON("ok", "", http.StatusOK, "<html></html>"))
	})
	defer server.Close()

	client := newClientForTest(t, server.URL, 1)

	var wg sync.WaitGroup
	wg.Add(2)

	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			_, err := client.Fetch(context.Background(), "https://www.rubinot.com.br")
			if err != nil {
				t.Errorf("unexpected fetch error: %v", err)
			}
		}()
	}

	wg.Wait()
	if maxInFlight.Load() != 1 {
		t.Fatalf("expected max in-flight requests to be 1, got %d", maxInFlight.Load())
	}
}

func newClientForTest(t *testing.T, flaresolverrURL string, maxConcurrency int) *Client {
	t.Helper()
	t.Setenv("SCRAPE_MAX_CONCURRENCY", fmt.Sprintf("%d", maxConcurrency))
	resetSharedScrapeSemaphoreForTests()

	return NewClient(FetchOptions{
		FlareSolverrURL: flaresolverrURL,
		MaxTimeoutMs:    120000,
	})
}

func newFlareSolverrServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected HTTP method: %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		handler(w, r)
	}))
}

func flaresolverrResponseJSON(status, message string, targetStatus int, html string) string {
	return fmt.Sprintf(
		`{"status":%q,"message":%q,"solution":{"response":%q,"status":%d,"url":"https://www.rubinot.com.br"}}`,
		status,
		message,
		html,
		targetStatus,
	)
}

func assertValidationErrorCode(t *testing.T, err error, expectedCode int) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error code %d but got nil", expectedCode)
	}

	var validationErr validation.Error
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected validation.Error, got %T: %v", err, err)
	}
	if validationErr.Code() != expectedCode {
		t.Fatalf("expected error code %d, got %d (err=%v)", expectedCode, validationErr.Code(), err)
	}
}

func TestClientFetchAllPagesSuccess(t *testing.T) {
	resetGlobalCDPForTests()
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/deaths" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		page := r.URL.Query().Get("page")
		writeJSON(w, map[string]any{
			"deaths": []map[string]any{{"page": page}},
			"pagination": map[string]any{
				"totalPages": 3,
			},
		})
	}))
	defer api.Close()

	cdpSrv := newMockCDPProxyServer(t, api)
	defer cdpSrv.Close()

	client := NewClient(testFetchOptionsWithCDP("", cdpSrv.URL))
	first := baseURLOf(api) + "/api/deaths?page=1"
	buildPageURL := func(page int) string {
		return fmt.Sprintf("%s/api/deaths?page=%d", baseURLOf(api), page)
	}
	extractTotalPages := func(body string) (int, error) {
		var payload struct {
			Pagination struct {
				TotalPages int `json:"totalPages"`
			} `json:"pagination"`
		}
		if err := json.Unmarshal([]byte(body), &payload); err != nil {
			return 0, err
		}
		return payload.Pagination.TotalPages, nil
	}

	bodies, sources, err := client.FetchAllPages(context.Background(), first, buildPageURL, extractTotalPages)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(bodies) != 3 {
		t.Fatalf("expected 3 bodies, got %d", len(bodies))
	}
	if len(sources) != 3 {
		t.Fatalf("expected 3 sources, got %d", len(sources))
	}
	if sources[2] != baseURLOf(api)+"/api/deaths?page=3" {
		t.Fatalf("unexpected page 3 source: %s", sources[2])
	}
}

func TestClientFetchAllPagesRetriesNonJSON(t *testing.T) {
	resetGlobalCDPForTests()
	var pageTwoCalls atomic.Int32
	cdpSrv := newMockCDPServer(t, func(path string) string {
		switch path {
		case "/api/deaths?page=1":
			return `{"pagination":{"totalPages":2},"deaths":[]}`
		case "/api/deaths?page=2":
			if pageTwoCalls.Add(1) < 3 {
				return "<html>challenge</html>"
			}
			return `{"pagination":{"totalPages":2},"deaths":[]}`
		default:
			return `{}`
		}
	})
	defer cdpSrv.Close()

	client := NewClient(testFetchOptionsWithCDP("", cdpSrv.URL))
	first := "https://rubinot.com.br/api/deaths?page=1"
	buildPageURL := func(page int) string {
		return fmt.Sprintf("https://rubinot.com.br/api/deaths?page=%d", page)
	}
	extractTotalPages := func(body string) (int, error) {
		var payload struct {
			Pagination struct {
				TotalPages int `json:"totalPages"`
			} `json:"pagination"`
		}
		if err := json.Unmarshal([]byte(body), &payload); err != nil {
			return 0, err
		}
		return payload.Pagination.TotalPages, nil
	}

	bodies, _, err := client.FetchAllPages(context.Background(), first, buildPageURL, extractTotalPages)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(bodies) != 2 {
		t.Fatalf("expected 2 bodies, got %d", len(bodies))
	}
	if pageTwoCalls.Load() != 3 {
		t.Fatalf("expected 3 attempts for page 2, got %d", pageTwoCalls.Load())
	}
}

func TestClientFetchAllPagesExhaustedRetries(t *testing.T) {
	resetGlobalCDPForTests()
	cdpSrv := newMockCDPServer(t, func(path string) string {
		switch path {
		case "/api/deaths?page=1":
			return `{"pagination":{"totalPages":2},"deaths":[]}`
		case "/api/deaths?page=2":
			return "__REJECT__:net::ERR_FAILED"
		default:
			return `{}`
		}
	})
	defer cdpSrv.Close()

	client := NewClient(testFetchOptionsWithCDP("", cdpSrv.URL))
	first := "https://rubinot.com.br/api/deaths?page=1"
	buildPageURL := func(page int) string {
		return fmt.Sprintf("https://rubinot.com.br/api/deaths?page=%d", page)
	}
	extractTotalPages := func(body string) (int, error) {
		var payload struct {
			Pagination struct {
				TotalPages int `json:"totalPages"`
			} `json:"pagination"`
		}
		if err := json.Unmarshal([]byte(body), &payload); err != nil {
			return 0, err
		}
		return payload.Pagination.TotalPages, nil
	}

	_, _, err := client.FetchAllPages(context.Background(), first, buildPageURL, extractTotalPages)
	assertValidationErrorCode(t, err, validation.ErrorUpstreamUnknown)
}
