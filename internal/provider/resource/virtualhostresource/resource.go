package virtualhostresource

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/domainclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/ingressv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiext"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/resource/common"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &Resource{}
var _ resource.ResourceWithImportState = &Resource{}

func New() resource.Resource {
	return &Resource{}
}

type Resource struct {
	client mittwaldv2.Client
}

func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtualhost"
}

func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	builder := common.AttributeBuilderFor("virtualhost")
	resp.Schema = schema.Schema{
		MarkdownDescription: "This resource models a virtualhost.",

		Attributes: map[string]schema.Attribute{
			"id":         builder.Id(),
			"project_id": builder.ProjectId(),
			"hostname": schema.StringAttribute{
				Description: "The desired hostname for the virtualhost.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"default": schema.BoolAttribute{
				MarkdownDescription: "Describes if this vhost is the project's default virtual host. The default virtual host will never be deleted. If you attempt to delete this resource via terraform, it will simply revert to an unmanaged state.",
				Computed:            true,
			},
			"paths": schema.MapNestedAttribute{
				Description: "The desired paths for the virtualhost.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"app": schema.StringAttribute{
							MarkdownDescription: "The ID of an app installation that this path should point to.",
							Optional:            true,
						},
						"redirect": schema.StringAttribute{
							MarkdownDescription: "The URL to redirect to.",
							Optional:            true,
						},
						"container": schema.SingleNestedAttribute{
							Optional: true,
							Attributes: map[string]schema.Attribute{
								"container_id": schema.StringAttribute{
									MarkdownDescription: "The ID of a container (!= the ID of a container *stack*) that this path should point to.",
									Required:            true,
								},
								"port": schema.StringAttribute{
									MarkdownDescription: "A port number/protocol combination of the referenced container that traffic should be redirected to (example: `8080/tcp`)",
									Required:            true,
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.ClientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ResourceModel

	client := apiext.NewDomainClient(r.client)

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// ignoring error on purpose
	existing, _ := client.GetIngressByName(ctx, data.ProjectID.ValueString(), data.Hostname.ValueString())
	if existing != nil && existing.IsDefault {
		var current ResourceModel

		resp.Diagnostics.Append(current.FromAPIModel(ctx, existing)...)
		if resp.Diagnostics.HasError() {
			return
		}

		data.ID = types.StringValue(existing.Id)

		body := data.ToUpdateRequest(ctx, &resp.Diagnostics, &current)
		if resp.Diagnostics.HasError() {
			return
		}

		providerutil.
			Try[any](&resp.Diagnostics, "API error while updating virtual host").
			DoResp(r.client.Domain().UpdateIngressPaths(ctx, body))
	} else {
		ingress := providerutil.
			Try[*domainclientv2.CreateIngressResponse](&resp.Diagnostics, "API error while creating virtual host").
			DoValResp(client.CreateIngress(ctx, data.ToCreateRequest(ctx, &resp.Diagnostics)))
		if resp.Diagnostics.HasError() {
			return
		}

		data.ID = types.StringValue(ingress.Id)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.read(ctx, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	data := ResourceModel{}

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.read(ctx, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Resource) read(ctx context.Context, data *ResourceModel) (res diag.Diagnostics) {
	ingress := providerutil.
		Try[*ingressv2.Ingress](&res, "API error while fetching ingress").
		DoValResp(r.client.Domain().GetIngress(ctx, domainclientv2.GetIngressRequest{IngressID: data.ID.ValueString()}))

	if res.HasError() {
		return
	}

	res.Append(data.FromAPIModel(ctx, ingress)...)

	return
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData, stateData ResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)

	body := planData.ToUpdateRequest(ctx, &resp.Diagnostics, &stateData)
	if resp.Diagnostics.HasError() {
		return
	}

	providerutil.
		Try[any](&resp.Diagnostics, "API error while updating virtual host").
		DoResp(r.client.Domain().UpdateIngressPaths(ctx, body))

	resp.Diagnostics.Append(r.read(ctx, &stateData)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &stateData)...)
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if data.Default.ValueBool() {
		tflog.Debug(ctx, "virtualhost is default host; not deleting")
		return
	}

	providerutil.
		Try[any](&resp.Diagnostics, "API error while deleting virtual host").
		DoResp(r.client.Domain().DeleteIngress(ctx, data.ToDeleteRequest()))
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
