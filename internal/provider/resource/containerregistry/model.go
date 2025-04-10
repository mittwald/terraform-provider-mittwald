package containerregistryresource

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ContainerRegistryModel struct {
	ID              types.String `tfsdk:"id"`
	ProjectID       types.String `tfsdk:"project_id"`
	DefaultRegistry types.Bool   `tfsdk:"default_registry"`
	Description     types.String `tfsdk:"description"`
	URI             types.String `tfsdk:"uri"`
	Credentials     types.Object `tfsdk:"credentials"`
}

type ContainerRegistryCredentialsModel struct {
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

var containerRegistryCredentialsAttributeTypes = map[string]attr.Type{
	"username": types.StringType,
	"password": types.StringType,
}
