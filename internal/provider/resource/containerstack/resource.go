package containerstackresource

import (
	"context"
	"encoding/json"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/containerclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/containerv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiext"
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
	builder := common.AttributeBuilderFor("redis_database")
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
						"image": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The image to use for the container.",
						},
						"description": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "A description for the container.",
						},
						"command": schema.ListAttribute{
							Optional:            true,
							MarkdownDescription: "The command to run inside the container.",
							ElementType:         types.StringType,
						},
						"entrypoint": schema.ListAttribute{
							Optional:            true,
							MarkdownDescription: "The entrypoint to use for the container.",
							ElementType:         types.StringType,
						},
						"environment": schema.MapAttribute{
							Optional:            true,
							MarkdownDescription: "A map of environment variables to set inside the container.",
							ElementType:         types.StringType,
						},
						"ports": schema.SetNestedAttribute{
							Optional:            true,
							MarkdownDescription: "A ports to expose from the container. Follows the format `<public-port>:<container-port>/<protocol>`.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"container_port": schema.Int32Attribute{
										Required:            true,
										MarkdownDescription: "The container port to expose.",
									},
									"public_port": schema.Int32Attribute{
										Optional:            true,
										Computed:            true,
										MarkdownDescription: "The public port to expose; will default to the same value as `container_port`.",
									},
									"protocol": schema.StringAttribute{
										Optional:            true,
										Computed:            true,
										MarkdownDescription: "The protocol to use for the port. Defaults to `tcp`.",
										Default:             stringdefault.StaticString("tcp"),
									},
								},
							},
						},
						"volumes": schema.SetNestedAttribute{
							Optional:            true,
							MarkdownDescription: "A list of volumes to mount into the container.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"volume": schema.StringAttribute{
										Optional:            true,
										MarkdownDescription: "The name of the volume to mount.",
									},
									"project_path": schema.StringAttribute{
										Required:            true,
										MarkdownDescription: "Path to a directory in the project filesystem.",
									},
									"mount_path": schema.StringAttribute{
										Required:            true,
										MarkdownDescription: "The path to mount the volume to.",
									},
								},
							},
						},
					},
				},
			},
			"volumes": schema.MapNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{},
				},
			},
		},
	}
}

func (r *Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.ClientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ContainerStackModel

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
	}

	declareRequest := data.ToDeclareRequest(ctx, &resp.Diagnostics)

	j, _ := json.Marshal(declareRequest)
	tflog.Debug(ctx, "Creating container", map[string]any{"request": string(j)})

	stack := providerutil.
		Try[*containerv2.StackResponse](&resp.Diagnostics, "API error while declaring stack").
		DoValResp(r.client.Container().DeclareStack(ctx, *data.ToDeclareRequest(ctx, &resp.Diagnostics)))
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(stack.Id)

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

	res.Append(data.FromAPIModel(ctx, stack)...)

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

	_ = providerutil.
		Try[*containerv2.StackResponse](&resp.Diagnostics, "API error while declaring stack").
		DoValResp(r.client.Container().DeclareStack(ctx, *planData.ToDeclareRequest(ctx, &resp.Diagnostics)))

	resp.Diagnostics.Append(r.read(ctx, &stateData)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &stateData)...)
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

}
