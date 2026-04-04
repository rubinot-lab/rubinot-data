package scraper

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type cdpTarget struct {
	ID                   string `json:"id"`
	Type                 string `json:"type"`
	URL                  string `json:"url"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

type cdpRequest struct {
	ID     int64  `json:"id"`
	Method string `json:"method"`
	Params any    `json:"params"`
}

type cdpResponse struct {
	ID     int64             `json:"id"`
	Result json.RawMessage   `json:"result,omitempty"`
	Error  *cdpResponseError `json:"error,omitempty"`
}

type cdpResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type cdpEvalResult struct {
	Result struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	} `json:"result"`
	ExceptionDetails json.RawMessage `json:"exceptionDetails,omitempty"`
}

type BatchResult struct {
	Status string `json:"status"`
	Value  string `json:"value"`
}

type cdpBinaryResult struct {
	Status      int    `json:"status"`
	ContentType string `json:"contentType"`
	BodyBase64  string `json:"bodyBase64"`
}

type CDPClient struct {
	sem     chan struct{}
	conn    *websocket.Conn
	baseURL string
	nextID  atomic.Int64
}

func NewCDPClient(baseURL string) *CDPClient {
	sem := make(chan struct{}, 1)
	return &CDPClient{baseURL: strings.TrimRight(baseURL, "/"), sem: sem}
}

func (c *CDPClient) acquireSem(ctx context.Context) error {
	select {
	case c.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *CDPClient) acquireSemBlocking() {
	c.sem <- struct{}{}
}

func (c *CDPClient) releaseSem() {
	<-c.sem
}

func (c *CDPClient) httpBaseURL() string {
	u := c.baseURL
	u = strings.Replace(u, "ws://", "http://", 1)
	u = strings.Replace(u, "wss://", "https://", 1)
	return u
}

func (c *CDPClient) discoverPageTarget(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.httpBaseURL()+"/json/list", nil)
	if err != nil {
		return "", fmt.Errorf("build CDP target request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch CDP targets: %w", err)
	}
	defer resp.Body.Close()

	var targets []cdpTarget
	if err := json.NewDecoder(resp.Body).Decode(&targets); err != nil {
		return "", fmt.Errorf("decode CDP targets: %w", err)
	}

	for _, t := range targets {
		if t.Type == "page" {
			return t.WebSocketDebuggerURL, nil
		}
	}
	return "", fmt.Errorf("no page target found among %d CDP targets", len(targets))
}

func (c *CDPClient) Connect(ctx context.Context) error {
	if err := c.acquireSem(ctx); err != nil {
		return err
	}
	defer c.releaseSem()

	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}

	wsURL, err := c.discoverPageTarget(ctx)
	if err != nil {
		return err
	}

	parsed, err := url.Parse(wsURL)
	if err != nil {
		return fmt.Errorf("parse CDP WebSocket URL: %w", err)
	}
	baseHost := strings.TrimPrefix(c.baseURL, "ws://")
	baseHost = strings.TrimPrefix(baseHost, "http://")
	parsed.Host = baseHost
	wsURL = parsed.String()

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}
	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("dial CDP WebSocket %s: %w", wsURL, err)
	}
	conn.SetReadLimit(10 * 1024 * 1024)

	c.conn = conn
	return nil
}

func (c *CDPClient) Evaluate(ctx context.Context, expression string) (string, error) {
	if err := c.acquireSem(ctx); err != nil {
		return "", fmt.Errorf("CDP lock cancelled: %w", err)
	}
	defer c.releaseSem()

	if c.conn == nil {
		return "", fmt.Errorf("CDP not connected")
	}

	id := c.nextID.Add(1)
	msg := cdpRequest{
		ID:     id,
		Method: "Runtime.evaluate",
		Params: map[string]any{
			"expression":    expression,
			"awaitPromise":  true,
			"returnByValue": true,
		},
	}

	if err := c.conn.WriteJSON(msg); err != nil {
		c.conn.Close()
		c.conn = nil
		return "", fmt.Errorf("write CDP message: %w", err)
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(30 * time.Second)
	}
	c.conn.SetReadDeadline(deadline)

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			c.conn.Close()
			c.conn = nil
			return "", fmt.Errorf("read CDP response: %w", err)
		}

		var resp cdpResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			continue
		}
		if resp.ID != id {
			continue
		}
		if resp.Error != nil {
			return "", fmt.Errorf("CDP error %d: %s", resp.Error.Code, resp.Error.Message)
		}

		var evalResult cdpEvalResult
		if err := json.Unmarshal(resp.Result, &evalResult); err != nil {
			return "", fmt.Errorf("decode eval result: %w", err)
		}
		if len(evalResult.ExceptionDetails) > 0 {
			return "", fmt.Errorf("JS exception: %s", string(evalResult.ExceptionDetails))
		}

		return evalResult.Result.Value, nil
	}
}

type cdpFetchResult struct {
	OK     bool   `json:"ok"`
	Status int    `json:"status"`
	Body   string `json:"body"`
}

func (c *CDPClient) Fetch(ctx context.Context, apiPath string) (string, error) {
	escaped := strings.ReplaceAll(apiPath, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "'", "\\'")

	expression := fmt.Sprintf(`
		(async () => {
			for (let attempt = 0; attempt < 2; attempt++) {
				try {
					const resp = await fetch('%s');
					const text = await resp.text();
					if (resp.ok) {
						return JSON.stringify({ ok: true, status: resp.status, body: text });
					}
					if ((resp.status === 429 || resp.status >= 500) && attempt === 0) {
						await new Promise(r => setTimeout(r, 300));
						continue;
					}
					return JSON.stringify({ ok: false, status: resp.status, body: text.substring(0, 500) });
				} catch (e) {
					if (attempt === 0) {
						await new Promise(r => setTimeout(r, 300));
						continue;
					}
					return JSON.stringify({ ok: false, status: 0, body: 'fetch error: ' + String(e.message || e) });
				}
			}
			return JSON.stringify({ ok: false, status: 0, body: 'retries exhausted' });
		})()
	`, escaped)

	raw, err := c.Evaluate(ctx, expression)
	if err != nil {
		return "", err
	}

	var result cdpFetchResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return "", fmt.Errorf("decode CDP fetch wrapper for %s: %w", apiPath, err)
	}

	if !result.OK {
		preview := result.Body
		if len(preview) > 200 {
			preview = preview[:200]
		}
		return "", fmt.Errorf("upstream HTTP %d for %s: %s", result.Status, apiPath, preview)
	}

	return result.Body, nil
}

func (c *CDPClient) FetchBinary(ctx context.Context, apiPath string) ([]byte, int, string, error) {
	escaped := strings.ReplaceAll(apiPath, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "'", "\\'")

	expression := fmt.Sprintf(`
		(async () => {
			const resp = await fetch('%s');
			const buffer = await resp.arrayBuffer();
			const bytes = new Uint8Array(buffer);
			const chunkSize = 0x8000;
			let binary = '';
			for (let i = 0; i < bytes.length; i += chunkSize) {
				binary += String.fromCharCode.apply(null, bytes.subarray(i, i + chunkSize));
			}
			return JSON.stringify({
				status: resp.status,
				contentType: resp.headers.get('content-type') || 'application/octet-stream',
				bodyBase64: btoa(binary),
			});
		})()
	`, escaped)

	raw, err := c.Evaluate(ctx, expression)
	if err != nil {
		return nil, 0, "", err
	}

	var out cdpBinaryResult
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, 0, "", fmt.Errorf("decode CDP binary response: %w", err)
	}

	if out.ContentType == "" {
		out.ContentType = "application/octet-stream"
	}

	body := []byte{}
	if out.BodyBase64 != "" {
		decoded, err := base64.StdEncoding.DecodeString(out.BodyBase64)
		if err != nil {
			return nil, 0, "", fmt.Errorf("decode CDP binary payload: %w", err)
		}
		body = decoded
	}

	return body, out.Status, out.ContentType, nil
}

func (c *CDPClient) BatchFetch(ctx context.Context, apiPaths []string) ([]BatchResult, error) {
	if len(apiPaths) == 0 {
		return []BatchResult{}, nil
	}

	fetchCalls := make([]string, 0, len(apiPaths))
	for _, path := range apiPaths {
		escaped := strings.ReplaceAll(path, "\\", "\\\\")
		escaped = strings.ReplaceAll(escaped, "'", "\\'")
		fetchCalls = append(fetchCalls, fmt.Sprintf(`
			(async () => {
				for (let attempt = 0; attempt < 2; attempt++) {
					try {
						const resp = await fetch('%s');
						const text = await resp.text();
						if (resp.ok) return text;
						if ((resp.status === 429 || resp.status >= 500) && attempt === 0) {
							await new Promise(r => setTimeout(r, 300));
							continue;
						}
						throw new Error('HTTP ' + resp.status + ': ' + text.substring(0, 200));
					} catch (e) {
						if (attempt === 0 && !String(e.message).startsWith('HTTP ')) {
							await new Promise(r => setTimeout(r, 300));
							continue;
						}
						throw e;
					}
				}
				throw new Error('retries exhausted');
			})()`, escaped))
	}

	expression := fmt.Sprintf(`
		(async () => {
			const results = await Promise.allSettled(
				[%s]
			);
			return JSON.stringify(results.map((result) => ({
				status: result.status,
				value: result.status === 'fulfilled'
					? result.value
					: String(result.reason && result.reason.message ? result.reason.message : (result.reason || 'fetch failed')),
			})));
		})()
	`, strings.Join(fetchCalls, ","))

	raw, err := c.Evaluate(ctx, expression)
	if err != nil {
		return nil, err
	}

	var out []BatchResult
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("decode CDP batch response: %w", err)
	}
	if len(out) != len(apiPaths) {
		return nil, fmt.Errorf("invalid CDP batch response size: got %d, expected %d", len(out), len(apiPaths))
	}

	return out, nil
}

func (c *CDPClient) ConnectToURL(ctx context.Context, wsURL string) error {
	if err := c.acquireSem(ctx); err != nil {
		return err
	}
	defer c.releaseSem()

	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}

	parsed, err := url.Parse(wsURL)
	if err != nil {
		return fmt.Errorf("parse CDP WebSocket URL: %w", err)
	}
	baseHost := strings.TrimPrefix(c.baseURL, "ws://")
	baseHost = strings.TrimPrefix(baseHost, "http://")
	parsed.Host = baseHost
	wsURL = parsed.String()

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}
	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("dial CDP WebSocket %s: %w", wsURL, err)
	}
	conn.SetReadLimit(10 * 1024 * 1024)

	c.conn = conn
	return nil
}

func (c *CDPClient) CreateTarget(ctx context.Context, targetURL string) (string, error) {
	if err := c.acquireSem(ctx); err != nil {
		return "", err
	}
	defer c.releaseSem()

	if c.conn == nil {
		return "", fmt.Errorf("CDP not connected")
	}

	id := c.nextID.Add(1)
	msg := cdpRequest{
		ID:     id,
		Method: "Target.createTarget",
		Params: map[string]any{
			"url": targetURL,
		},
	}

	if err := c.conn.WriteJSON(msg); err != nil {
		c.conn.Close()
		c.conn = nil
		return "", fmt.Errorf("write CDP createTarget: %w", err)
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(30 * time.Second)
	}
	c.conn.SetReadDeadline(deadline)

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			c.conn.Close()
			c.conn = nil
			return "", fmt.Errorf("read CDP createTarget response: %w", err)
		}

		var resp cdpResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			continue
		}
		if resp.ID != id {
			continue
		}
		if resp.Error != nil {
			return "", fmt.Errorf("CDP createTarget error %d: %s", resp.Error.Code, resp.Error.Message)
		}

		var result struct {
			TargetID string `json:"targetId"`
		}
		if err := json.Unmarshal(resp.Result, &result); err != nil {
			return "", fmt.Errorf("decode createTarget result: %w", err)
		}
		return result.TargetID, nil
	}
}

func (c *CDPClient) Navigate(ctx context.Context, pageURL string) error {
	if err := c.acquireSem(ctx); err != nil {
		return err
	}
	defer c.releaseSem()

	if c.conn == nil {
		return fmt.Errorf("CDP not connected")
	}

	id := c.nextID.Add(1)
	msg := cdpRequest{
		ID:     id,
		Method: "Page.navigate",
		Params: map[string]any{
			"url": pageURL,
		},
	}

	if err := c.conn.WriteJSON(msg); err != nil {
		c.conn.Close()
		c.conn = nil
		return fmt.Errorf("write CDP navigate: %w", err)
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(30 * time.Second)
	}
	c.conn.SetReadDeadline(deadline)

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			c.conn.Close()
			c.conn = nil
			return fmt.Errorf("read CDP navigate response: %w", err)
		}

		var resp cdpResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			continue
		}
		if resp.ID != id {
			continue
		}
		if resp.Error != nil {
			return fmt.Errorf("CDP navigate error %d: %s", resp.Error.Code, resp.Error.Message)
		}
		return nil
	}
}

func (c *CDPClient) DiscoverPageTarget(ctx context.Context) (string, error) {
	return c.discoverPageTarget(ctx)
}

func (c *CDPClient) Close() error {
	c.acquireSemBlocking()
	defer c.releaseSem()
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

func (c *CDPClient) IsConnected() bool {
	select {
	case c.sem <- struct{}{}:
		connected := c.conn != nil
		c.releaseSem()
		return connected
	default:
		return true
	}
}
