package containerstackresource

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/containerclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/containerv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiext"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/resource/common"
	"reflect"
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
		MarkdownDescription: "This resource models a container stack.\n\nA container stack may consist of multiple containers and volumes.",

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
								stringplanmodifier.UseStateForUnknown(),
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
							Required:            true,
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
								mapplanmodifier.UseStateForUnknown(),
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
									},
								},
							},
						},
						"volumes": schema.SetNestedAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "Volumes to mount into the container.",
							NestedObject: schema.NestedAttributeObject{
								Validators: []validator.Object{
									&VolumeMountValidator{},
								},
								Attributes: map[string]schema.Attribute{
									"volume": schema.StringAttribute{
										Optional: true,
										MarkdownDescription: "The name of the volume to mount. A volume of this name " +
											"must be specified in the top-level `volumes` attribute.\n\n" +
											"Either this attribute, or `project_path` must be set.",
									},
									"project_path": schema.StringAttribute{
										Optional: true,
										MarkdownDescription: "Path to a directory in the project filesystem.\n\n" +
											"Either this attribute, or `volume` must be set.",
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
				Optional: true,
				Computed: true,
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

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data, current ContainerStackModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := apiext.NewContainerClient(r.client)

	if data.DefaultStack.ValueBool() {
		stack, err := client.GetDefaultStack(ctx, data.ProjectID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("failed to get default stack", err.Error())
			return
		}

		tflog.Debug(ctx, "using project default stack", map[string]any{"stack_id": stack.Id})

		data.ID = types.StringValue(stack.Id)

		updateRequest := data.ToUpdateRequest(ctx, &current, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			tflog.Debug(ctx, "error while building update request")
			return
		}

		_ = providerutil.
			Try[*containerv2.StackResponse](&resp.Diagnostics, "API error while declaring stack").
			DoValResp(r.client.Container().UpdateStack(ctx, *updateRequest))
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		declareRequest := data.ToDeclareRequest(ctx, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			tflog.Debug(ctx, "error while building declare request")
			return
		}

		stack := providerutil.
			Try[*containerv2.StackResponse](&resp.Diagnostics, "API error while declaring stack").
			DoValResp(r.client.Container().DeclareStack(ctx, *declareRequest))
		if resp.Diagnostics.HasError() {
			return
		}

		data.ID = types.StringValue(stack.Id)
	}

	resp.Diagnostics.Append(r.read(ctx, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Resource) read(ctx context.Context, data *ContainerStackModel) (res diag.Diagnostics) {
	stack := providerutil.
		Try[*containerv2.StackResponse](&res, "API error while fetching stack").
		DoValResp(r.client.Container().GetStack(ctx, containerclientv2.GetStackRequest{StackID: data.ID.ValueString()}))

	if res.HasError() {
		return
	}

	res.Append(data.FromAPIModel(ctx, stack, true)...)

	return
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	data := ContainerStackModel{}

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.read(ctx, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData, stateData ContainerStackModel

	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var stack *containerv2.StackResponse

	if stateData.DefaultStack.ValueBool() {
		req := planData.ToUpdateRequest(ctx, &stateData, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}

		stack = providerutil.
			Try[*containerv2.StackResponse](&resp.Diagnostics, "API error while updating stack").
			DoValResp(r.client.Container().UpdateStack(ctx, *req))
	} else {
		req := planData.ToDeclareRequest(ctx, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}

		stack = providerutil.
			Try[*containerv2.StackResponse](&resp.Diagnostics, "API error while declaring stack").
			DoValResp(r.client.Container().DeclareStack(ctx, *req))
	}

	containerModels := planData.ContainerModels(ctx, &resp.Diagnostics)
	for _, service := range stack.Services {
		serviceConfig, ok := containerModels[service.ServiceName]
		if !ok {
			continue
		}

		if serviceConfig.NoRecreateOnChange.ValueBool() {
			continue
		}

		if reflect.DeepEqual(service.DeployedState, service.PendingState) {
			continue
		}

		req := containerclientv2.RecreateServiceRequest{
			StackID:   stack.Id,
			ServiceID: service.Id,
		}

		tflog.Debug(ctx, "recreating service", map[string]any{"stack_id": stack.Id, "service_id": service.Id})

		providerutil.
			Try[any](&resp.Diagnostics, "API error while recreating container").
			DoResp(r.client.Container().RecreateService(ctx, req))
	}

	resp.Diagnostics.Append(r.read(ctx, &stateData)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &stateData)...)
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var stateData ContainerStackModel

	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// The "default" stack has a special role, and we will not delete it even if
	// the user requests it. Instead, we will simply purge all containers and volumes
	// from it
	if stateData.DefaultStack.ValueBool() {
		_ = providerutil.
			Try[*containerv2.StackResponse](&resp.Diagnostics, "API error while removing containers from stack").
			DoValResp(r.client.Container().UpdateStack(ctx, *stateData.ToDeletePatchRequest(ctx, &resp.Diagnostics)))
	} else {
		resp.Diagnostics.AddError("not implemented", "removing non-default stacks is not supported, yet")
	}
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// TODO
}
