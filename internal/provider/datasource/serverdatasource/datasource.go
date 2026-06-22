package serverdatasource

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/contractclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/projectclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/contractv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/projectv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/resource/serverresource"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &DataSource{}

func New() datasource.DataSource {
	return &DataSource{}
}

// DataSource defines the data source implementation.
type DataSource struct {
	client mittwaldv2.Client
}

func (d *DataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server"
}

func (d *DataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A data source that selects a server by its ID or short ID. Exactly one of `id` or `short_id` must be set.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the server. Exactly one of `id` or `short_id` must be set.",
				Optional:            true,
				Computed:            true,
			},
			"contract_id": schema.StringAttribute{
				MarkdownDescription: "The contract ID associated with the server",
				Computed:            true,
			},
			"customer_id": schema.StringAttribute{
				MarkdownDescription: "ID of the customer the server belongs to",
				Computed:            true,
			},
			"article_id": schema.StringAttribute{
				MarkdownDescription: "The article ID determining the machine type of the server",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A human-readable description for the server",
				Computed:            true,
			},
			"diskspace_gb": schema.Int64Attribute{
				MarkdownDescription: "The amount of disk space for the server, in GiB",
				Computed:            true,
			},
			"short_id": schema.StringAttribute{
				MarkdownDescription: "The short ID of the server (for example `s-4e7tz3`). Exactly one of `id` or `short_id` must be set.",
				Optional:            true,
				Computed:            true,
			},
			"machine_type": schema.StringAttribute{
				MarkdownDescription: "The machine type of the server",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The current status of the server",
				Computed:            true,
			},
			"cluster_name": schema.StringAttribute{
				MarkdownDescription: "The name of the cluster the server is running on",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The time at which the server was created",
				Computed:            true,
			},
		},
	}
}

func (d *DataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = providerutil.ClientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (d *DataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasID := !data.ID.IsNull()
	hasShortID := !data.ShortID.IsNull()
	if hasID == hasShortID {
		resp.Diagnostics.AddError(
			"Invalid server selector",
			"Exactly one of \"id\" or \"short_id\" must be set.",
		)
		return
	}

	// The API resolves both the full ID and the short ID via the same path
	// parameter, so either selector can be passed directly.
	selector := data.ID.ValueString()
	if hasShortID {
		selector = data.ShortID.ValueString()
	}

	server := providerutil.
		Try[*projectv2.Server](&resp.Diagnostics, "error while reading server").
		DoValResp(d.client.Project().GetServer(ctx, projectclientv2.GetServerRequest{ServerID: selector}))

	if resp.Diagnostics.HasError() {
		return
	}

	serverID := server.Id

	contract := providerutil.
		Try[*contractv2.Contract](&resp.Diagnostics, "error while reading server contract").
		DoValResp(d.client.Contract().GetDetailOfContractByServer(ctx, contractclientv2.GetDetailOfContractByServerRequest{ServerID: serverID}))

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(data.fromAPIModel(server, contract)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *DataSourceModel) fromAPIModel(server *projectv2.Server, contract *contractv2.Contract) (diags diag.Diagnostics) {
	d.ID = types.StringValue(server.Id)
	d.CustomerID = types.StringValue(server.CustomerId)
	d.Description = types.StringValue(server.Description)
	d.ShortID = types.StringValue(server.ShortId)
	d.MachineType = types.StringValue(server.MachineType.Name)
	d.Status = types.StringValue(string(server.Status))
	d.ClusterName = types.StringValue(server.ClusterName)
	d.CreatedAt = types.StringValue(server.CreatedAt.Format(time.RFC3339))

	gib, err := serverresource.ParseStorageGiB(server.Storage)
	if err != nil {
		diags.AddError("error while parsing server storage", err.Error())
		return
	}
	d.DiskspaceGB = types.Int64Value(gib)

	d.ContractID = types.StringValue(contract.ContractId)
	if len(contract.BaseItem.Articles) > 0 {
		d.ArticleID = types.StringValue(contract.BaseItem.Articles[0].Id)
	} else {
		d.ArticleID = types.StringNull()
	}

	return
}
