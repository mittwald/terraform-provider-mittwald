package redisdatabaseresource

import "github.com/hashicorp/terraform-plugin-framework/types"

// ResourceModel describes the resource data model.
type ResourceModel struct {
	ID          types.String `tfsdk:"id"`
	ProjectID   types.String `tfsdk:"project_id"`
	Version     types.String `tfsdk:"version"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Hostname    types.String `tfsdk:"hostname"`

	Configuration types.Object `tfsdk:"configuration"`
}

type RedisConfigurationModel struct {
	AdditionalFlags types.List   `tfsdk:"additional_flags"`
	MaxMemoryMB     types.Int64  `tfsdk:"max_memory_mb"`
	MaxMemoryPolicy types.String `tfsdk:"max_memory_policy"`
	Persistent      types.Bool   `tfsdk:"persistent"`
}
