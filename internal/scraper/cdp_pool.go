package scraper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type CDPPool struct {
	mu              sync.Mutex
	tabs            []*CDPClient
	available       chan int
	cdpURL          string
	siteURL         string
	flareSolverrURL string
	size            int
}

func NewCDPPool(cdpURL, siteURL string, _ int) *CDPPool {
	return &CDPPool{
		cdpURL:          strings.TrimRight(cdpURL, "/"),
		siteURL:         strings.TrimRight(siteURL, "/"),
		flareSolverrURL: os.Getenv("FLARESOLVERR_URL"),
		size:            1,
		available:        make(chan int, 1),
	}
}

func (p *CDPPool) warmFlareSolverrSession(ctx context.Context) error {
	if p.flareSolverrURL == "" {
		return nil
	}

	createBody, _ := json.Marshal(map[string]string{
		"cmd":     "sessions.create",
		"session": "rubinot-cdp",
	})
	req, err := http.NewRequestWithContext(ctx, "POST", p.flareSolverrURL, bytes.NewReader(createBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("flaresolverr session create: %w", err)
	}
	resp.Body.Close()

	warmBody, _ := json.Marshal(map[string]interface{}{
		"cmd":               "request.get",
		"url":               p.siteURL + "/",
		"session":           "rubinot-cdp",
		"maxTimeout":        120000,
		"disableMedia":      true,
		"session_ttl_minutes": 30,
	})
	req2, err := http.NewRequestWithContext(ctx, "POST", p.flareSolverrURL, bytes.NewReader(warmBody))
	if err != nil {
		return err
	}
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		return fmt.Errorf("flaresolverr session warm: %w", err)
	}
	resp2.Body.Close()

	log.Printf("[cdp-pool] FlareSolverr session warmed for %s", p.siteURL)
	return nil
}

func (p *CDPPool) Init(ctx context.Context) error {
	var warmErr error
	for attempt := 0; attempt < 10; attempt++ {
		warmErr = p.warmFlareSolverrSession(ctx)
		if warmErr == nil {
			break
		}
		wait := time.Duration(attempt+1) * 3 * time.Second
		log.Printf("[cdp-pool] FlareSolverr warm attempt %d/10 failed (waiting %s): %v", attempt+1, wait, warmErr)
		time.Sleep(wait)
	}
	if warmErr != nil {
		return fmt.Errorf("FlareSolverr session warm failed after 10 attempts: %w", warmErr)
	}

	discovery := NewCDPClient(p.cdpURL)
	defaultWSURL, err := discovery.DiscoverPageTarget(ctx)
	if err != nil {
		return fmt.Errorf("discover default page target: %w", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.tabs = make([]*CDPClient, p.size)

	tab0 := NewCDPClient(p.cdpURL)
	if err := tab0.ConnectToURL(ctx, defaultWSURL); err != nil {
		return fmt.Errorf("connect tab 0: %w", err)
	}
	p.tabs[0] = tab0
	p.available <- 0

	log.Printf("[cdp-pool] initialized with 1 tab (FlareSolverr-cleared tab only)")
	return nil
}

func (p *CDPPool) createTab(ctx context.Context, creator *CDPClient, idx int) (*CDPClient, error) {
	targetID, err := creator.CreateTarget(ctx, "about:blank")
	if err != nil {
		return nil, fmt.Errorf("create tab %d: %w", idx, err)
	}

	wsURL := fmt.Sprintf("ws://%s/devtools/page/%s", extractHost(p.cdpURL), targetID)
	tab := NewCDPClient(p.cdpURL)
	if err := tab.ConnectToURL(ctx, wsURL); err != nil {
		return nil, fmt.Errorf("connect tab %d: %w", idx, err)
	}

	if err := tab.Navigate(ctx, p.siteURL+"/"); err != nil {
		tab.Close()
		return nil, fmt.Errorf("navigate tab %d: %w", idx, err)
	}

	return tab, nil
}

func (p *CDPPool) Acquire(ctx context.Context) (*CDPClient, int, error) {
	select {
	case idx := <-p.available:
		p.mu.Lock()
		tab := p.tabs[idx]
		p.mu.Unlock()

		CDPPoolAvailable.Set(float64(len(p.available)))
		if tab == nil || !tab.IsConnected() {
			CDPPoolRebuilds.Inc()
			rebuilt, err := p.rebuildTab(ctx, idx)
			if err != nil {
				p.available <- idx
				return nil, 0, fmt.Errorf("rebuild tab %d: %w", idx, err)
			}
			return rebuilt, idx, nil
		}
		return tab, idx, nil
	case <-ctx.Done():
		return nil, 0, ctx.Err()
	}
}

func (p *CDPPool) Release(idx int) {
	p.available <- idx
	CDPPoolAvailable.Set(float64(len(p.available)))
}

func (p *CDPPool) rebuildTab(ctx context.Context, idx int) (*CDPClient, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.tabs[idx] != nil {
		p.tabs[idx].Close()
	}

	log.Printf("[cdp-pool] rebuilding tab %d — reconnecting to FlareSolverr tab", idx)

	if err := p.warmFlareSolverrSession(ctx); err != nil {
		log.Printf("[cdp-pool] FlareSolverr warm during rebuild failed: %v", err)
	}

	discovery := NewCDPClient(p.cdpURL)
	wsURL, err := discovery.DiscoverPageTarget(ctx)
	if err != nil {
		return nil, fmt.Errorf("discover page target for rebuild: %w", err)
	}

	tab := NewCDPClient(p.cdpURL)
	if err := tab.ConnectToURL(ctx, wsURL); err != nil {
		return nil, fmt.Errorf("connect rebuilt tab: %w", err)
	}

	p.tabs[idx] = tab
	return tab, nil
}

func (p *CDPPool) recoverBaseTab(ctx context.Context) (*CDPClient, error) {
	if err := p.warmFlareSolverrSession(ctx); err != nil {
		log.Printf("[cdp-pool] FlareSolverr warm during recovery failed: %v", err)
	}

	discovery := NewCDPClient(p.cdpURL)
	wsURL, err := discovery.DiscoverPageTarget(ctx)
	if err != nil {
		return nil, fmt.Errorf("discover page target: %w", err)
	}

	tab0 := NewCDPClient(p.cdpURL)
	if err := tab0.ConnectToURL(ctx, wsURL); err != nil {
		return nil, fmt.Errorf("connect recovered tab: %w", err)
	}

	for i, t := range p.tabs {
		if t == nil || !t.IsConnected() {
			p.tabs[i] = tab0
			return tab0, nil
		}
	}

	p.tabs[0] = tab0
	return tab0, nil
}

func (p *CDPPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, tab := range p.tabs {
		if tab != nil {
			tab.Close()
		}
	}
}

func (p *CDPPool) AvailableCount() int {
	return len(p.available)
}

func extractHost(cdpURL string) string {
	host := cdpURL
	host = strings.TrimPrefix(host, "ws://")
	host = strings.TrimPrefix(host, "wss://")
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimRight(host, "/")
	return host
}
