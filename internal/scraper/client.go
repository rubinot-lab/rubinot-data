package scraper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
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
	cookieMinTTL            = 30 * time.Second
	cookieFallbackTTL       = 10 * time.Minute
	cookieSafetyMargin      = 1 * time.Minute
)

type FetchOptions struct {
	FlareSolverrURL string
	MaxTimeoutMs    int
}

type Client struct {
	httpClient      *resty.Client
	directHTTP      *http.Client
	flareSolverrURL string
	maxTimeoutMs    int
	semaphore       chan struct{}
	cookieCache     *cookieCache
	browserFetcher  func(ctx context.Context, apiURL, origin string) ([]byte, int, error)
}

type flareSolverrRequest struct {
	Cmd        string            `json:"cmd"`
	URL        string            `json:"url"`
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

type fetchResult struct {
	HTML      string
	URL       string
	Cookies   []*http.Cookie
	UserAgent string
}

type cachedCookies struct {
	cookies   []*http.Cookie
	userAgent string
	expiresAt time.Time
}

type cookieCache struct {
	mu       sync.RWMutex
	byOrigin map[string]cachedCookies
}

func newCookieCache() *cookieCache {
	return &cookieCache{byOrigin: make(map[string]cachedCookies)}
}

func (cc *cookieCache) get(origin string) ([]*http.Cookie, string, bool) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	entry, ok := cc.byOrigin[origin]
	if !ok || len(entry.cookies) == 0 || time.Now().After(entry.expiresAt) {
		return nil, "", false
	}
	return cloneCookies(entry.cookies), entry.userAgent, true
}

func (cc *cookieCache) set(origin string, cookies []*http.Cookie, userAgent string, expiresAt time.Time) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.byOrigin[origin] = cachedCookies{
		cookies:   cloneCookies(cookies),
		userAgent: strings.TrimSpace(userAgent),
		expiresAt: expiresAt,
	}
}

func (cc *cookieCache) reset() {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.byOrigin = make(map[string]cachedCookies)
}

var (
	scrapeSemaphoreOnce sync.Once
	scrapeSemaphore     chan struct{}
	sharedHTTPOnce      sync.Once
	sharedHTTPClient    *resty.Client
	htmlTagPattern      = regexp.MustCompile(`<[^>]+>`)
	maintenancePattern  = regexp.MustCompile(`(?is)server\s+is\s+under\s+maintenance,\s*please\s+visit\s+later\.?`)
	sharedCookieCache   = newCookieCache()
)

func sharedRestyClient() *resty.Client {
	sharedHTTPOnce.Do(func() {
		sharedHTTPClient = resty.New().SetTimeout(defaultRequestTimeout).SetRetryCount(0)
	})
	return sharedHTTPClient
}

func NewClient(opts FetchOptions) *Client {
	normalized := normalizeFetchOptions(opts)
	client := &Client{
		httpClient:      sharedRestyClient(),
		directHTTP:      &http.Client{Timeout: defaultRequestTimeout},
		flareSolverrURL: normalized.FlareSolverrURL,
		maxTimeoutMs:    normalized.MaxTimeoutMs,
		semaphore:       sharedScrapeSemaphore(),
		cookieCache:     sharedCookieCache,
	}
	client.browserFetcher = client.fetchJSONViaBrowser
	return client
}

func (c *Client) Fetch(ctx context.Context, sourceURL string) (string, error) {
	result, err := c.fetch(ctx, sourceURL)
	if err != nil {
		return "", err
	}
	return result.HTML, nil
}

func (c *Client) fetch(ctx context.Context, sourceURL string) (*fetchResult, error) {
	ctx, span := tracer.Start(ctx, "scraper.Client.fetch")
	defer span.End()

	if err := acquireSemaphore(ctx, c.semaphore); err != nil {
		FlareSolverrRequests.WithLabelValues("timeout").Inc()
		return nil, validation.NewError(validation.ErrorFlareSolverrTimeout, fmt.Sprintf("flaresolverr timeout: %v", err), err)
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
		return nil, mapped
	}
	if res.StatusCode() != http.StatusOK {
		FlareSolverrRequests.WithLabelValues("error").Inc()
		return nil, validation.NewError(validation.ErrorFlareSolverrNon200, fmt.Sprintf("flaresolverr returned non-200: %d", res.StatusCode()), nil)
	}
	if strings.ToLower(out.Status) != "ok" {
		code := validation.ErrorUpstreamUnknown
		if isTimeoutText(out.Message) {
			code = validation.ErrorFlareSolverrTimeout
			FlareSolverrRequests.WithLabelValues("timeout").Inc()
		} else {
			FlareSolverrRequests.WithLabelValues("error").Inc()
		}
		return nil, validation.NewError(code, fmt.Sprintf("flaresolverr error: %s", out.Message), nil)
	}

	UpstreamStatus.WithLabelValues(endpointFromURL(sourceURL), strconv.Itoa(out.Solution.Status)).Inc()

	switch out.Solution.Status {
	case http.StatusOK:
	case http.StatusServiceUnavailable:
		FlareSolverrRequests.WithLabelValues("ok").Inc()
		UpstreamMaintenance.Inc()
		return nil, validation.NewError(validation.ErrorUpstreamMaintenanceMode, validation.UpstreamMaintenanceMessage, nil)
	case http.StatusForbidden:
		FlareSolverrRequests.WithLabelValues("ok").Inc()
		return nil, validation.NewError(validation.ErrorUpstreamForbidden, fmt.Sprintf("target returned forbidden status via flaresolverr: %d", out.Solution.Status), nil)
	default:
		FlareSolverrRequests.WithLabelValues("ok").Inc()
		return nil, validation.NewError(validation.ErrorUpstreamUnknown, fmt.Sprintf("target returned non-200 via flaresolverr: %d", out.Solution.Status), nil)
	}

	html := out.Solution.Response
	if isMaintenanceURL(out.Solution.URL) {
		FlareSolverrRequests.WithLabelValues("ok").Inc()
		UpstreamMaintenance.Inc()
		return nil, validation.NewError(validation.ErrorUpstreamMaintenanceMode, validation.UpstreamMaintenanceMessage, nil)
	}

	lowerHTML := strings.ToLower(html)
	if strings.Contains(lowerHTML, "just a moment") || strings.Contains(lowerHTML, "cf-browser-verification") {
		FlareSolverrRequests.WithLabelValues("cf_challenge").Inc()
		CloudflareChallenges.Inc()
		return nil, validation.NewError(validation.ErrorCloudflareChallengePresent, "cloudflare challenge page still present after flaresolverr", nil)
	}
	if containsMaintenanceMessage(html) {
		FlareSolverrRequests.WithLabelValues("ok").Inc()
		UpstreamMaintenance.Inc()
		return nil, validation.NewError(validation.ErrorUpstreamMaintenanceMode, validation.UpstreamMaintenanceMessage, nil)
	}

	FlareSolverrRequests.WithLabelValues("ok").Inc()
	return &fetchResult{
		HTML:      html,
		URL:       out.Solution.URL,
		Cookies:   toHTTPCookies(out.Solution.Cookies),
		UserAgent: strings.TrimSpace(out.Solution.UserAgent),
	}, nil
}

func (c *Client) FetchJSON(ctx context.Context, apiURL string, result any) error {
	ctx, span := tracer.Start(ctx, "scraper.Client.FetchJSON")
	defer span.End()

	origin, err := originFromURL(apiURL)
	if err != nil {
		return validation.NewError(validation.ErrorUpstreamUnknown, fmt.Sprintf("invalid API URL: %v", err), err)
	}

	cookies, userAgent, err := c.getOrRefreshCookies(ctx, origin)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return validation.NewError(validation.ErrorUpstreamUnknown, fmt.Sprintf("failed to create request: %v", err), err)
	}
	req.Header.Set("Origin", origin)
	req.Header.Set("Referer", origin+"/")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,pt-BR;q=0.8")
	if strings.TrimSpace(userAgent) == "" {
		userAgent = defaultBrowserUserAgent
	}
	req.Header.Set("User-Agent", userAgent)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	resp, err := c.directHTTP.Do(req)
	if err != nil {
		return mapClientError(err)
	}
	defer resp.Body.Close()

	UpstreamStatus.WithLabelValues(endpointFromURL(apiURL), strconv.Itoa(resp.StatusCode)).Inc()
	if resp.StatusCode == http.StatusServiceUnavailable {
		UpstreamMaintenance.Inc()
		return validation.NewError(validation.ErrorUpstreamMaintenanceMode, validation.UpstreamMaintenanceMessage, nil)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return validation.NewError(validation.ErrorUpstreamUnknown, fmt.Sprintf("failed to read response body: %v", err), err)
	}

	statusCode := resp.StatusCode
	if c.shouldFallbackToBrowser(statusCode, body) && c.browserFetcher != nil {
		browserBody, browserStatus, browserErr := c.browserFetcher(ctx, apiURL, origin)
		if browserErr == nil {
			body = browserBody
			statusCode = browserStatus
			UpstreamStatus.WithLabelValues(endpointFromURL(apiURL), fmt.Sprintf("%d-browser", browserStatus)).Inc()
		}
	}

	var errResp struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &errResp) == nil && strings.TrimSpace(errResp.Error) != "" {
		lower := strings.ToLower(errResp.Error)
		switch {
		case strings.Contains(lower, "maintenance"):
			UpstreamMaintenance.Inc()
			return validation.NewError(validation.ErrorUpstreamMaintenanceMode, validation.UpstreamMaintenanceMessage, nil)
		case strings.Contains(lower, "access denied"):
			c.cookieCache.reset()
			return validation.NewError(validation.ErrorUpstreamForbidden, "API access denied", nil)
		default:
			if resp.StatusCode == http.StatusNotFound {
				return validation.NewError(validation.ErrorEntityNotFound, errResp.Error, nil)
			}
			return validation.NewError(validation.ErrorUpstreamUnknown, errResp.Error, nil)
		}
	}

	switch statusCode {
	case http.StatusOK:
	case http.StatusForbidden, http.StatusUnauthorized:
		c.cookieCache.reset()
		return validation.NewError(validation.ErrorUpstreamForbidden, fmt.Sprintf("API returned %d", statusCode), nil)
	case http.StatusNotFound:
		return validation.NewError(validation.ErrorEntityNotFound, "entity not found", nil)
	default:
		return validation.NewError(validation.ErrorUpstreamUnknown, fmt.Sprintf("API returned %d", statusCode), nil)
	}

	if err := json.Unmarshal(body, result); err != nil {
		return validation.NewError(validation.ErrorUpstreamUnknown, fmt.Sprintf("failed to decode JSON: %v", err), err)
	}

	return nil
}

func (c *Client) shouldFallbackToBrowser(statusCode int, body []byte) bool {
	switch statusCode {
	case http.StatusForbidden, http.StatusUnauthorized, http.StatusTooManyRequests,
		http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	}

	if len(body) == 0 {
		return false
	}

	lowerBody := strings.ToLower(string(body))
	if strings.Contains(lowerBody, "cf-browser-verification") || strings.Contains(lowerBody, "just a moment") {
		return true
	}
	if strings.Contains(lowerBody, "\"error\":\"access denied\"") || strings.Contains(lowerBody, "access denied") {
		return true
	}

	return false
}

func (c *Client) getOrRefreshCookies(ctx context.Context, origin string) ([]*http.Cookie, string, error) {
	if cached, userAgent, ok := c.cookieCache.get(origin); ok {
		return cached, userAgent, nil
	}

	fetchResult, err := c.fetch(ctx, origin)
	if err != nil {
		return nil, "", err
	}

	if len(fetchResult.Cookies) == 0 {
		return nil, "", validation.NewError(validation.ErrorUpstreamUnknown, "flaresolverr did not return cookies", nil)
	}

	expiresAt := cookiesExpiry(fetchResult.Cookies)
	userAgent := strings.TrimSpace(fetchResult.UserAgent)
	if userAgent == "" {
		userAgent = defaultBrowserUserAgent
	}
	c.cookieCache.set(origin, fetchResult.Cookies, userAgent, expiresAt)
	return cloneCookies(fetchResult.Cookies), userAgent, nil
}

func cookiesExpiry(cookies []*http.Cookie) time.Time {
	now := time.Now()
	expiresAt := now.Add(cookieFallbackTTL)
	foundExpiringCookie := false

	for _, cookie := range cookies {
		if cookie == nil {
			continue
		}
		if cookie.Expires.IsZero() {
			continue
		}
		if cookie.Expires.Before(now) {
			continue
		}
		if !foundExpiringCookie || cookie.Expires.Before(expiresAt) {
			expiresAt = cookie.Expires
			foundExpiringCookie = true
		}
	}

	expiresAt = expiresAt.Add(-cookieSafetyMargin)
	if expiresAt.Before(now.Add(cookieMinTTL)) {
		return now.Add(cookieMinTTL)
	}
	return expiresAt
}

func toHTTPCookies(in []flareSolverrCookie) []*http.Cookie {
	out := make([]*http.Cookie, 0, len(in))
	now := time.Now()

	for _, item := range in {
		if strings.TrimSpace(item.Name) == "" || strings.TrimSpace(item.Value) == "" {
			continue
		}

		cookie := &http.Cookie{
			Name:   item.Name,
			Value:  item.Value,
			Domain: strings.TrimSpace(item.Domain),
			Path:   strings.TrimSpace(item.Path),
		}
		if cookie.Path == "" {
			cookie.Path = "/"
		}

		if !math.IsNaN(item.Expires) && !math.IsInf(item.Expires, 0) && item.Expires > 0 {
			sec := int64(item.Expires)
			if sec > 0 {
				expires := time.Unix(sec, 0)
				if expires.After(now) {
					cookie.Expires = expires
				}
			}
		}

		out = append(out, cookie)
	}

	return out
}

func cloneCookies(cookies []*http.Cookie) []*http.Cookie {
	result := make([]*http.Cookie, 0, len(cookies))
	for _, cookie := range cookies {
		if cookie == nil {
			continue
		}
		copy := *cookie
		result = append(result, &copy)
	}
	return result
}

func originFromURL(rawURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("missing scheme or host")
	}
	return parsed.Scheme + "://" + parsed.Host, nil
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
	sharedCookieCache = newCookieCache()
	resetBrowserRuntimeForTests()
}
