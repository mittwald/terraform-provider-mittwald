package mailoutboxresource

import "github.com/hashicorp/terraform-plugin-framework/types"

// ResourceModel describes the resource data model.
type ResourceModel struct {
	ID          types.String `tfsdk:"id"`
	ProjectID   types.String `tfsdk:"project_id"`
	Description types.String `tfsdk:"description"`
	Password    types.String `tfsdk:"password"`
}
