package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/mittwald/terraform-provider-mittwald/api/mittwaldv2"
)

// clientFromProviderData is a helper function to extract the client from the
// provider data.
func clientFromProviderData(req *resource.ConfigureRequest, resp *resource.ConfigureResponse) mittwaldv2.ClientBuilder {
	if req.ProviderData == nil {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			"Expected mittwaldv2.ClientBuilder, got: nil. Please report this issue to the provider developers at https://github.com/mittwald/terraform-provider-mittwald/issues.",
		)

		return nil
	}

	client, ok := req.ProviderData.(mittwaldv2.ClientBuilder)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected mittwaldv2.ClientBuilder, got: %T. Please report this issue to the provider developers at https://github.com/mittwald/terraform-provider-mittwald/issues.", req.ProviderData),
		)

		return nil
	}

	return client
}
