package remotefileresource

import "github.com/hashicorp/terraform-plugin-framework/types"

// ResourceModel describes the resource data model.
type ResourceModel struct {
	ID          types.String `tfsdk:"id"`
	ContainerID types.String `tfsdk:"container_id"`
	StackID     types.String `tfsdk:"stack_id"`
	AppID       types.String `tfsdk:"app_id"`
	SSHUser     types.String `tfsdk:"ssh_user"`
	Path        types.String `tfsdk:"path"`
	Contents    types.String `tfsdk:"contents"`
}

// Reset resets the model to its zero values
func (m *ResourceModel) Reset() {
	m.ID = types.StringNull()
	m.ContainerID = types.StringNull()
	m.StackID = types.StringNull()
	m.AppID = types.StringNull()
	m.SSHUser = types.StringNull()
	m.Path = types.StringNull()
	m.Contents = types.StringNull()
}
