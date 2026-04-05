package scraper

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

const maxFetchRetries = 3

type cacheEntry struct {
	value     string
	expiresAt time.Time
}

type CachedFetcher struct {
	pool       *CDPPool
	group      singleflight.Group
	cache      sync.Map
	ttl        time.Duration
	warmMu     sync.Mutex
	lastWarmAt time.Time
	cfBlocked  atomic.Bool
}

func NewCachedFetcher(pool *CDPPool, ttl time.Duration) *CachedFetcher {
	return &CachedFetcher{pool: pool, ttl: ttl}
}

func (f *CachedFetcher) IsReady() bool {
	return !f.cfBlocked.Load()
}

const reWarmCooldown = 90 * time.Second

func (f *CachedFetcher) triggerReWarm() {
	f.cfBlocked.Store(true)

	if !f.warmMu.TryLock() {
		return
	}

	if time.Since(f.lastWarmAt) < reWarmCooldown {
		f.warmMu.Unlock()
		log.Printf("[cdp-pool] skipping re-warm, last warm was %s ago (cooldown %s)", time.Since(f.lastWarmAt).Round(time.Second), reWarmCooldown)
		return
	}

	go func() {
		defer f.warmMu.Unlock()
		warmCtx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()
		log.Printf("[cdp-pool] triggering background FlareSolverr re-warm (pod marked not-ready)")
		if err := f.pool.warmFlareSolverrSession(warmCtx); err != nil {
			log.Printf("[cdp-pool] background re-warm failed: %v", err)
		} else {
			f.lastWarmAt = time.Now()
			f.cfBlocked.Store(false)
			log.Printf("[cdp-pool] background re-warm succeeded (pod marked ready)")
		}
	}()
}

func (f *CachedFetcher) SetLastWarmAt(t time.Time) {
	f.lastWarmAt = t
}

func (f *CachedFetcher) FetchJSON(ctx context.Context, apiURL string) (string, error) {
	cacheKey, err := apiPathFromURL(apiURL)
	if err != nil {
		return "", err
	}

	if entry, ok := f.cache.Load(cacheKey); ok {
		ce := entry.(*cacheEntry)
		if time.Now().Before(ce.expiresAt) {
			CacheRequests.WithLabelValues("hit").Inc()
			return ce.value, nil
		}
		f.cache.Delete(cacheKey)
	}
	CacheRequests.WithLabelValues("miss").Inc()

	result, err, shared := f.group.Do(cacheKey, func() (interface{}, error) {
		var lastErr error
		for attempt := 0; attempt < maxFetchRetries; attempt++ {
			tab, idx, acquireErr := f.pool.Acquire(ctx)
			if acquireErr != nil {
				return nil, acquireErr
			}

			started := time.Now()
			body, fetchErr := tab.Fetch(ctx, cacheKey)
			CDPFetchDuration.Observe(time.Since(started).Seconds())
			f.pool.Release(idx)

			if fetchErr != nil {
				lastErr = fetchErr
				CDPFetchRequests.WithLabelValues("error").Inc()

				errMsg := fetchErr.Error()
				isCF := (strings.Contains(errMsg, "HTTP 403") && strings.Contains(errMsg, "Just a moment")) ||
					(strings.Contains(errMsg, "HTTP 403") && strings.Contains(errMsg, "Access denied")) ||
					strings.Contains(errMsg, "Failed to fetch") ||
					strings.Contains(errMsg, "CDP not connected")
				if isCF {
					CDPFetchRequests.WithLabelValues("cf_challenge").Inc()
					log.Printf("[cdp-pool] session issue detected on fetch for %s: %s", cacheKey, errMsg[:min(len(errMsg), 80)])
					f.triggerReWarm()
					time.Sleep(3 * time.Second)
				}

				if attempt < maxFetchRetries-1 {
					log.Printf("[retry] FetchJSON attempt %d/%d failed for %s: %v", attempt+1, maxFetchRetries, cacheKey, fetchErr)
					select {
					case <-ctx.Done():
						return nil, ctx.Err()
					case <-time.After(200 * time.Millisecond):
					}
				}
				continue
			}

			trimmed := strings.TrimSpace(body)
			if len(trimmed) == 0 || (trimmed[0] != '{' && trimmed[0] != '[') {
				CDPFetchRequests.WithLabelValues("non_json").Inc()
				preview := trimmed
				if len(preview) > 200 {
					preview = preview[:200]
				}

				if strings.Contains(trimmed, "Just a moment") || strings.Contains(trimmed, "cf-browser-verification") {
					CDPFetchRequests.WithLabelValues("cf_challenge").Inc()
					log.Printf("[cdp-pool] Cloudflare challenge detected on fetch for %s, re-warming FlareSolverr session", cacheKey)
					if warmErr := f.pool.warmFlareSolverrSession(ctx); warmErr != nil {
						log.Printf("[cdp-pool] FlareSolverr re-warm failed: %v", warmErr)
					}
				}

				lastErr = fmt.Errorf("CDP returned non-JSON response for %s: %s", cacheKey, preview)
				if attempt < maxFetchRetries-1 {
					log.Printf("[retry] FetchJSON attempt %d/%d non-JSON for %s: %s", attempt+1, maxFetchRetries, cacheKey, preview)
					select {
					case <-ctx.Done():
						return nil, ctx.Err()
					case <-time.After(200 * time.Millisecond):
					}
				}
				continue
			}

			CDPFetchRequests.WithLabelValues("ok").Inc()
			UpstreamStatus.WithLabelValues(endpointFromURL(apiURL), "200").Inc()

			f.cache.Store(cacheKey, &cacheEntry{
				value:     body,
				expiresAt: time.Now().Add(f.ttl),
			})
			return body, nil
		}
		return nil, lastErr
	})

	if shared {
		SingleflightDedup.Inc()
	}

	if err != nil {
		return "", err
	}
	return result.(string), nil
}

func (f *CachedFetcher) FetchBinary(ctx context.Context, apiPath string) ([]byte, string, error) {
	tab, idx, err := f.pool.Acquire(ctx)
	if err != nil {
		return nil, "", err
	}
	defer f.pool.Release(idx)

	body, _, contentType, fetchErr := tab.FetchBinary(ctx, apiPath)
	if fetchErr != nil {
		return nil, "", fetchErr
	}
	return body, contentType, nil
}

func (f *CachedFetcher) BatchFetchJSON(ctx context.Context, apiURLs []string) (map[string]string, error) {
	results := make(map[string]string, len(apiURLs))
	pending := make([]string, 0)
	pendingKeys := make([]string, 0)

	for _, apiURL := range apiURLs {
		key, keyErr := apiPathFromURL(apiURL)
		if keyErr != nil {
			return nil, keyErr
		}

		if entry, ok := f.cache.Load(key); ok {
			ce := entry.(*cacheEntry)
			if time.Now().Before(ce.expiresAt) {
				CacheRequests.WithLabelValues("hit").Inc()
				results[apiURL] = ce.value
				continue
			}
			f.cache.Delete(key)
		}
		CacheRequests.WithLabelValues("miss").Inc()
		pending = append(pending, apiURL)
		pendingKeys = append(pendingKeys, key)
	}

	if len(pending) == 0 {
		return results, nil
	}

	var lastErr error
	for attempt := 0; attempt < maxFetchRetries; attempt++ {
		tab, idx, err := f.pool.Acquire(ctx)
		if err != nil {
			return nil, err
		}

		started := time.Now()
		batchResults, err := tab.BatchFetch(ctx, pendingKeys)
		CDPFetchDuration.Observe(time.Since(started).Seconds())
		f.pool.Release(idx)

		if err != nil {
			lastErr = err
			CDPFetchRequests.WithLabelValues("error").Add(float64(len(pending)))
			if attempt < maxFetchRetries-1 {
				log.Printf("[retry] BatchFetchJSON attempt %d/%d failed for %d URLs: %v", attempt+1, maxFetchRetries, len(pending), err)
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(200 * time.Millisecond):
				}
			}
			continue
		}

		batchOK := true
		for i, br := range batchResults {
			if br.Status == "fulfilled" {
				trimmed := strings.TrimSpace(br.Value)
				if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
					CDPFetchRequests.WithLabelValues("ok").Inc()
					UpstreamStatus.WithLabelValues(endpointFromURL(pending[i]), "200").Inc()
					f.cache.Store(pendingKeys[i], &cacheEntry{
						value:     br.Value,
						expiresAt: time.Now().Add(f.ttl),
					})
					results[pending[i]] = br.Value
				} else {
					CDPFetchRequests.WithLabelValues("non_json").Inc()
					preview := trimmed
					if len(preview) > 200 {
						preview = preview[:200]
					}
					lastErr = fmt.Errorf("CDP batch non-JSON for %s: %s", pendingKeys[i], preview)
					batchOK = false
					break
				}
			} else {
				CDPFetchRequests.WithLabelValues("error").Inc()
				lastErr = fmt.Errorf("CDP batch item failed for %s: %s", pendingKeys[i], br.Value)
				batchOK = false
				break
			}
		}

		if batchOK {
			return results, nil
		}

		for _, key := range pending {
			delete(results, key)
		}
		if attempt < maxFetchRetries-1 {
			log.Printf("[retry] BatchFetchJSON attempt %d/%d partial failure: %v", attempt+1, maxFetchRetries, lastErr)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(200 * time.Millisecond):
			}
		}
	}

	return nil, lastErr
}
