package projectdatasource

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/contractv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/projectv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/resource/projectresource"
)

// DataSourceModel describes the data source data model.
//
// It mirrors the mittwald_project resource, except for the resource's write-only
// `use_free_trial` attribute, which has no meaning outside of an order and
// cannot be read back from the API.
type DataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	ShortID     types.String `tfsdk:"short_id"`
	ServerID    types.String `tfsdk:"server_id"`
	CustomerID  types.String `tfsdk:"customer_id"`
	ArticleID   types.String `tfsdk:"article_id"`
	ContractID  types.String `tfsdk:"contract_id"`
	Description types.String `tfsdk:"description"`
	DiskspaceGB types.Int64  `tfsdk:"diskspace_gb"`
	Directories types.Map    `tfsdk:"directories"`
	DefaultIPs  types.List   `tfsdk:"default_ips"`
}

// fromAPIModel maps an API project into the model, reusing the resource's
// mapping so that the two cannot drift apart.
//
// contract is the project's own contract, and is nil for a project on a server.
func (d *DataSourceModel) fromAPIModel(ctx context.Context, project *projectv2.Project, ips []string, contract *contractv2.Contract) (res diag.Diagnostics) {
	var mapped projectresource.ResourceModel

	res.Append(mapped.FromAPIModel(ctx, project, ips)...)
	if res.HasError() {
		return
	}

	d.ID = mapped.ID
	d.ShortID = mapped.ShortID
	d.ServerID = mapped.ServerID
	d.CustomerID = mapped.CustomerID
	d.Description = mapped.Description
	d.DiskspaceGB = mapped.DiskspaceGB
	d.Directories = mapped.Directories
	d.DefaultIPs = mapped.DefaultIPs

	d.ContractID = types.StringNull()
	d.ArticleID = types.StringNull()

	if contract != nil {
		d.ContractID = types.StringValue(contract.ContractId)
		if len(contract.BaseItem.Articles) > 0 {
			d.ArticleID = types.StringValue(contract.BaseItem.Articles[0].Id)
		}
	}

	return
}
