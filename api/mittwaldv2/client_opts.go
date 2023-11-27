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

func WithDebugging() ClientBuilderOption {
	return func(_ *clientBuilder, c *Client) {
		c.RequestEditors = append(c.RequestEditors, func(ctx context.Context, req *http.Request) error {
			logParams := map[string]any{}

			if req.Body != nil {
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
