package scraper

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	rubinidataDefaultURL    = "https://api.rubinidata.com"
	rubinidataMaxRetries    = 3
	rubinidataRequestTimeout = 30 * time.Second
)

type RubinidataClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewRubinidataClient(baseURL, apiKey string) *RubinidataClient {
	return &RubinidataClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: rubinidataRequestTimeout},
	}
}

func NewRubinidataClientFromEnv() *RubinidataClient {
	baseURL := os.Getenv("RUBINIDATA_URL")
	if baseURL == "" {
		baseURL = rubinidataDefaultURL
	}
	return NewRubinidataClient(baseURL, os.Getenv("RUBINIDATA_API_KEY"))
}

func (c *RubinidataClient) Fetch(ctx context.Context, upstreamURL string) (string, error) {
	translatedPath, err := translatePath(upstreamURL)
	if err != nil {
		return "", fmt.Errorf("translate path: %w", err)
	}

	var lastErr error
	for attempt := range rubinidataMaxRetries {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
			}
		}

		body, statusCode, fetchErr := c.doGet(ctx, translatedPath)
		if fetchErr != nil {
			lastErr = fetchErr
			continue
		}

		if statusCode >= 500 {
			lastErr = fmt.Errorf("rubinidata returned HTTP %d", statusCode)
			continue
		}

		upstreamPath, _ := apiPathFromURL(upstreamURL)
		adapted, adaptErr := adaptResponse(upstreamPath, body)
		if adaptErr != nil {
			return "", fmt.Errorf("adapt response: %w", adaptErr)
		}

		return adapted, nil
	}

	return "", fmt.Errorf("rubinidata fetch failed after %d retries: %w", rubinidataMaxRetries, lastErr)
}

const batchConcurrency = 10

func (c *RubinidataClient) BatchFetch(ctx context.Context, paths []string) (map[string]string, error) {
	results := make(map[string]string, len(paths))
	var mu sync.Mutex
	var firstErr error
	sem := make(chan struct{}, batchConcurrency)
	var wg sync.WaitGroup

	for _, p := range paths {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			body, err := c.Fetch(ctx, path)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if firstErr == nil {
					firstErr = fmt.Errorf("batch fetch %s: %w", path, err)
				}
				return
			}
			results[path] = body
		}(p)
	}
	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}
	return results, nil
}

func (c *RubinidataClient) FetchBinary(ctx context.Context, upstreamPath string) ([]byte, string, error) {
	translatedPath, err := translatePath(upstreamPath)
	if err != nil {
		return nil, "", fmt.Errorf("translate path: %w", err)
	}

	reqURL := c.baseURL + translatedPath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("HTTP GET %s: %w", reqURL, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read response body: %w", err)
	}

	return raw, resp.Header.Get("Content-Type"), nil
}

func (c *RubinidataClient) doGet(ctx context.Context, path string) (string, int, error) {
	reqURL := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("HTTP GET %s: %w", reqURL, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, fmt.Errorf("read response body: %w", err)
	}

	return string(raw), resp.StatusCode, nil
}

var rubinidataVocationMap = map[string]string{
	"0": "0",
	"1": "0",
	"2": "4",
	"3": "2",
	"4": "3",
	"5": "1",
	"9": "5",
}

func IsRubinidataProvider() bool {
	return strings.EqualFold(os.Getenv("UPSTREAM_PROVIDER"), "rubinidata")
}

func translateVocationID(rubinotID string) string {
	if mapped, ok := rubinidataVocationMap[rubinotID]; ok {
		return mapped
	}
	return rubinotID
}

func resolveWorldName(rawID string) string {
	id, err := strconv.Atoi(rawID)
	if err != nil {
		return rawID
	}
	if name := worldNameByID(id); name != "" {
		return name
	}
	return rawID
}

func translatePath(upstreamURL string) (string, error) {
	if upstreamURL == "" {
		return "", fmt.Errorf("empty upstream URL")
	}

	parsed, err := url.Parse(upstreamURL)
	if err != nil {
		return "", fmt.Errorf("parse upstream URL: %w", err)
	}

	path := parsed.Path
	query := parsed.Query()

	switch {
	case path == "/api/worlds":
		return "/v1/worlds", nil

	case strings.HasPrefix(path, "/api/worlds/"):
		name := strings.TrimPrefix(path, "/api/worlds/")
		return "/v1/world/" + name, nil

	case path == "/api/characters/search":
		name := query.Get("name")
		if name == "" {
			return "", fmt.Errorf("character search requires name parameter")
		}
		return "/v1/characters/" + url.PathEscape(name), nil

	case path == "/api/guilds" && query.Get("world") != "":
		worldName := resolveWorldName(query.Get("world"))
		result := "/v1/guilds/" + url.PathEscape(worldName)
		if page := query.Get("page"); page != "" {
			result += "?page=" + page
		}
		return result, nil

	case path == "/api/guilds":
		return "", fmt.Errorf("guilds list requires world parameter")

	case strings.HasPrefix(path, "/api/guilds/"):
		name := strings.TrimPrefix(path, "/api/guilds/")
		return "/v1/guild/" + url.PathEscape(name), nil

	case path == "/api/highscores/categories":
		return "/v1/highscores/categories", nil

	case path == "/api/highscores":
		worldName := resolveWorldName(query.Get("world"))
		params := url.Values{}
		params.Set("world", worldName)
		if cat := query.Get("category"); cat != "" {
			params.Set("category", cat)
		}
		if voc := query.Get("vocation"); voc != "" {
			params.Set("vocation", translateVocationID(voc))
		}
		return "/v1/highscores?" + params.Encode(), nil

	case path == "/api/killstats":
		worldName := resolveWorldName(query.Get("world"))
		return "/v1/killstatistics/" + url.PathEscape(worldName), nil

	case path == "/api/deaths":
		worldName := resolveWorldName(query.Get("world"))
		result := "/v1/deaths/" + url.PathEscape(worldName)
		if page := query.Get("page"); page != "" {
			result += "?page=" + page
		}
		return result, nil

	case path == "/api/bans":
		worldName := resolveWorldName(query.Get("world"))
		result := "/v1/banishments/" + url.PathEscape(worldName)
		if page := query.Get("page"); page != "" {
			result += "?page=" + page
		}
		return result, nil

	case path == "/api/transfers":
		if page := query.Get("page"); page != "" {
			return "/v1/transfers?page=" + page, nil
		}
		return "/v1/transfers", nil

	case path == "/api/boosted":
		return "/v1/boosted", nil

	case path == "/api/outfit":
		result := "/v1/outfit"
		if raw := parsed.RawQuery; raw != "" {
			result += "?" + raw
		}
		return result, nil

	default:
		return "", fmt.Errorf("unrecognized upstream path: %s", path)
	}
}
// rubinidata bridge v2.8.0
