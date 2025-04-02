package containerstackresource

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ContainerStackModel struct {
	ID           types.String `tfsdk:"id"`
	ProjectID    types.String `tfsdk:"project_id"`
	DefaultStack types.Bool   `tfsdk:"default_stack"`
	Containers   types.Map    `tfsdk:"containers"`
	Volumes      types.Map    `tfsdk:"volumes"`
}

type ContainerModel struct {
	ID          types.String `tfsdk:"id"`
	Image       types.String `tfsdk:"image"`
	Description types.String `tfsdk:"description"`
	Command     types.List   `tfsdk:"command"`
	Entrypoint  types.List   `tfsdk:"entrypoint"`
	Environment types.Map    `tfsdk:"environment"`
	Ports       types.Set    `tfsdk:"ports"`
	Volumes     types.Set    `tfsdk:"volumes"`
}

type ContainerPortModel struct {
	ContainerPort types.Int32  `tfsdk:"container_port"`
	PublicPort    types.Int32  `tfsdk:"public_port"`
	Protocol      types.String `tfsdk:"protocol"`
}

type ContainerVolumeModel struct {
	Volume      types.String `tfsdk:"volume"`
	ProjectPath types.String `tfsdk:"project_path"`
	MountPath   types.String `tfsdk:"mount_path"`
}

type VolumeModel struct{}
