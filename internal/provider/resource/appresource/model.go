package appresource

import "github.com/hashicorp/terraform-plugin-framework/types"

type ResourceModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	DatabaseID       types.String `tfsdk:"database_id"` // TODO: There may theoretically be multiple database links
	Description      types.String `tfsdk:"description"`
	App              types.String `tfsdk:"app"`
	Version          types.String `tfsdk:"version"`
	VersionCurrent   types.String `tfsdk:"version_current"`
	DocumentRoot     types.String `tfsdk:"document_root"`
	InstallationPath types.String `tfsdk:"installation_path"`
	UpdatePolicy     types.String `tfsdk:"update_policy"`
	UserInputs       types.Map    `tfsdk:"user_inputs"`
	Dependencies     types.Map    `tfsdk:"dependencies"`
}

type DependencyModel struct {
	Version      types.String `tfsdk:"version"`
	UpdatePolicy types.String `tfsdk:"update_policy"`
}
