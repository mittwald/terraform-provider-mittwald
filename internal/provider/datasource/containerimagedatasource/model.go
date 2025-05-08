package containerimagedatasource

import "github.com/hashicorp/terraform-plugin-framework/types"

type ContainerImageDataSourceModel struct {
	Image      types.String `tfsdk:"image"`
	RegistryID types.String `tfsdk:"registry_id"`
	ProjectID  types.String `tfsdk:"project_id"`
	Command    types.List   `tfsdk:"command"`
	Entrypoint types.List   `tfsdk:"entrypoint"`
}
