package localrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	http    *http.Client
	token   string
}

func NewClient(addr string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 3 * time.Second}
	}
	return &Client{
		baseURL: BaseURL(addr),
		http:    httpClient,
	}
}

func NewClientFromState(path string, httpClient *http.Client) (*Client, *State, error) {
	state, err := LoadState(path)
	if err != nil {
		return nil, nil, err
	}
	return NewClient(state.Addr, httpClient).WithToken(state.Token), state, nil
}

func (c *Client) WithToken(token string) *Client {
	c.token = strings.TrimSpace(token)
	return c
}

func (c *Client) Ping(ctx context.Context) (*HealthResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+HealthPath, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, decodeHTTPError(resp)
	}

	var out HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Exec(ctx context.Context, payload *ExecRequest) (*ExecResponse, error) {
	if payload == nil {
		payload = &ExecRequest{}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+ExecPath, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set(HeaderAuthToken, c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, decodeHTTPError(resp)
	}

	var out ExecResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func decodeHTTPError(resp *http.Response) error {
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	text := strings.TrimSpace(string(data))
	if text == "" {
		text = resp.Status
	}
	return fmt.Errorf("localrpc http %d: %s", resp.StatusCode, text)
}
