package projectresource

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/projectclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/projectv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiext"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiutils"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/resource/common"
	"time"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &Resource{}
var _ resource.ResourceWithImportState = &Resource{}

func New() resource.Resource {
	return &Resource{}
}

// Resource defines the resource implementation.
type Resource struct {
	client mittwaldv2.Client
}

func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	builder := common.AttributeBuilderFor("project")
	resp.Schema = schema.Schema{
		MarkdownDescription: "This resource models a project on the mittwald cloud platform; a project is either provisioned on a server (in which case a `server_id` is required), or as a stand-alone project (currently not supported).",

		Attributes: map[string]schema.Attribute{
			"server_id": schema.StringAttribute{
				MarkdownDescription: "ID of the server this project belongs to",
				Optional:            true,
			},
			"id": builder.Id(),
			"short_id": schema.StringAttribute{
				MarkdownDescription: "The short ID of the project",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": builder.Description(),
			"directories": schema.MapAttribute{
				Computed:            true,
				MarkdownDescription: "Contains a map of data directories within the project",
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"default_ips": schema.ListAttribute{
				Computed:            true,
				MarkdownDescription: "Contains a list of default IP addresses for the project",
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
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

	client := r.client.Project()

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(data.Validate()...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectResponse := providerutil.
		Try[*projectclientv2.CreateProjectResponse](&resp.Diagnostics, "error while creating project").
		DoValResp(client.CreateProject(
			ctx,
			data.ToCreateRequest(),
		))

	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(projectResponse.Id)

	resp.Diagnostics.Append(r.read(ctx, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	readCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	resp.Diagnostics.Append(r.read(readCtx, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Resource) read(ctx context.Context, data *ResourceModel) (res diag.Diagnostics) {
	client := apiext.NewProjectClient(r.client)

	pr := providerutil.
		Try[*projectv2.Project](&res, "error while reading project").
		IgnoreNotFound().
		DoVal(apiutils.Poll(ctx, apiutils.PollOpts{}, client.GetProject, projectclientv2.GetProjectRequest{ProjectID: data.ID.ValueString()}))

	ips := providerutil.
		Try[[]string](&res, "error while reading project ips").
		IgnoreNotFound().
		DoVal(client.GetProjectDefaultIPs(ctx, data.ID.ValueString()))

	if res.HasError() {
		return
	}

	res.Append(data.FromAPIModel(ctx, pr, ips)...)

	return
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var dataPlan, dataState ResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &dataPlan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &dataState)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !dataPlan.Description.Equal(dataState.Description) {
		updateReq := projectclientv2.UpdateProjectDescriptionRequest{
			ProjectID: dataState.ID.ValueString(),
			Body: projectclientv2.UpdateProjectDescriptionRequestBody{
				Description: dataPlan.Description.ValueString(),
			},
		}
		if _, err := r.client.Project().UpdateProjectDescription(ctx, updateReq); err != nil {
			resp.Diagnostics.AddError("Error while updating project description", err.Error())
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &dataPlan)...)
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	deleteReq := projectclientv2.DeleteProjectRequest{ProjectID: data.ID.ValueString()}

	providerutil.
		Try[any](&resp.Diagnostics, "error while deleting project").
		IgnoreNotFound().
		DoResp(r.client.Project().DeleteProject(ctx, deleteReq))
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
