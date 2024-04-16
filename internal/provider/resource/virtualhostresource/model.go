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
	App      types.String `tfsdk:"app"`
	Redirect types.String `tfsdk:"redirect"`
}

var pathType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"app":      types.StringType,
		"redirect": types.StringType,
	},
}
