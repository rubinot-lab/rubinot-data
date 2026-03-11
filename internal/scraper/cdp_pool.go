package scraper

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

type CDPPool struct {
	mu        sync.Mutex
	tabs      []*CDPClient
	available chan int
	cdpURL    string
	siteURL   string
	size      int
}

func NewCDPPool(cdpURL, siteURL string, size int) *CDPPool {
	return &CDPPool{
		cdpURL:    strings.TrimRight(cdpURL, "/"),
		siteURL:   strings.TrimRight(siteURL, "/"),
		size:      size,
		available: make(chan int, size),
	}
}

func (p *CDPPool) Init(ctx context.Context) error {
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

	for i := 1; i < p.size; i++ {
		tab, err := p.createTab(ctx, tab0, i)
		if err != nil {
			return err
		}
		p.tabs[i] = tab
		p.available <- i
	}

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

		if tab == nil || !tab.IsConnected() {
			rebuilt, err := p.rebuildTab(ctx, idx)
			if err != nil {
				p.available <- idx
				CDPPoolRebuilds.Inc()
				return nil, 0, fmt.Errorf("rebuild tab %d: %w", idx, err)
			}
			CDPPoolRebuilds.Inc()
			return rebuilt, idx, nil
		}
		return tab, idx, nil
	case <-ctx.Done():
		return nil, 0, ctx.Err()
	}
}

func (p *CDPPool) Release(idx int) {
	p.available <- idx
}

func (p *CDPPool) rebuildTab(ctx context.Context, idx int) (*CDPClient, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.tabs[idx] != nil {
		p.tabs[idx].Close()
	}

	var creator *CDPClient
	for _, t := range p.tabs {
		if t != nil && t.IsConnected() {
			creator = t
			break
		}
	}
	if creator == nil {
		return nil, fmt.Errorf("no healthy tabs available to create target")
	}

	tab, err := p.createTab(ctx, creator, idx)
	if err != nil {
		return nil, err
	}

	p.tabs[idx] = tab
	return tab, nil
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
