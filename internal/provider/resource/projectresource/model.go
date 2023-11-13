package projectresource

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ResourceModel describes the resource data model.
type ResourceModel struct {
	ID          types.String `tfsdk:"id"`
	ServerID    types.String `tfsdk:"server_id"`
	Description types.String `tfsdk:"description"`
	Directories types.Map    `tfsdk:"directories"`
	DefaultIPs  types.List   `tfsdk:"default_ips"`
}

func (m *ResourceModel) Validate() (d diag.Diagnostics) {
	if m.ServerID.IsNull() {
		d.AddError("Missing value", "server_id is a required field")
	}

	return
}
