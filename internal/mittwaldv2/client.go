package mittwaldv2

import "net/http"

type Client struct {
	client  http.Client
	apiKey  string
	baseURL string
}

type ClientOpt func(*Client)

func WithAPIKey(apiKey string) ClientOpt {
	return func(c *Client) {
		c.apiKey = apiKey
	}
}

func New(opts ...ClientOpt) *Client {
	c := Client{}
	c.client = *http.DefaultClient
	c.baseURL = "https://api.mittwald.de/v2"

	for _, opt := range opts {
		opt(&c)
	}

	return &c
}
