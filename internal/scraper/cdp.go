package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
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
	ID     int64            `json:"id"`
	Result json.RawMessage  `json:"result,omitempty"`
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

type CDPClient struct {
	mu      sync.Mutex
	conn    *websocket.Conn
	baseURL string
	nextID  atomic.Int64
}

func NewCDPClient(baseURL string) *CDPClient {
	return &CDPClient{baseURL: strings.TrimRight(baseURL, "/")}
}

func (c *CDPClient) discoverPageTarget(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/json/list", nil)
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
	c.mu.Lock()
	defer c.mu.Unlock()

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
	c.mu.Lock()
	defer c.mu.Unlock()

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

func (c *CDPClient) Fetch(ctx context.Context, apiPath string) (string, error) {
	expression := fmt.Sprintf(`
		(async () => {
			const resp = await fetch('%s');
			return await resp.text();
		})()
	`, strings.ReplaceAll(apiPath, "'", "\\'"))

	return c.Evaluate(ctx, expression)
}

func (c *CDPClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

func (c *CDPClient) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil
}
