package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/mittwald/terraform-provider-mittwald/api/mittwaldv2"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ProjectByShortIdDataSource{}

func NewProjectByShortIdDataSource() datasource.DataSource {
	return &ProjectByShortIdDataSource{}
}

// ProjectByShortIdDataSource defines the data source implementation.
type ProjectByShortIdDataSource struct {
	client mittwaldv2.ClientBuilder
}

// ProjectByShortIdDataSourceModel describes the data source data model.
type ProjectByShortIdDataSourceModel struct {
	Id      types.String `tfsdk:"id"`
	ShortId types.String `tfsdk:"short_id"`
}

func (d *ProjectByShortIdDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_by_shortid"
}

func (d *ProjectByShortIdDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "A data source that selects a project based on its short ID.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The project identifier",
				Computed:            true,
			},
			"short_id": schema.StringAttribute{
				MarkdownDescription: "The project short ID",
				Required:            true,
			},
		},
	}
}

func (d *ProjectByShortIdDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(mittwaldv2.ClientBuilder)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *ProjectByShortIdDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ProjectByShortIdDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projects, err := d.client.Project().ListProjects(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read example, got error: %s", err))
		return
	}

	if !data.ShortId.IsNull() {
		for _, project := range projects {
			if project.ShortId == data.ShortId.ValueString() {
				data.Id = types.StringValue(project.Id.String())
				break
			}
		}
	}

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "read a data source")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
