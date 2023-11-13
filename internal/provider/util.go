package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/mittwald/terraform-provider-mittwald/api/mittwaldv2"
)

// clientFromProviderData is a helper function to extract the client from the
// provider data.
func clientFromProviderData(providerData any, d *diag.Diagnostics) mittwaldv2.ClientBuilder {
	if providerData == nil {
		d.AddError(
			"Unexpected Resource Configure Type",
			"Expected mittwaldv2.ClientBuilder, got: nil. Please report this issue to the provider developers at https://github.com/mittwald/terraform-provider-mittwald/issues.",
		)

		return nil
	}

	client, ok := providerData.(mittwaldv2.ClientBuilder)

	if !ok {
		d.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected mittwaldv2.ClientBuilder, got: %T. Please report this issue to the provider developers at https://github.com/mittwald/terraform-provider-mittwald/issues.", providerData),
		)

		return nil
	}

	return client
}
