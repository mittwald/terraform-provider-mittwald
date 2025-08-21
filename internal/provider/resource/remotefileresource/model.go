package remotefileresource

import "github.com/hashicorp/terraform-plugin-framework/types"

// ResourceModel describes the resource data model.
type ResourceModel struct {
	ID            types.String `tfsdk:"id"`
	ContainerID   types.String `tfsdk:"container_id"`
	StackID       types.String `tfsdk:"stack_id"`
	AppID         types.String `tfsdk:"app_id"`
	SSHUser       types.String `tfsdk:"ssh_user"`
	SSHPrivateKey types.String `tfsdk:"ssh_private_key"`
	Path          types.String `tfsdk:"path"`
	Contents      types.String `tfsdk:"contents"`
}
