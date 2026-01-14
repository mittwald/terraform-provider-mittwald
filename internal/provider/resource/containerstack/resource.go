package containerstackresource

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/resource/common"
)

var _ resource.Resource = &Resource{}
var _ resource.ResourceWithImportState = &Resource{}

func New() resource.Resource {
	return &Resource{}
}

type Resource struct {
	client mittwaldv2.Client
}

func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_stack"
}
func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	builder := common.AttributeBuilderFor("container_stack")
	resp.Schema = schema.Schema{
		MarkdownDescription: "This resource models a container stack.\n\n" +
			"A container stack may consist of multiple containers and volumes.\n\n" +
			"**IMPORTANT**: Currently, the mStudio API supports one \"default\" stack per project. " +
			"In the future, support for multiple stacks within the same project will be added.\n\n" +
			"This resource's API already pre-empts this functionality; however, at the moment, you " +
			"can only manage containers in a project's default stack. To use the default stack, set " +
			"the `default_stack` attribute to `true`.",

		Attributes: map[string]schema.Attribute{
			"id":         builder.Id(),
			"project_id": builder.ProjectId(),
			"default_stack": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Set this flag to use the project's default stack. Otherwise, a new stack will be created.",
			},
			"containers": schema.MapNestedAttribute{
				Required: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The generated container ID",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseNonNullStateForUnknown(),
							},
						},
						"image": schema.StringAttribute{
							Required: true,
							MarkdownDescription: "The image to use for the container. Follows the usual Docker image format, " +
								"e.g. `nginx:latest` or `registry.example.com/my-image:latest`.\n\n  " +
								"Note that when using a non-standard registry (or a standard registry with credentials), " +
								"you will probably also need to add a `mittwald_container_registry` resource somewhere " +
								"in your plan.",
							PlanModifiers: []planmodifier.String{
								&StripLibraryPrefixFromImage{},
							},
						},
						"description": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "A description for the container.",
						},
						"command": schema.ListAttribute{
							Required: true,
							MarkdownDescription: "The command to run inside the container.\n\n" +
								"Note that this is a required value, even if the image already has a default command. " +
								"To use the default command, use the `mittwald_container_image` data source to first " +
								"determine the default command, and then use that value here.",
							ElementType: types.StringType,
						},
						"entrypoint": schema.ListAttribute{
							Required: true,
							MarkdownDescription: "The entrypoint to use for the container.\n\n" +
								"Note that this is a required value, even if the image already has a default entrypoint. " +
								"To use the default entrypoint, use the `mittwald_container_image` data source to first " +
								"determine the default entrypoint, and then use that value here.",
							ElementType: types.StringType,
						},
						"environment": schema.MapAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "A map of environment variables to set inside the container.",
							ElementType:         types.StringType,
							PlanModifiers: []planmodifier.Map{
								mapplanmodifier.UseNonNullStateForUnknown(),
							},
						},
						"ports": schema.SetNestedAttribute{
							Optional:            true,
							MarkdownDescription: "A port to expose from the container.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"container_port": schema.Int32Attribute{
										Required:            true,
										MarkdownDescription: "The container port to expose.",
									},
									"public_port": schema.Int32Attribute{
										Optional: true,
										Computed: true,
										MarkdownDescription: "The public port to expose; when omitted, this will " +
											"default to the same value as `container_port`.",
									},
									"protocol": schema.StringAttribute{
										Optional: true,
										Computed: true,
										MarkdownDescription: "The protocol to use for the port. Currently, the only" +
											" supported value is `tcp`, which is also the default.",
										Default: stringdefault.StaticString("tcp"),
										Validators: []validator.String{
											&PortProtocolValidator{},
										},
									},
								},
							},
						},
						"volumes": schema.SetNestedAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "Volumes to mount into the container.",
							PlanModifiers: []planmodifier.Set{
								setplanmodifier.UseNonNullStateForUnknown(),
							},
							NestedObject: schema.NestedAttributeObject{
								Validators: []validator.Object{
									&VolumeMountValidator{},
								},
								Attributes: map[string]schema.Attribute{
									"volume": schema.StringAttribute{
										Optional: true,
										MarkdownDescription: "The name of the volume to mount. A volume of this name " +
											"must be specified in the top-level `volumes` attribute.\n\n" +
											"    Either this attribute, or `project_path` must be set.",
									},
									"project_path": schema.StringAttribute{
										Optional: true,
										MarkdownDescription: "Path to a directory in the project filesystem.\n\n" +
											"    Either this attribute, or `volume` must be set.",
									},
									"mount_path": schema.StringAttribute{
										Required:            true,
										MarkdownDescription: "The path to mount the volume to.",
									},
								},
							},
						},
						"no_recreate_on_change": schema.BoolAttribute{
							Optional:            true,
							MarkdownDescription: "Set this flag to **not** recreate the container if any of the configuration changes. This includes changes to the image, command, entrypoint, environment variables, and ports. If this is set, you will need to manually recreate the container to apply any changes.",
							WriteOnly:           true,
						},
					},
				},
			},
			"volumes": schema.MapNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "A map of volumes that should be provisioned for this stack.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{},
				},
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.ClientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// TODO
}
