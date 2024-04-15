package appresource

import "github.com/hashicorp/terraform-plugin-framework/types"

type ResourceModel struct {
	ID                       types.String `tfsdk:"id"`
	ShortID                  types.String `tfsdk:"short_id"`
	ProjectID                types.String `tfsdk:"project_id"`
	Databases                types.Set    `tfsdk:"databases"`
	Description              types.String `tfsdk:"description"`
	App                      types.String `tfsdk:"app"`
	Version                  types.String `tfsdk:"version"`
	VersionCurrent           types.String `tfsdk:"version_current"`
	DocumentRoot             types.String `tfsdk:"document_root"`
	InstallationPath         types.String `tfsdk:"installation_path"`
	InstallationPathAbsolute types.String `tfsdk:"installation_path_absolute"`
	UpdatePolicy             types.String `tfsdk:"update_policy"`
	UserInputs               types.Map    `tfsdk:"user_inputs"`
	Dependencies             types.Map    `tfsdk:"dependencies"`
}
