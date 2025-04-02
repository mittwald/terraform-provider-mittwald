package virtualhostresource

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ResourceModel struct {
	ID        types.String `tfsdk:"id"`
	ProjectID types.String `tfsdk:"project_id"`
	Hostname  types.String `tfsdk:"hostname"`
	Paths     types.Map    `tfsdk:"paths"`
}

type PathModel struct {
	App       types.String `tfsdk:"app"`
	Redirect  types.String `tfsdk:"redirect"`
	Container types.Object `tfsdk:"container"`
}

type ContainerPathModel struct {
	ContainerID types.String `tfsdk:"container_id"`
	Port        types.String `tfsdk:"port"`
}

var pathType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"app":       types.StringType,
		"redirect":  types.StringType,
		"container": types.StringType,
	},
}

var containerPathType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"container_id": types.StringType,
		"port":         types.StringType,
	},
}
