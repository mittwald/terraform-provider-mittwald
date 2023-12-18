package providertesting

import (
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider"
	"os"
	"testing"
)

var TestAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"mittwald": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func TestAccPreCheck(t *testing.T) {
	if _, hasAPIToken := os.LookupEnv("MITTWALD_API_TOKEN"); !hasAPIToken {
		t.Fatal("MITTWALD_API_TOKEN not set")
	}

	if _, hasServerID := os.LookupEnv("MITTWALD_ACCTEST_SERVER_ID"); !hasServerID {
		t.Fatal("MITTWALD_ACCTEST_SERVER_ID not set")
	}
}
