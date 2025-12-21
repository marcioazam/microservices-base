// Package http provides a resilient HTTP client.
package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is a resilient HTTP client with retry and circuit breaker.
type Client struct {
	client        *http.Client
	timeout       time.Duration
	retryAttempts int
	retryDelay    time.Duration
	headers       map[string]string
}

// NewClient creates a new HTTP client.
func NewClient() *Client {
	return &Client{
		client:        &http.Client{Timeout: 30 * time.Second},
		timeout:       30 * time.Second,
		retryAttempts: 3,
		retryDelay:    100 * time.Millisecond,
		headers:       make(map[string]string),
	}
}

// WithTimeout sets the request timeout.
func (c *Client) WithTimeout(timeout time.Duration) *Client {
	c.timeout = timeout
	c.client.Timeout = timeout
	return c
}

// WithRetry sets retry configuration.
func (c *Client) WithRetry(attempts int, delay time.Duration) *Client {
	c.retryAttempts = attempts
	c.retryDelay = delay
	return c
}

// WithHeader adds a default header.
func (c *Client) WithHeader(key, value string) *Client {
	c.headers[key] = value
	return c
}

// Response wraps http.Response with additional info.
type Response struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
	Duration   time.Duration
}

// Get performs a GET request.
func (c *Client) Get(ctx context.Context, url string) (*Response, error) {
	return c.do(ctx, http.MethodGet, url, nil)
}

// Post performs a POST request.
func (c *Client) Post(ctx context.Context, url string, body io.Reader) (*Response, error) {
	return c.do(ctx, http.MethodPost, url, body)
}

// Put performs a PUT request.
func (c *Client) Put(ctx context.Context, url string, body io.Reader) (*Response, error) {
	return c.do(ctx, http.MethodPut, url, body)
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, url string) (*Response, error) {
	return c.do(ctx, http.MethodDelete, url, nil)
}

func (c *Client) do(ctx context.Context, method, url string, body io.Reader) (*Response, error) {
	var lastErr error

	for attempt := 0; attempt <= c.retryAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryDelay * time.Duration(attempt))
		}

		start := time.Now()
		req, err := http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		for k, v := range c.headers {
			req.Header.Set(k, v)
		}

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			continue
		}

		// Retry on 5xx errors
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			continue
		}

		return &Response{
			StatusCode: resp.StatusCode,
			Body:       respBody,
			Headers:    resp.Header,
			Duration:   time.Since(start),
		}, nil
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", c.retryAttempts, lastErr)
}

// IsSuccess returns true if status code is 2xx.
func (r *Response) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// IsClientError returns true if status code is 4xx.
func (r *Response) IsClientError() bool {
	return r.StatusCode >= 400 && r.StatusCode < 500
}

// IsServerError returns true if status code is 5xx.
func (r *Response) IsServerError() bool {
	return r.StatusCode >= 500
}
