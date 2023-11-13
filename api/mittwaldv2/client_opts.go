package mittwaldv2

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
