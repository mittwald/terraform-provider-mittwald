package containerregistryresource

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/containerv2"
)

func (m *ContainerRegistryModel) FromAPIModel(ctx context.Context, registry *containerv2.Registry) diag.Diagnostics {
	return m.FromAPIModelWithCredentials(ctx, registry, types.StringNull(), types.Int64Null())
}

func (m *ContainerRegistryModel) FromAPIModelWithCredentials(_ context.Context, registry *containerv2.Registry, password types.String, passwordVersion types.Int64) diag.Diagnostics {
	diags := make(diag.Diagnostics, 0)

	m.ID = types.StringValue(registry.Id)
	m.URI = types.StringValue(registry.Uri)
	m.Description = types.StringValue(registry.Description)

	if registry.Credentials != nil {
		if m.Credentials.IsNull() {
			creds, d := types.ObjectValue(containerRegistryCredentialsAttributeTypes, map[string]attr.Value{
				"username":            types.StringValue(registry.Credentials.Username),
				"password_wo":         password,
				"password_wo_version": passwordVersion,
			})

			diags.Append(d...)
			m.Credentials = creds
		} else {
			attrs := m.Credentials.Attributes()
			attrs["username"] = types.StringValue(registry.Credentials.Username)

			if !password.IsNull() {
				attrs["password_wo"] = password
			}
			if !passwordVersion.IsNull() {
				attrs["password_wo_version"] = passwordVersion
			}

			creds, d := types.ObjectValue(containerRegistryCredentialsAttributeTypes, attrs)

			diags.Append(d...)
			m.Credentials = creds
		}
	} else {
		m.Credentials = types.ObjectNull(containerRegistryCredentialsAttributeTypes)
	}

	return diags
}
