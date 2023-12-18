package providertesting

import (
	"github.com/mittwald/terraform-provider-mittwald/api/mittwaldv2"
	"os"
)

func TestClient() mittwaldv2.ClientBuilder {
	apiKey := os.Getenv("MITTWALD_API_TOKEN")

	opts := make([]mittwaldv2.ClientBuilderOption, 0)
	opts = append(opts, mittwaldv2.WithAPIToken(apiKey))

	return mittwaldv2.New(opts...)
}
