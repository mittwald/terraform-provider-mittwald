package providerutil

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/mittwald/terraform-provider-mittwald/api/mittwaldv2"
)

// ClientFromProviderData is a helper function to extract the client from the
// provider data.
func ClientFromProviderData(providerData any, d *diag.Diagnostics) mittwaldv2.ClientBuilder {
	if providerData == nil {
		return nil
	}

	client, ok := providerData.(mittwaldv2.ClientBuilder)

	if !ok {
		d.AddError(
			"mittwald API client has unexpected type",
			fmt.Sprintf("Expected mittwaldv2.ClientBuilder, got: %T. Please report this issue to the provider developers at https://github.com/mittwald/terraform-provider-mittwald/issues.", providerData),
		)

		return nil
	}

	return client
}
