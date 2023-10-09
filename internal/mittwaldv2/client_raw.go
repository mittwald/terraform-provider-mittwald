package mittwaldv2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"io"
	"net/http"
	"time"
)

func (c *Client) Do(ctx context.Context, req *http.Request, out any) error {
	req.Header.Set("X-Access-Token", c.apiKey)
	req.Header.Set("Accept", "application/json")

	tflog.Debug(ctx, fmt.Sprintf("request: %s %s", req.Method, req.URL.String()))

	resp, err := c.client.Do(req.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}

	if resp.StatusCode >= 400 {
		// TODO: Handle VErrors correctly and return a specific error type
		// {"params":{"traceId":"..."},"message":"...","type":"VError"}
		response, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d (%s)", resp.StatusCode, string(response))
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

func (c *Client) Poll(ctx context.Context, path string, out any) error {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	attempts := 20

	for i := 0; i < attempts; i++ {
		if err = c.Do(ctx, req, out); err == nil {
			return nil
		}
		time.Sleep(time.Duration(i) * 100 * time.Millisecond)
	}

	return fmt.Errorf("failed to execute request after %d attempts: %w", attempts, err)
}

func (c *Client) Get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if err := c.Do(ctx, req, out); err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}

	return nil
}

func (c *Client) Delete(ctx context.Context, path string) error {
	req, err := http.NewRequest(http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if err := c.Do(ctx, req, nil); err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}

	return nil
}

func (c *Client) Post(ctx context.Context, path string, input, out any) error {
	body, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("failed to marshal input: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if err := c.Do(ctx, req, out); err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}

	return nil
}

func (c *Client) Patch(ctx context.Context, path string, input, out any) error {
	body, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("failed to marshal input: %w", err)
	}

	req, err := http.NewRequest(http.MethodPatch, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if err := c.Do(ctx, req, out); err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}

	return nil
}
