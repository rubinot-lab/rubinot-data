package scraper

import (
	"context"
)

type OptimizedClient struct {
	Fetcher *CachedFetcher
}

func NewOptimizedClient(fetcher *CachedFetcher) *OptimizedClient {
	return &OptimizedClient{Fetcher: fetcher}
}

func (c *OptimizedClient) FetchJSON(ctx context.Context, apiURL string, result any) error {
	body, err := c.Fetcher.FetchJSON(ctx, apiURL)
	if err != nil {
		return err
	}
	return parseJSONBody(body, result)
}

func (c *OptimizedClient) BatchFetchJSON(ctx context.Context, apiURLs []string) (map[string]string, error) {
	return c.Fetcher.BatchFetchJSON(ctx, apiURLs)
}
