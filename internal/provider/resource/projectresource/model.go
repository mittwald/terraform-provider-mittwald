package projectresource

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ResourceModel describes the resource data model. It is shared with the
// mittwald_project data source, so that both cannot drift apart; the timeouts
// configuration is kept out of it, because resources and data sources use
// distinct types for it.
type ResourceModel struct {
	ID          types.String `tfsdk:"id"`
	ShortID     types.String `tfsdk:"short_id"`
	ServerID    types.String `tfsdk:"server_id"`
	Description types.String `tfsdk:"description"`
	Directories types.Map    `tfsdk:"directories"`
	DefaultIPs  types.List   `tfsdk:"default_ips"`
}

// ResourceModelWithTimeouts is the model actually used by the resource; it
// extends ResourceModel with the resource's `timeouts` block.
type ResourceModelWithTimeouts struct {
	ResourceModel

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (m *ResourceModel) Validate() (d diag.Diagnostics) {
	if m.ServerID.IsNull() {
		d.AddError("Missing value", "server_id is a required field")
	}

	return
}
