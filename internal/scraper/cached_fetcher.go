package scraper

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

type cacheEntry struct {
	value     string
	expiresAt time.Time
}

type CachedFetcher struct {
	pool  *CDPPool
	group singleflight.Group
	cache sync.Map
	ttl   time.Duration
}

func NewCachedFetcher(pool *CDPPool, ttl time.Duration) *CachedFetcher {
	return &CachedFetcher{pool: pool, ttl: ttl}
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
		tab, idx, acquireErr := f.pool.Acquire(ctx)
		if acquireErr != nil {
			return nil, acquireErr
		}
		defer f.pool.Release(idx)

		started := time.Now()
		body, fetchErr := tab.Fetch(ctx, cacheKey)
		CDPFetchDuration.Observe(time.Since(started).Seconds())

		if fetchErr != nil {
			CDPFetchRequests.WithLabelValues("error").Inc()
			return nil, fetchErr
		}

		trimmed := strings.TrimSpace(body)
		if len(trimmed) == 0 || (trimmed[0] != '{' && trimmed[0] != '[') {
			CDPFetchRequests.WithLabelValues("non_json").Inc()
			return nil, fmt.Errorf("CDP returned non-JSON response for %s", cacheKey)
		}

		CDPFetchRequests.WithLabelValues("ok").Inc()
		UpstreamStatus.WithLabelValues(endpointFromURL(apiURL), "200").Inc()

		f.cache.Store(cacheKey, &cacheEntry{
			value:     body,
			expiresAt: time.Now().Add(f.ttl),
		})
		return body, nil
	})

	if shared {
		SingleflightDedup.Inc()
	}

	if err != nil {
		return "", err
	}
	return result.(string), nil
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

	tab, idx, err := f.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer f.pool.Release(idx)

	started := time.Now()
	batchResults, err := tab.BatchFetch(ctx, pendingKeys)
	CDPFetchDuration.Observe(time.Since(started).Seconds())

	if err != nil {
		CDPFetchRequests.WithLabelValues("error").Add(float64(len(pending)))
		return nil, err
	}

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
				return nil, fmt.Errorf("CDP returned non-JSON for %s", pendingKeys[i])
			}
		} else {
			CDPFetchRequests.WithLabelValues("error").Inc()
			return nil, fmt.Errorf("CDP batch item failed for %s: %s", pendingKeys[i], br.Value)
		}
	}

	return results, nil
}
