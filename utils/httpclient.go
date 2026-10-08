package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	maxBodySize    = 1 << 20 // 1 MB
	defaultTimeout = 30 * time.Second
)

type HttpClient struct {
	httpClient *http.Client
}

func NewHttpClient(timeout time.Duration) *HttpClient {
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &HttpClient{
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:          100,
				MaxIdleConnsPerHost:   10,
				IdleConnTimeout:       90 * time.Second,
				ResponseHeaderTimeout: 30 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
			},
		},
	}
}

// DoJSON executes a request with JSON payload serialization and response unmarshalling
func (c *HttpClient) DoJSON(ctx context.Context, method, url string, payload any, responseDst any) error {
	var bodyReader io.Reader
	if payload != nil {
		buf, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("httpclient: failed to marshal payload: %w", err)
		}
		bodyReader = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("httpclient: failed to create request: %w", err)
	}

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("httpclient: request execution failed: %w", err)
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
		return fmt.Errorf("httpclient: unexpected status code %d: %s", resp.StatusCode, string(respBody))
	}

	if responseDst != nil {
		if err := json.NewDecoder(resp.Body).Decode(responseDst); err != nil {
			return fmt.Errorf("httpclient: failed to decode response: %w", err)
		}
	}

	return nil
}
