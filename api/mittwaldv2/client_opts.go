package mittwaldv2

import (
	"bytes"
	"context"
	"fmt"
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

func WithDebugging(withRequestBodies bool) ClientBuilderOption {
	return func(_ *clientBuilder, c *Client) {
		c.RequestEditors = append(c.RequestEditors, func(ctx context.Context, req *http.Request) error {
			logParams := map[string]any{
				"method": req.Method,
				"url":    req.URL.String(),
			}

			if req.Body != nil && withRequestBodies {
				body, err := io.ReadAll(req.Body)
				if err != nil {
					return err
				}

				req.Body = io.NopCloser(bytes.NewBuffer(body))
				logParams["body"] = string(body)
			}

			tflog.Debug(ctx, fmt.Sprintf("executing %s request to %s", req.Method, req.URL.String()), logParams)
			return nil
		})
	}
}
