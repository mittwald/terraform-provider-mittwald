package projectresource

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/projectclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/projectv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiext"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiutils"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/resource/common"
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

func (r *Resource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	builder := common.AttributeBuilderFor("project")
	resp.Schema = schema.Schema{
		MarkdownDescription: "This resource models a project on the mittwald cloud platform; a project is either provisioned on a server (in which case a `server_id` is required), or as a stand-alone project (currently not supported).",

		Attributes: map[string]schema.Attribute{
			"server_id": schema.StringAttribute{
				MarkdownDescription: "ID of the server this project belongs to. Must be a full UUID (not a short ID like s-XXXXXX).",
				Optional:            true,
				Validators: []validator.String{
					&common.UUIDValidator{},
				},
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

		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				CreateDescription: "Time to wait for the project to be created. This includes waiting for the " +
					"project's default ingress (and with it, the `default_ips` attribute) to become available; " +
					"defaults to 10 minutes.",
				Read:            true,
				ReadDescription: "Time to wait when reading the project's current state; defaults to 2 minutes.",
			}),
		},
	}
}

func (r *Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.ClientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ResourceModelWithTimeouts

	client := r.client.Project()

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(data.Validate()...)

	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := data.Timeouts.Create(ctx, DefaultCreateTimeout)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

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

	resp.Diagnostics.Append(r.readAfterCreate(ctx, &data.ResourceModel)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ResourceModelWithTimeouts

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, diags := data.Timeouts.Read(ctx, DefaultReadTimeout)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	readCtx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	found, diags := r.read(readCtx, &data.ResourceModel)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// read refreshes the given model from the API. It reports whether the project
// still exists; if it does not, the model is left untouched and the caller
// should remove the resource from the state.
func (r *Resource) read(ctx context.Context, data *ResourceModel) (bool, diag.Diagnostics) {
	var res diag.Diagnostics

	client := apiext.NewProjectClient(r.client)

	pr := providerutil.
		Try[*projectv2.Project](&res, "error while reading project").
		IgnoreNotFound().
		DoValResp(client.GetProject(ctx, projectclientv2.GetProjectRequest{ProjectID: data.ID.ValueString()}))

	if res.HasError() || pr == nil {
		return false, res
	}

	// A missing default ingress is not an error during a refresh; it just means
	// that the project's IP addresses are not available (yet).
	ips := PollDefaultIPs(ctx, client, data.ID.ValueString(), readTimeoutHint, &res)

	if res.HasError() {
		return false, res
	}

	res.Append(data.FromAPIModel(ctx, pr, ips)...)

	return true, res
}

// readAfterCreate is like read, but tolerates the project itself not being
// visible right away, and waits longer for the default IPs to become available.
// This is necessary because immediately after project creation, neither the
// project nor its default ingress may be visible yet.
func (r *Resource) readAfterCreate(ctx context.Context, data *ResourceModel) (res diag.Diagnostics) {
	client := apiext.NewProjectClient(r.client)

	pr, err := apiutils.PollRequest(ctx, apiutils.PollOpts{}, client.GetProject, projectclientv2.GetProjectRequest{ProjectID: data.ID.ValueString()})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			res.AddError(
				"error while reading project",
				"the project was created, but did not become readable in time. "+createTimeoutHint,
			)
		} else {
			res.AddError("error while reading project", err.Error())
		}

		return
	}

	if pr == nil {
		res.AddError("error while reading project", "the project was created, but could not be read back")
		return
	}

	// If the default ingress does not show up in time, continue with an empty
	// list of IP addresses; the project itself has been created successfully,
	// and failing here would leave the resource tainted (and all computed
	// attributes unknown, which Terraform rejects outright).
	ips := PollDefaultIPs(ctx, client, data.ID.ValueString(), createTimeoutHint, &res)

	if res.HasError() {
		return
	}

	res.Append(data.FromAPIModel(ctx, pr, ips)...)

	return
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var dataPlan, dataState ResourceModelWithTimeouts

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
	var data ResourceModelWithTimeouts

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
