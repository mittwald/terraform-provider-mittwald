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
	Username        types.String `tfsdk:"username"`
	Password        types.String `tfsdk:"password_wo"`
	PasswordVersion types.Int64  `tfsdk:"password_wo_version"`
}

var containerRegistryCredentialsAttributeTypes = map[string]attr.Type{
	"username":            types.StringType,
	"password_wo":         types.StringType,
	"password_wo_version": types.Int64Type,
}
