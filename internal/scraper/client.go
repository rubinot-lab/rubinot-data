package scraper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
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
	CDPURL          string
}

type Client struct {
	httpClient      *resty.Client
	flareSolverrURL string
	maxTimeoutMs    int
	semaphore       chan struct{}
	cdpURL          string
}

type flareSolverrRequest struct {
	Cmd        string            `json:"cmd"`
	URL        string            `json:"url,omitempty"`
	Session    string            `json:"session,omitempty"`
	MaxTimeout int               `json:"maxTimeout,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
}

type flareSolverrCookie struct {
	Name    string  `json:"name"`
	Value   string  `json:"value"`
	Domain  string  `json:"domain"`
	Path    string  `json:"path"`
	Expires float64 `json:"expires"`
}

type flareSolverrResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	Solution struct {
		Response  string               `json:"response"`
		Status    int                  `json:"status"`
		URL       string               `json:"url"`
		Cookies   []flareSolverrCookie `json:"cookies"`
		UserAgent string               `json:"userAgent"`
	} `json:"solution"`
}

const cdpSessionName = "rubinot-cdp"

var (
	scrapeSemaphoreOnce sync.Once
	scrapeSemaphore     chan struct{}
	sharedHTTPOnce      sync.Once
	sharedHTTPClient    *resty.Client
	htmlTagPattern      = regexp.MustCompile(`<[^>]+>`)
	maintenancePattern  = regexp.MustCompile(`(?is)server\s+is\s+under\s+maintenance,\s*please\s+visit\s+later\.?`)

	globalCDPMu    sync.Mutex
	globalCDP      *CDPClient
	globalCDPReady bool
)

func sharedRestyClient() *resty.Client {
	sharedHTTPOnce.Do(func() {
		sharedHTTPClient = resty.New().SetTimeout(defaultRequestTimeout).SetRetryCount(0)
	})
	return sharedHTTPClient
}

func NewClient(opts FetchOptions) *Client {
	normalized := normalizeFetchOptions(opts)
	return &Client{
		httpClient:      sharedRestyClient(),
		flareSolverrURL: normalized.FlareSolverrURL,
		maxTimeoutMs:    normalized.MaxTimeoutMs,
		semaphore:       sharedScrapeSemaphore(),
		cdpURL:          normalized.CDPURL,
	}
}

func (c *Client) Fetch(ctx context.Context, sourceURL string) (string, error) {
	ctx, span := tracer.Start(ctx, "scraper.Client.Fetch")
	defer span.End()

	if err := acquireSemaphore(ctx, c.semaphore); err != nil {
		FlareSolverrRequests.WithLabelValues("timeout").Inc()
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

	fsStarted := time.Now()
	var out flareSolverrResponse
	res, err := c.httpClient.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(payload).
		SetResult(&out).
		Post(c.flareSolverrURL)
	FlareSolverrDuration.Observe(time.Since(fsStarted).Seconds())

	if err != nil {
		mapped := mapClientError(err)
		if isTimeoutError(err) {
			FlareSolverrRequests.WithLabelValues("timeout").Inc()
		} else {
			FlareSolverrRequests.WithLabelValues("error").Inc()
		}
		return "", mapped
	}
	if res.StatusCode() != http.StatusOK {
		FlareSolverrRequests.WithLabelValues("error").Inc()
		return "", validation.NewError(validation.ErrorFlareSolverrNon200, fmt.Sprintf("flaresolverr returned non-200: %d", res.StatusCode()), nil)
	}
	if strings.ToLower(out.Status) != "ok" {
		code := validation.ErrorUpstreamUnknown
		if isTimeoutText(out.Message) {
			code = validation.ErrorFlareSolverrTimeout
			FlareSolverrRequests.WithLabelValues("timeout").Inc()
		} else {
			FlareSolverrRequests.WithLabelValues("error").Inc()
		}
		return "", validation.NewError(code, fmt.Sprintf("flaresolverr error: %s", out.Message), nil)
	}

	UpstreamStatus.WithLabelValues(endpointFromURL(sourceURL), strconv.Itoa(out.Solution.Status)).Inc()

	switch out.Solution.Status {
	case http.StatusOK, http.StatusNotFound:
	case http.StatusServiceUnavailable:
		FlareSolverrRequests.WithLabelValues("ok").Inc()
		UpstreamMaintenance.Inc()
		return "", validation.NewError(validation.ErrorUpstreamMaintenanceMode, validation.UpstreamMaintenanceMessage, nil)
	case http.StatusForbidden:
		FlareSolverrRequests.WithLabelValues("ok").Inc()
		return "", validation.NewError(validation.ErrorUpstreamForbidden, fmt.Sprintf("target returned forbidden status via flaresolverr: %d", out.Solution.Status), nil)
	default:
		FlareSolverrRequests.WithLabelValues("ok").Inc()
		return "", validation.NewError(validation.ErrorUpstreamUnknown, fmt.Sprintf("target returned non-200 via flaresolverr: %d", out.Solution.Status), nil)
	}

	body := out.Solution.Response
	if isMaintenanceURL(out.Solution.URL) {
		FlareSolverrRequests.WithLabelValues("ok").Inc()
		UpstreamMaintenance.Inc()
		return "", validation.NewError(validation.ErrorUpstreamMaintenanceMode, validation.UpstreamMaintenanceMessage, nil)
	}

	lowerBody := strings.ToLower(body)
	if strings.Contains(lowerBody, "just a moment") || strings.Contains(lowerBody, "cf-browser-verification") {
		FlareSolverrRequests.WithLabelValues("cf_challenge").Inc()
		CloudflareChallenges.Inc()
		return "", validation.NewError(validation.ErrorCloudflareChallengePresent, "cloudflare challenge page still present after flaresolverr", nil)
	}
	if containsMaintenanceMessage(body) {
		FlareSolverrRequests.WithLabelValues("ok").Inc()
		UpstreamMaintenance.Inc()
		return "", validation.NewError(validation.ErrorUpstreamMaintenanceMode, validation.UpstreamMaintenanceMessage, nil)
	}

	FlareSolverrRequests.WithLabelValues("ok").Inc()
	return body, nil
}

func (c *Client) ensureCDP(ctx context.Context) (*CDPClient, error) {
	if c.cdpURL == "" {
		return nil, nil
	}

	globalCDPMu.Lock()
	defer globalCDPMu.Unlock()

	if globalCDPReady && globalCDP != nil && globalCDP.IsConnected() && globalCDP.baseURL == c.cdpURL {
		return globalCDP, nil
	}

	if globalCDP != nil {
		globalCDP.Close()
		globalCDP = nil
		globalCDPReady = false
	}

	if c.flareSolverrURL != "" {
		type sessionReq struct {
			Cmd     string `json:"cmd"`
			Session string `json:"session,omitempty"`
		}
		_, _ = c.httpClient.R().
			SetContext(ctx).
			SetHeader("Content-Type", "application/json").
			SetBody(sessionReq{Cmd: "sessions.create", Session: cdpSessionName}).
			Post(c.flareSolverrURL)

		var fsResp flareSolverrResponse
		_, err := c.httpClient.R().
			SetContext(ctx).
			SetHeader("Content-Type", "application/json").
			SetBody(flareSolverrRequest{
				Cmd:        "request.get",
				URL:        strings.TrimRight(os.Getenv("RUBINOT_BASE_URL"), "/") + "/",
				MaxTimeout: c.maxTimeoutMs,
				Session:    cdpSessionName,
			}).
			SetResult(&fsResp).
			Post(c.flareSolverrURL)
		if err != nil {
			return nil, fmt.Errorf("navigate via FlareSolverr for CDP init: %w", err)
		}
	}

	cdp := NewCDPClient(c.cdpURL)
	if err := cdp.Connect(ctx); err != nil {
		return nil, fmt.Errorf("connect CDP: %w", err)
	}

	globalCDP = cdp
	globalCDPReady = true
	return cdp, nil
}

func (c *Client) FetchJSON(ctx context.Context, apiURL string, result any) error {
	ctx, span := tracer.Start(ctx, "scraper.Client.FetchJSON")
	defer span.End()

	body, err := c.fetchJSONBody(ctx, apiURL)
	if err != nil {
		return err
	}

	return parseJSONBody(body, result)
}

func (c *Client) fetchJSONBody(ctx context.Context, apiURL string) (string, error) {
	cdp, err := c.ensureCDP(ctx)
	if err != nil {
		return "", validation.NewError(validation.ErrorFlareSolverrConnection, fmt.Sprintf("CDP init failed: %v", err), err)
	}
	if cdp == nil {
		return "", validation.NewError(validation.ErrorFlareSolverrConnection, "CDP_URL not configured; JSON API requires CDP", nil)
	}

	return c.fetchViaCDP(ctx, cdp, apiURL)
}

func (c *Client) fetchViaCDP(ctx context.Context, cdp *CDPClient, apiURL string) (string, error) {
	parsed, err := url.Parse(apiURL)
	if err != nil {
		return "", fmt.Errorf("parse API URL: %w", err)
	}
	fetchPath := parsed.Path
	if parsed.RawQuery != "" {
		fetchPath += "?" + parsed.RawQuery
	}

	started := time.Now()
	body, err := cdp.Fetch(ctx, fetchPath)
	CDPFetchDuration.Observe(time.Since(started).Seconds())

	if err != nil {
		CDPFetchRequests.WithLabelValues("error").Inc()
		globalCDPMu.Lock()
		globalCDPReady = false
		globalCDPMu.Unlock()
		return "", validation.NewError(validation.ErrorFlareSolverrConnection, fmt.Sprintf("CDP fetch failed: %v", err), err)
	}

	CDPFetchRequests.WithLabelValues("ok").Inc()
	UpstreamStatus.WithLabelValues(endpointFromURL(apiURL), "200").Inc()
	return body, nil
}

func parseJSONBody(body string, result any) error {
	var errResp struct {
		Error string `json:"error"`
	}
	if json.Unmarshal([]byte(body), &errResp) == nil && strings.TrimSpace(errResp.Error) != "" {
		lower := strings.ToLower(errResp.Error)
		switch {
		case strings.Contains(lower, "maintenance"):
			UpstreamMaintenance.Inc()
			return validation.NewError(validation.ErrorUpstreamMaintenanceMode, validation.UpstreamMaintenanceMessage, nil)
		case strings.Contains(lower, "not found"):
			return validation.NewError(validation.ErrorEntityNotFound, errResp.Error, nil)
		case strings.Contains(lower, "access denied"):
			return validation.NewError(validation.ErrorUpstreamForbidden, "API access denied", nil)
		default:
			return validation.NewError(validation.ErrorUpstreamUnknown, errResp.Error, nil)
		}
	}

	if err := json.Unmarshal([]byte(body), result); err != nil {
		return validation.NewError(validation.ErrorUpstreamUnknown, fmt.Sprintf("failed to decode JSON: %v", err), err)
	}

	return nil
}

func endpointFromURL(sourceURL string) string {
	lower := strings.ToLower(sourceURL)
	switch {
	case strings.Contains(lower, "/api/worlds/"):
		return "world"
	case strings.Contains(lower, "/api/worlds"):
		return "worlds"
	case strings.Contains(lower, "/api/characters"):
		return "character"
	case strings.Contains(lower, "/api/guilds/"):
		return "guild"
	case strings.Contains(lower, "/api/guilds"):
		return "guilds"
	case strings.Contains(lower, "/api/highscores"):
		return "highscores"
	case strings.Contains(lower, "/api/killstats"):
		return "killstatistics"
	case strings.Contains(lower, "/api/deaths"):
		return "deaths"
	case strings.Contains(lower, "/api/transfers"):
		return "transfers"
	case strings.Contains(lower, "/api/bans"):
		return "banishments"
	case strings.Contains(lower, "/api/events"):
		return "events"
	case strings.Contains(lower, "/api/news"):
		return "news"
	case strings.Contains(lower, "/api/bazaar"):
		return "auctions"
	case strings.Contains(lower, "subtopic=worlds") && !strings.Contains(lower, "world="):
		return "worlds"
	case strings.Contains(lower, "subtopic=worlds") && strings.Contains(lower, "world="):
		return "world"
	case strings.Contains(lower, "subtopic=characters"):
		return "character"
	case strings.Contains(lower, "subtopic=guilds") && strings.Contains(lower, "guildname="):
		return "guild"
	case strings.Contains(lower, "subtopic=guilds"):
		return "guilds"
	case strings.Contains(lower, "subtopic=houses") && strings.Contains(lower, "houseid="):
		return "house"
	case strings.Contains(lower, "subtopic=houses"):
		return "houses"
	case strings.Contains(lower, "subtopic=highscores"):
		return "highscores"
	case strings.Contains(lower, "subtopic=killstatistics"):
		return "killstatistics"
	case strings.Contains(lower, "subtopic=latestdeaths"):
		return "deaths"
	case strings.Contains(lower, "subtopic=transferstatistics"):
		return "transfers"
	case strings.Contains(lower, "subtopic=bans"):
		return "banishments"
	case strings.Contains(lower, "subtopic=eventcalendar"):
		return "events"
	case strings.Contains(lower, "charactertrades"):
		return "auctions"
	case strings.Contains(lower, "/news"):
		return "news"
	default:
		return "unknown"
	}
}

func normalizeFetchOptions(opts FetchOptions) FetchOptions {
	if opts.FlareSolverrURL == "" && opts.CDPURL == "" {
		opts.FlareSolverrURL = defaultFlareSolverrURL
	}
	if opts.MaxTimeoutMs <= 0 {
		opts.MaxTimeoutMs = defaultMaxTimeoutMs
	}
	if opts.CDPURL == "" {
		opts.CDPURL = os.Getenv("CDP_URL")
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

func containsMaintenanceMessage(html string) bool {
	if maintenancePattern.MatchString(html) {
		return true
	}

	withoutTags := htmlTagPattern.ReplaceAllString(html, " ")
	return maintenancePattern.MatchString(withoutTags)
}

func isMaintenanceURL(rawURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return false
	}

	path := strings.ToLower(strings.TrimRight(parsed.Path, "/"))
	return path == "/maintenance"
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
	sharedHTTPClient = nil
	sharedHTTPOnce = sync.Once{}
	globalCDPMu.Lock()
	globalCDP = nil
	globalCDPReady = false
	globalCDPMu.Unlock()
}
