package containerimagedatasource

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ContainerImageDataSource{}

func New() datasource.DataSource {
	return &ContainerImageDataSource{}
}

// ContainerImageDataSource defines the data source implementation.
type ContainerImageDataSource struct {
	client mittwaldv2.Client
}

func (d *ContainerImageDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_image"
}

func (d *ContainerImageDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `A data source that queries metadata for a given container image.

This data source should typically be used in conjunction with the ` + "`mittwald_container_stack`" + `
resource to select the default values for the ` + "`command`" + ` and ` + "`entrypoint`" + ` attributes.`,

		Attributes: map[string]schema.Attribute{
			"image": schema.StringAttribute{
				MarkdownDescription: "The image to use for the container. Follows the usual Docker image format, " +
					"e.g. `nginx:latest` or `registry.example.com/my-image:latest`.",
				Required: true,
			},
			"registry_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the registry where the image is stored. This attribute (or `project_id`) " +
					"is required if the image is not public.",
				Optional: true,
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the project where the image is stored. This attribute (or `registry_id`) " +
					" is required if the image is not public.",
				Optional: true,
			},
			"command": schema.ListAttribute{
				MarkdownDescription: "The command to run in the container.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"entrypoint": schema.ListAttribute{
				MarkdownDescription: "The entrypoint to run in the container.",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (d *ContainerImageDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = providerutil.ClientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (d *ContainerImageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ContainerImageDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	containerClient := d.client.Container()
	containerReq := data.ToRequest()
	container, _, err := containerClient.GetContainerImageConfig(ctx, *containerReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get container image metadata", err.Error())
		return
	}

	resp.Diagnostics.Append(data.FromAPIModel(container)...)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
