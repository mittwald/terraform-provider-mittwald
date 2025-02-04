package providerutil

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
)

// ClientFromProviderData is a helper function to extract the client from the
// provider data.
func ClientFromProviderData(providerData any, d *diag.Diagnostics) mittwaldv2.Client {
	if providerData == nil {
		return nil
	}

	client, ok := providerData.(mittwaldv2.Client)

	if !ok {
		d.AddError(
			"mittwald API client has unexpected type",
			fmt.Sprintf("Expected mittwaldv2.Client, got: %T. Please report this issue to the provider developers at https://github.com/mittwald/terraform-provider-mittwald/issues.", providerData),
		)

		return nil
	}

	return client
}
