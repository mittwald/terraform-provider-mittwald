package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/terraform-provider-mittwald/api/mittwaldv2"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &SystemSoftwareDataSource{}

func NewSystemSoftwareDataSource() datasource.DataSource {
	return &SystemSoftwareDataSource{}
}

// ProjectByShortIdDataSource defines the data source implementation.
type SystemSoftwareDataSource struct {
	client mittwaldv2.ClientBuilder
}

// SystemSoftwareDataSourceModel describes the data source data model.
type SystemSoftwareDataSourceModel struct {
	Name        types.String `tfsdk:"name"`
	Recommended types.Bool   `tfsdk:"recommended"`
	Selector    types.String `tfsdk:"selector"`

	Version   types.String `tfsdk:"version"`
	VersionID types.String `tfsdk:"version_id"`
}

func (m *SystemSoftwareDataSourceModel) SelectorOrDefault() string {
	if m.Selector.IsNull() {
		return "*"
	}
	return m.Selector.ValueString()
}

func (d *SystemSoftwareDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_systemsoftware"
}

func (d *SystemSoftwareDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "A data source that selects versions of system components, such as PHP, MySQL, etc.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "The system software name",
				Required:            true,
			},
			"recommended": schema.BoolAttribute{
				MarkdownDescription: "Set this to just select the recommended version",
				Optional:            true,
			},
			"selector": schema.StringAttribute{
				MarkdownDescription: "A version selector, such as `>= 7.4`",
				Optional:            true,
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "The selected version",
				Computed:            true,
			},
			"version_id": schema.StringAttribute{
				MarkdownDescription: "The selected version ID",
				Computed:            true,
			},
		},
	}
}

func (d *SystemSoftwareDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SystemSoftwareDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SystemSoftwareDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	systemSoftware, ok, err := d.client.App().GetSystemSoftwareByName(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get system software", err.Error())
		return
	} else if !ok {
		resp.Diagnostics.AddError("System software not found", fmt.Sprintf("System software '%s' not found", data.Name.ValueString()))
		return
	}

	versions, err := d.client.App().SelectSystemSoftwareVersion(ctx, systemSoftware.Id, data.SelectorOrDefault())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get recommended system software version", err.Error())
		return
	}

	if data.Recommended.ValueBool() {
		recommended, ok := versions.Recommended()
		if !ok {
			resp.Diagnostics.AddError("No recommended system software version found", fmt.Sprintf("No recommended version found for '%s'", data.Name.ValueString()))
			return
		}

		data.Version = types.StringValue(recommended.InternalVersion)
		data.VersionID = types.StringValue(recommended.Id.String())
	} else {
		data.Version = types.StringValue(versions[len(versions)-1].InternalVersion)
		data.VersionID = types.StringValue(versions[len(versions)-1].Id.String())
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
