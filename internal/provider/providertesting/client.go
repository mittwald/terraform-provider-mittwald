package providertesting

import (
	"context"
	"github.com/mittwald/api-client-go/mittwaldv2"
	generatedv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"os"
)

func TestClient() generatedv2.Client {
	apiKey := os.Getenv("MITTWALD_API_TOKEN")

	opts := make([]mittwaldv2.ClientOption, 0)
	opts = append(opts, mittwaldv2.WithAccessToken(apiKey))

	client, _ := mittwaldv2.New(context.Background(), opts...)
	return client
}
