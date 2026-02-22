package scraper

import (
	"context"
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
