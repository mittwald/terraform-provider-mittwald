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

func New(opts ...ClientBuilderOption) ClientBuilder {
	httpClient := http.DefaultClient
	internalClient := Client{
		Server: "https://api.mittwald.de/v2",
		Client: httpClient,
	}

	builder := &clientBuilder{
		internalClient: &ClientWithResponses{
			ClientInterface: &internalClient,
		},
	}

	for _, opt := range opts {
		opt(builder, &internalClient)
	}

	return builder
}
