package scraper

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/giovannirco/rubinot-data/internal/validation"
	"github.com/go-resty/resty/v2"
)

const (
	defaultFlareSolverrURL  = "http://flaresolverr.network.svc.cluster.local:8191/v1"
	defaultMaxTimeoutMs     = 120000
	defaultMaxConcurrency   = 8
	defaultRequestTimeout   = 140 * time.Second
	defaultBrowserUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36"
)

type FetchOptions struct {
	FlareSolverrURL string
	MaxTimeoutMs    int
}

type Client struct {
	httpClient      *resty.Client
	flareSolverrURL string
	maxTimeoutMs    int
	semaphore       chan struct{}
}

type flareSolverrRequest struct {
	Cmd        string            `json:"cmd"`
	URL        string            `json:"url"`
	MaxTimeout int               `json:"maxTimeout,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
}

type flareSolverrResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	Solution struct {
		Response string `json:"response"`
		Status   int    `json:"status"`
		URL      string `json:"url"`
	} `json:"solution"`
}

var (
	scrapeSemaphoreOnce sync.Once
	scrapeSemaphore     chan struct{}
)

func NewClient(opts FetchOptions) *Client {
	normalized := normalizeFetchOptions(opts)
	return &Client{
		httpClient:      resty.New().SetTimeout(defaultRequestTimeout).SetRetryCount(0),
		flareSolverrURL: normalized.FlareSolverrURL,
		maxTimeoutMs:    normalized.MaxTimeoutMs,
		semaphore:       sharedScrapeSemaphore(),
	}
}

func (c *Client) Fetch(ctx context.Context, sourceURL string) (string, error) {
	ctx, span := tracer.Start(ctx, "scraper.Client.Fetch")
	defer span.End()

	if err := acquireSemaphore(ctx, c.semaphore); err != nil {
		return "", validation.NewError(validation.ErrorFlareSolverrTimeout, fmt.Sprintf("flaresolverr timeout: %v", err), err)
	}
	defer releaseSemaphore(c.semaphore)

	payload := flareSolverrRequest{
		Cmd:        "request.get",
		URL:        sourceURL,
		MaxTimeout: c.maxTimeoutMs,
		Headers: map[string]string{
			"User-Agent":      defaultBrowserUserAgent,
			"Accept-Language": "en-US,en;q=0.9,pt-BR;q=0.8",
		},
	}

	var out flareSolverrResponse
	res, err := c.httpClient.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(payload).
		SetResult(&out).
		Post(c.flareSolverrURL)
	if err != nil {
		return "", mapClientError(err)
	}
	if res.StatusCode() != http.StatusOK {
		return "", validation.NewError(validation.ErrorFlareSolverrNon200, fmt.Sprintf("flaresolverr returned non-200: %d", res.StatusCode()), nil)
	}
	if strings.ToLower(out.Status) != "ok" {
		code := validation.ErrorUpstreamUnknown
		if isTimeoutText(out.Message) {
			code = validation.ErrorFlareSolverrTimeout
		}
		return "", validation.NewError(code, fmt.Sprintf("flaresolverr error: %s", out.Message), nil)
	}
	switch out.Solution.Status {
	case http.StatusOK:
	case http.StatusServiceUnavailable:
		return "", validation.NewError(validation.ErrorUpstreamMaintenanceMode, fmt.Sprintf("target returned maintenance status via flaresolverr: %d", out.Solution.Status), nil)
	case http.StatusForbidden:
		return "", validation.NewError(validation.ErrorUpstreamForbidden, fmt.Sprintf("target returned forbidden status via flaresolverr: %d", out.Solution.Status), nil)
	default:
		return "", validation.NewError(validation.ErrorUpstreamUnknown, fmt.Sprintf("target returned non-200 via flaresolverr: %d", out.Solution.Status), nil)
	}

	html := out.Solution.Response
	lowerHTML := strings.ToLower(html)
	if strings.Contains(lowerHTML, "just a moment") || strings.Contains(lowerHTML, "cf-browser-verification") {
		return "", validation.NewError(validation.ErrorCloudflareChallengePresent, "cloudflare challenge page still present after flaresolverr", nil)
	}
	if strings.Contains(lowerHTML, "maintenance") {
		return "", validation.NewError(validation.ErrorUpstreamMaintenanceMode, "upstream maintenance mode detected", nil)
	}

	return html, nil
}

func normalizeFetchOptions(opts FetchOptions) FetchOptions {
	if opts.FlareSolverrURL == "" {
		opts.FlareSolverrURL = defaultFlareSolverrURL
	}
	if opts.MaxTimeoutMs <= 0 {
		opts.MaxTimeoutMs = defaultMaxTimeoutMs
	}
	return opts
}

func sharedScrapeSemaphore() chan struct{} {
	scrapeSemaphoreOnce.Do(func() {
		scrapeSemaphore = make(chan struct{}, envInt("SCRAPE_MAX_CONCURRENCY", defaultMaxConcurrency))
	})
	return scrapeSemaphore
}

func acquireSemaphore(ctx context.Context, semaphore chan struct{}) error {
	select {
	case semaphore <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func releaseSemaphore(semaphore chan struct{}) {
	select {
	case <-semaphore:
	default:
	}
}

func mapClientError(err error) error {
	if isTimeoutError(err) {
		return validation.NewError(validation.ErrorFlareSolverrTimeout, fmt.Sprintf("flaresolverr timeout: %v", err), err)
	}
	return validation.NewError(validation.ErrorFlareSolverrConnection, fmt.Sprintf("flaresolverr request failed: %v", err), err)
}

func isTimeoutError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	return isTimeoutText(err.Error())
}

func isTimeoutText(text string) bool {
	lower := strings.ToLower(text)
	return strings.Contains(lower, "timeout") || strings.Contains(lower, "deadline exceeded")
}

func envInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func resetSharedScrapeSemaphoreForTests() {
	scrapeSemaphore = nil
	scrapeSemaphoreOnce = sync.Once{}
}
