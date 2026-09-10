package projectdatasource

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/contractclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/projectclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/contractv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/projectv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiext"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/resource/projectresource"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &DataSource{}

func New() datasource.DataSource {
	return &DataSource{}
}

// DataSource defines the data source implementation for the mittwald_project
// data source. It allows looking up an existing project either by its full ID
// or by its short ID, and exposes the same set of attributes as the
// mittwald_project resource.
type DataSource struct {
	client mittwaldv2.Client
}

// dataSourceModel describes the data source data model.
//
// It mirrors the mittwald_project resource, except for the resource's write-only
// `use_free_trial` attribute, which has no meaning outside of an order and
// cannot be read back from the API. Because of that difference, its fields are
// populated explicitly in fromAPIModel rather than by embedding
// projectresource.ResourceModel directly.
type dataSourceModel struct {
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
	Status      types.String `tfsdk:"status"`
	CreatedAt   types.String `tfsdk:"created_at"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

// fromAPIModel maps an API project into the model, reusing the resource's
// mapping so that the two cannot drift apart.
//
// contract is the project's own contract, and is nil for a project on a server.
func (d *dataSourceModel) fromAPIModel(ctx context.Context, project *projectv2.Project, ips []string, contract *contractv2.Contract) (res diag.Diagnostics) {
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
	d.Status = mapped.Status
	d.CreatedAt = mapped.CreatedAt

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

// readTimeoutHint is appended to diagnostics caused by an exhausted read
// timeout, to point users at the knob they can turn.
const readTimeoutHint = "If this happens regularly, increase the `timeouts.read` value on this data source."

func (d *DataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (d *DataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Selects an existing project on the mittwald cloud platform.\n\n" +
			"Exactly one of `id` or `short_id` must be set; the other is populated from the API, " +
			"alongside the remaining project attributes (such as `default_ips`). This is useful for " +
			"referencing projects that are not managed by this Terraform configuration, for example to " +
			"attach a `mittwald_virtualhost` to a project's default IP addresses.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The project identifier (full UUID). Either `id` or `short_id` must be set.",
				Optional:            true,
				Computed:            true,
			},
			"short_id": schema.StringAttribute{
				MarkdownDescription: "The project short ID (for example `p-XXXXXX`). Either `id` or `short_id` must be set.",
				Optional:            true,
				Computed:            true,
			},
			"server_id": schema.StringAttribute{
				MarkdownDescription: "ID of the server this project belongs to. Null for stand-alone projects.",
				Computed:            true,
			},
			"customer_id": schema.StringAttribute{
				MarkdownDescription: "ID of the customer this project belongs to.",
				Computed:            true,
			},
			"article_id": schema.StringAttribute{
				MarkdownDescription: "The article ID selecting the plan of a stand-alone project (for example a hosting plan " +
					"or a machine type, depending on the article). Null for projects on a server.",
				Computed: true,
			},
			"contract_id": schema.StringAttribute{
				MarkdownDescription: "The contract ID associated with a stand-alone project. Null for projects on a server, which are billed via the server's contract.",
				Computed:            true,
			},
			"diskspace_gb": schema.Int64Attribute{
				MarkdownDescription: "The amount of disk space the project is allotted, in GiB.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The project description.",
				Computed:            true,
			},
			"directories": schema.MapAttribute{
				MarkdownDescription: "Contains a map of data directories within the project.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"default_ips": schema.ListAttribute{
				MarkdownDescription: "Contains a list of default IP addresses for the project.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The current status of the project.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The time at which the project was created.",
				Computed:            true,
			},
		},

		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockWithOpts(ctx, timeouts.Opts{
				ReadDescription: "Time to wait when reading the project. This is an upper bound for the " +
					"(usually near-instant) API calls involved, including waiting for a not-yet-provisioned " +
					"default ingress (and with it, the `default_ips` attribute) to become available; " +
					"defaults to 2 minutes.",
			}),
		},
	}
}

func (d *DataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = providerutil.ClientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (d *DataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// Reuse the resource model and its API mapping so the data source and the
	// mittwald_project resource cannot drift when project attributes change.
	var data dataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, diags := data.Timeouts.Read(ctx, projectresource.DefaultReadTimeout)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	// The mittwald API resolves both full and short IDs through the same
	// endpoint, so either value can be passed straight through to GetProject.
	projectID, err := projectLookupID(data.ID, data.ShortID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid project selector", err.Error())
		return
	}

	client := apiext.NewProjectClient(d.client)

	project := providerutil.
		Try[*projectv2.Project](&resp.Diagnostics, "error while reading project").
		DoValResp(client.GetProject(ctx, projectclientv2.GetProjectRequest{ProjectID: projectID}))

	if resp.Diagnostics.HasError() {
		return
	}

	// A missing default ingress is not an error; the project's IP addresses may
	// simply not be available yet.
	ips := projectresource.PollDefaultIPs(ctx, client, project.Id, readTimeoutHint, &resp.Diagnostics)

	if resp.Diagnostics.HasError() {
		return
	}

	// Only a stand-alone project has a contract of its own; a project on a
	// server is billed via that server's contract.
	var contract *contractv2.Contract
	if project.ServerId == nil {
		contract = providerutil.
			Try[*contractv2.Contract](&resp.Diagnostics, "error while reading project contract").
			IgnoreNotFound().
			DoValResp(d.client.Contract().GetDetailOfContractByProject(ctx, contractclientv2.GetDetailOfContractByProjectRequest{ProjectID: project.Id}))

		if resp.Diagnostics.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(data.fromAPIModel(ctx, project, ips, contract)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
