package rds

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type Client struct {
	mu         sync.RWMutex
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) url(path string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.baseURL + path
}

func (c *Client) get(path string, result any) error {
	resp, err := c.httpClient.Get(c.url(path))
	if err != nil {
		return fmt.Errorf("rds GET %s: %w", path, err)
	}
	defer resp.Body.Close()
	return c.decode(resp, result)
}

func (c *Client) post(path string, body any, result any) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("rds marshal: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}
	resp, err := c.httpClient.Post(c.url(path), "application/json", bodyReader)
	if err != nil {
		return fmt.Errorf("rds POST %s: %w", path, err)
	}
	defer resp.Body.Close()
	return c.decode(resp, result)
}

func (c *Client) decode(resp *http.Response, result any) error {
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("rds read body: %w", err)
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("rds HTTP %d: %s", resp.StatusCode, string(data))
	}
	if result != nil {
		if err := json.Unmarshal(data, result); err != nil {
			return fmt.Errorf("rds decode: %w", err)
		}
	}
	return nil
}

func (c *Client) getRaw(path string) ([]byte, error) {
	resp, err := c.httpClient.Get(c.url(path))
	if err != nil {
		return nil, fmt.Errorf("rds GET %s: %w", path, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("rds read body: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("rds HTTP %d: %s", resp.StatusCode, string(data))
	}
	return data, nil
}

func (c *Client) postRaw(path string, contentType string, body io.Reader, result any) error {
	resp, err := c.httpClient.Post(c.url(path), contentType, body)
	if err != nil {
		return fmt.Errorf("rds POST %s: %w", path, err)
	}
	defer resp.Body.Close()
	return c.decode(resp, result)
}

// BaseURL returns the client's base URL.
func (c *Client) BaseURL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.baseURL
}

// Reconfigure updates the client's base URL and timeout for hot-reload.
func (c *Client) Reconfigure(baseURL string, timeout time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.baseURL = baseURL
	c.httpClient.Timeout = timeout
}

// checkResponse validates the RDS response envelope code.
func checkResponse(r *Response) error {
	if r.Code != 0 {
		return fmt.Errorf("rds error %d: %s", r.Code, r.Msg)
	}
	return nil
}
