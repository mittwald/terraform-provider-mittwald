package mittwaldv2

import (
	"bytes"
	"context"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"io"
	"net/http"
)

type ClientBuilderOption func(*clientBuilder, *Client)

func WithAPIToken(token string) ClientBuilderOption {
	return func(_ *clientBuilder, c *Client) {
		c.RequestEditors = append(c.RequestEditors, apiTokenRequestEditor(token))
	}
}

func WithEndpoint(endpoint string) ClientBuilderOption {
	return func(_ *clientBuilder, c *Client) {
		c.Server = endpoint
	}
}

type debuggingClient struct {
	HttpRequestDoer
	withRequestBodies bool
}

func (c *debuggingClient) Do(req *http.Request) (*http.Response, error) {
	logFields := map[string]any{
		"method": req.Method,
		"url":    req.URL.String(),
	}

	if req.Body != nil && c.withRequestBodies {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}

		req.Body = io.NopCloser(bytes.NewBuffer(body))
		logFields["body"] = string(body)
	}

	res, err := c.HttpRequestDoer.Do(req)

	if res != nil {
		logFields["status"] = res.StatusCode
	}

	if err != nil {
		logFields["err"] = err
	}

	tflog.Debug(context.Background(), "executed request", logFields)

	return res, err
}

func WithDebugging(withRequestBodies bool) ClientBuilderOption {
	return func(_ *clientBuilder, c *Client) {
		originalClient := c.Client
		c.Client = &debuggingClient{HttpRequestDoer: originalClient, withRequestBodies: withRequestBodies}
	}
}
