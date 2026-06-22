package serverdatasource

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DataSourceModel describes the data source data model. It mirrors the
// attributes exposed by the mittwald_server resource.
type DataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	ContractID  types.String `tfsdk:"contract_id"`
	CustomerID  types.String `tfsdk:"customer_id"`
	ArticleID   types.String `tfsdk:"article_id"`
	Description types.String `tfsdk:"description"`
	DiskspaceGB types.Int64  `tfsdk:"diskspace_gb"`
	ShortID     types.String `tfsdk:"short_id"`
	MachineType types.String `tfsdk:"machine_type"`
	Status      types.String `tfsdk:"status"`
	ClusterName types.String `tfsdk:"cluster_name"`
	CreatedAt   types.String `tfsdk:"created_at"`
}
