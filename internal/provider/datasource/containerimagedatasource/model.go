package containerimagedatasource

import "github.com/hashicorp/terraform-plugin-framework/types"

type ContainerImageDataSourceModel struct {
	Image        types.String `tfsdk:"image"`
	RegistryID   types.String `tfsdk:"registry_id"`
	ProjectID    types.String `tfsdk:"project_id"`
	Command      types.List   `tfsdk:"command"`
	Entrypoint   types.List   `tfsdk:"entrypoint"`
	Digest       types.String `tfsdk:"digest"`
	User         types.String `tfsdk:"user"`
	UserID       types.Int64  `tfsdk:"user_id"`
	IsUserRoot   types.Bool   `tfsdk:"is_user_root"`
	Env          types.List   `tfsdk:"env"`
	ExposedPorts types.List   `tfsdk:"exposed_ports"`
	Volumes      types.List   `tfsdk:"volumes"`
}
