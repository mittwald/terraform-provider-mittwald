package containerregistryresource

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
	resp.TypeName = req.ProviderTypeName + "_container_registry"
}
func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	builder := common.AttributeBuilderFor("container_registry")
	resp.Schema = schema.Schema{
		MarkdownDescription: "This resource models a container registry.\n\nIt may be used to configure private registries for use in mittwald_container_stack resources.",

		Attributes: map[string]schema.Attribute{
			"id":         builder.Id(),
			"project_id": builder.ProjectId(),
			"default_registry": schema.BoolAttribute{
				Description: "Describes if this registry is one of the default registries",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				Description: "Description for the registry",
				Required:    true,
			},
			"uri": schema.StringAttribute{
				MarkdownDescription: "Hostname for the registry, for example `gitlab.example.com`",
				Required:            true,
			},
			"credentials": schema.SingleNestedAttribute{
				MarkdownDescription: "Credentials for the registry",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"username": schema.StringAttribute{
						Description: "Username for the registry",
						Required:    true,
					},
					"password": schema.StringAttribute{
						Description: "Password or access token for the registry",
						Required:    true,
						Sensitive:   true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
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
	var data ContainerRegistryModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := apiext.NewContainerClient(r.client)

	existing, _ := client.GetRegistryByName(ctx, data.ProjectID.ValueString(), data.URI.ValueString())
	if existing != nil {
		data.ID = types.StringValue(existing.Id)
		data.DefaultRegistry = types.BoolValue(true)

		updateRequest := data.ToUpdateRequest(ctx, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}

		providerutil.
			Try[*containerv2.Registry](&resp.Diagnostics, "API error while updating registry").
			DoResp(r.client.Container().UpdateRegistry(ctx, *updateRequest))
	} else {
		data.DefaultRegistry = types.BoolValue(false)

		createRequest := data.ToCreateRequest(ctx, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}

		registry := providerutil.
			Try[*containerv2.Registry](&resp.Diagnostics, "API error while declaring registry").
			DoValResp(r.client.Container().CreateRegistry(ctx, *createRequest))

		data.ID = types.StringValue(registry.Id)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.read(ctx, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Resource) read(ctx context.Context, data *ContainerRegistryModel) (res diag.Diagnostics) {
	registry := providerutil.
		Try[*containerv2.Registry](&res, "API error while fetching registry").
		DoValResp(r.client.Container().GetRegistry(ctx, containerclientv2.GetRegistryRequest{RegistryID: data.ID.ValueString()}))

	if res.HasError() {
		return
	}

	res.Append(data.FromAPIModel(ctx, registry)...)

	return
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	data := ContainerRegistryModel{}

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.read(ctx, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData, stateData ContainerRegistryModel

	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updateRequest := planData.ToUpdateRequest(ctx, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	providerutil.
		Try[*containerv2.Registry](&resp.Diagnostics, "API error while updating registry").
		DoResp(r.client.Container().UpdateRegistry(ctx, *updateRequest))

	resp.Diagnostics.Append(r.read(ctx, &stateData)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &stateData)...)
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var stateData ContainerRegistryModel

	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// The "default" stack has a special role, and we will not delete it even if
	// the user requests it. Instead, we will simply purge all containers and volumes
	// from it
	if stateData.DefaultRegistry.ValueBool() {
		stateData.Credentials = types.ObjectNull(containerRegistryCredentialsAttributeTypes)

		updateRequest := stateData.ToUpdateRequest(ctx, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}

		providerutil.
			Try[*containerv2.StackResponse](&resp.Diagnostics, "API error while resetting registry").
			DoResp(r.client.Container().UpdateRegistry(ctx, *updateRequest))
	} else {
		providerutil.
			Try[*containerv2.StackResponse](&resp.Diagnostics, "API error while resetting registry").
			DoResp(r.client.Container().DeleteRegistry(ctx, *stateData.ToDeleteRequest()))
	}
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

}
