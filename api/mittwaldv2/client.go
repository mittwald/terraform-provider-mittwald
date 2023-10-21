package mittwaldv2

import (
	"context"
	"net/http"
)

func apiTokenRequestEditor(token string) RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	}
}

func NewWithAPIToken(token string) ClientBuilder {
	httpClient := http.DefaultClient
	client := ClientWithResponses{
		ClientInterface: &Client{
			Server: "https://api.mittwald.de/v2",
			Client: httpClient,
			RequestEditors: []RequestEditorFn{
				apiTokenRequestEditor(token),
			},
		},
	}

	return &clientBuilder{
		internalClient: &client,
	}
}
