package projectdatasource

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/projectclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/projectv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiext"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
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

func (d *DataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (d *DataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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
				MarkdownDescription: "ID of the server this project belongs to. Empty for stand-alone projects.",
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

	ips := providerutil.
		Try[[]string](&resp.Diagnostics, "error while reading project ips").
		IgnoreNotFound().
		DoVal(client.GetProjectDefaultIPs(ctx, project.Id))

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(data.FromAPIModel(ctx, project, ips)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
