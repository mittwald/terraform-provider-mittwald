package containerregistryresource

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/containerclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/containerv2"
)

func (m *ContainerRegistryModel) ToCreateRequest(ctx context.Context, d *diag.Diagnostics) *containerclientv2.CreateRegistryRequest {
	var credential ContainerRegistryCredentialsModel

	d.Append(m.Credentials.As(ctx, &credential, basetypes.ObjectAsOptions{})...)

	req := containerclientv2.CreateRegistryRequest{
		ProjectID: m.ProjectID.ValueString(),
		Body: containerv2.CreateRegistry{
			Description: m.Description.ValueString(),
			Uri:         m.URI.ValueString(),
			Credentials: nil,
		},
	}

	if !m.Credentials.IsNull() {
		req.Body.Credentials = &containerv2.SetRegistryCredentials{
			Username: credential.Username.ValueString(),
			Password: credential.Password.ValueString(),
		}
	}

	return &req
}

func (m *ContainerRegistryModel) ToDeleteRequest() *containerclientv2.DeleteRegistryRequest {
	return &containerclientv2.DeleteRegistryRequest{
		RegistryID: m.ID.ValueString(),
	}
}

func (m *ContainerRegistryModel) ToUpdateRequest(ctx context.Context, d *diag.Diagnostics) *containerclientv2.UpdateRegistryRequest {
	var credential ContainerRegistryCredentialsModel

	d.Append(m.Credentials.As(ctx, &credential, basetypes.ObjectAsOptions{})...)

	req := containerclientv2.UpdateRegistryRequest{
		RegistryID: m.ID.ValueString(),
		Body: containerv2.UpdateRegistry{
			Description: m.Description.ValueStringPointer(),
			Uri:         m.URI.ValueStringPointer(),
		},
	}

	if m.Credentials.IsNull() {
		req.Body.Credentials = &containerv2.UpdateRegistryCredentials{
			Value: nil,
		}
	} else {
		req.Body.Credentials = &containerv2.UpdateRegistryCredentials{
			Value: &containerv2.SetRegistryCredentials{
				Username: credential.Username.ValueString(),
				Password: credential.Password.ValueString(),
			},
		}
	}

	return &req
}
