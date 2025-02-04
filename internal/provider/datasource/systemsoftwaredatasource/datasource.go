package systemsoftwaredatasource

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiext"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
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
	resp.TypeName = req.ProviderTypeName + "_systemsoftware"
}

func (d *DataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `A data source that selects versions of system components, such as PHP, MySQL, etc.

This data source should typically be used in conjunction with the ` + "`mittwald_app`" + `
resource to select the respective versions for the ` + "`dependencies`" + ` attribute.`,

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
				MarkdownDescription: "A version selector, such as `>= 7.4`; if omitted, this will default to `*` (all versions)",
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

func (d *DataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = providerutil.ClientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (d *DataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	appClient := apiext.NewAppClient(d.client)

	systemSoftware, ok, err := appClient.GetSystemsoftwareByName(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get system software", err.Error())
		return
	} else if !ok {
		resp.Diagnostics.AddError("System software not found", fmt.Sprintf("System software '%s' not found", data.Name.ValueString()))
		return
	}

	versions, err := appClient.SelectSystemsoftwareVersion(ctx, systemSoftware.Id, data.SelectorOrDefault())
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
		data.VersionID = types.StringValue(recommended.Id)
	} else {
		data.Version = types.StringValue(versions[len(versions)-1].InternalVersion)
		data.VersionID = types.StringValue(versions[len(versions)-1].Id)
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
