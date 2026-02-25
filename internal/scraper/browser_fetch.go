package scraper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	cdpruntime "github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

const (
	defaultBrowserBootstrapPath = "/news"
	defaultBrowserFetchTimeout  = 90 * time.Second
)

type browserFetchResponse struct {
	Status int    `json:"status"`
	URL    string `json:"url"`
	Body   string `json:"body"`
	Error  string `json:"error"`
}

type browserRuntime struct {
	mu sync.Mutex

	origin       string
	allocatorCtx context.Context
	allocatorEnd context.CancelFunc
	browserCtx   context.Context
	browserEnd   context.CancelFunc
}

func (b *browserRuntime) resetLocked() {
	if b.browserEnd != nil {
		b.browserEnd()
	}
	if b.allocatorEnd != nil {
		b.allocatorEnd()
	}
	b.origin = ""
	b.allocatorCtx = nil
	b.allocatorEnd = nil
	b.browserCtx = nil
	b.browserEnd = nil
}

func (b *browserRuntime) ensureBrowserLocked(origin string) {
	if b.browserCtx != nil && b.origin == origin {
		return
	}

	b.resetLocked()

	allocatorOptions := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.UserAgent(defaultBrowserUserAgent),
	)
	if browserExecPath := strings.TrimSpace(os.Getenv("CHROME_BIN")); browserExecPath != "" {
		allocatorOptions = append(allocatorOptions, chromedp.ExecPath(browserExecPath))
	}

	allocatorCtx, allocatorEnd := chromedp.NewExecAllocator(context.Background(), allocatorOptions...)
	browserCtx, browserEnd := chromedp.NewContext(allocatorCtx)

	b.origin = origin
	b.allocatorCtx = allocatorCtx
	b.allocatorEnd = allocatorEnd
	b.browserCtx = browserCtx
	b.browserEnd = browserEnd
}

func (b *browserRuntime) fetchJSON(ctx context.Context, apiURL, origin string) ([]byte, int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		b.ensureBrowserLocked(origin)
		body, statusCode, err := b.fetchJSONLocked(ctx, apiURL, origin)
		if err == nil {
			return body, statusCode, nil
		}
		lastErr = err
		b.resetLocked()
	}

	if lastErr == nil {
		lastErr = errors.New("browser fetch failed")
	}
	return nil, 0, lastErr
}

func (b *browserRuntime) fetchJSONLocked(ctx context.Context, apiURL, origin string) ([]byte, int, error) {
	tabCtx, tabEnd := chromedp.NewContext(b.browserCtx)
	defer tabEnd()

	runCtx, runEnd := context.WithTimeout(tabCtx, effectiveBrowserTimeout(ctx))
	defer runEnd()

	bootstrapURL := origin + defaultBrowserBootstrapPath
	if err := chromedp.Run(runCtx,
		chromedp.Navigate(bootstrapURL),
		chromedp.WaitReady("body", chromedp.ByQuery),
	); err != nil {
		return nil, 0, fmt.Errorf("browser bootstrap navigation failed: %w", err)
	}

	if err := b.waitForChallengeResolution(runCtx); err != nil {
		return nil, 0, err
	}

	script := buildBrowserFetchScript(apiURL)
	var fetchResult browserFetchResponse
	if err := chromedp.Run(runCtx,
		chromedp.Evaluate(script, &fetchResult, chromedp.EvalAsValue, awaitPromise),
	); err != nil {
		return nil, 0, fmt.Errorf("browser fetch evaluate failed: %w", err)
	}

	if strings.TrimSpace(fetchResult.Error) != "" {
		return nil, 0, fmt.Errorf("browser fetch returned JS error: %s", fetchResult.Error)
	}

	return []byte(fetchResult.Body), fetchResult.Status, nil
}

func (b *browserRuntime) waitForChallengeResolution(ctx context.Context) error {
	for attempt := 0; attempt < 20; attempt++ {
		var title string
		var location string
		if err := chromedp.Run(ctx,
			chromedp.Title(&title),
			chromedp.Location(&location),
		); err != nil {
			return fmt.Errorf("browser readiness check failed: %w", err)
		}

		lowerTitle := strings.ToLower(strings.TrimSpace(title))
		lowerLocation := strings.ToLower(strings.TrimSpace(location))
		if !strings.Contains(lowerTitle, "just a moment") && !strings.Contains(lowerLocation, "__cf_chl") {
			return nil
		}

		if err := chromedp.Run(ctx, chromedp.Sleep(1*time.Second)); err != nil {
			return fmt.Errorf("browser wait interrupted: %w", err)
		}
	}

	return errors.New("cloudflare challenge did not resolve in browser context")
}

func buildBrowserFetchScript(apiURL string) string {
	quotedURL := strconv.Quote(apiURL)
	return fmt.Sprintf(`new Promise(async (resolve) => {
  try {
    const response = await fetch(%s, {
      method: 'GET',
      credentials: 'include',
      headers: {
        'Accept': 'application/json'
      }
    });
    const body = await response.text();
    resolve({ status: response.status, url: response.url, body: body });
  } catch (err) {
    resolve({ status: 0, url: '', body: '', error: String(err) });
  }
})`, quotedURL)
}

func effectiveBrowserTimeout(parent context.Context) time.Duration {
	if deadline, ok := parent.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining > 0 && remaining < defaultBrowserFetchTimeout {
			return remaining
		}
	}
	return defaultBrowserFetchTimeout
}

func awaitPromise(params *cdpruntime.EvaluateParams) *cdpruntime.EvaluateParams {
	return params.WithAwaitPromise(true)
}

var sharedBrowserRuntime = &browserRuntime{}

func (c *Client) fetchJSONViaBrowser(ctx context.Context, apiURL, origin string) ([]byte, int, error) {
	var (
		lastBody   []byte
		lastStatus int
		lastErr    error
	)

	for attempt := 0; attempt < 3; attempt++ {
		body, statusCode, err := sharedBrowserRuntime.fetchJSON(ctx, apiURL, origin)
		lastBody = body
		lastStatus = statusCode
		lastErr = err

		if err != nil {
			time.Sleep(300 * time.Millisecond)
			continue
		}
		if statusCode >= http.StatusInternalServerError || statusCode == 0 {
			time.Sleep(300 * time.Millisecond)
			continue
		}

		return body, statusCode, nil
	}

	if lastErr != nil {
		return nil, 0, lastErr
	}

	var envelope map[string]any
	if json.Unmarshal(lastBody, &envelope) == nil {
		if errorMessage, ok := envelope["error"].(string); ok && strings.TrimSpace(errorMessage) != "" {
			return lastBody, lastStatus, nil
		}
	}

	return lastBody, lastStatus, nil
}

func resetBrowserRuntimeForTests() {
	sharedBrowserRuntime.mu.Lock()
	defer sharedBrowserRuntime.mu.Unlock()
	sharedBrowserRuntime.resetLocked()
}
