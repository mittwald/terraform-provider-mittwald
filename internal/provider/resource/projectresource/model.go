package projectresource

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
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
	Status      types.String `tfsdk:"status"`
	CreatedAt   types.String `tfsdk:"created_at"`

	// The following attributes only apply to stand-alone projects, which are
	// ordered for a customer instead of being placed on an existing server.
	CustomerID   types.String `tfsdk:"customer_id"`
	ArticleID    types.String `tfsdk:"article_id"`
	ContractID   types.String `tfsdk:"contract_id"`
	DiskspaceGB  types.Int64  `tfsdk:"diskspace_gb"`
	UseFreeTrial types.Bool   `tfsdk:"use_free_trial"`
}

// ResourceModelWithTimeouts is the model actually used by the resource; it
// extends ResourceModel with the resource's `timeouts` block.
type ResourceModelWithTimeouts struct {
	ResourceModel

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

// IsStandalone reports whether the configuration describes a stand-alone
// project, as opposed to a project placed on an existing server.
//
// A stand-alone project is identified by the absence of `server_id`; an unknown
// `server_id` still counts as configured, because its value is merely not known
// yet.
func (m *ResourceModel) IsStandalone() bool {
	return m.ServerID.IsNull()
}
