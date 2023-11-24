package cronjobresource

import (
	"context"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/terraform-provider-mittwald/api/mittwaldv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
	"github.com/mittwald/terraform-provider-mittwald/internal/ptrutil"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &Resource{}
var _ resource.ResourceWithImportState = &Resource{}

func New() resource.Resource {
	return &Resource{}
}

type Resource struct {
	client mittwaldv2.ClientBuilder
}

func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cronjob"
}

func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "This resource models a cron job.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The generated cron job ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the project the cron job belongs to",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"app_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the app the cron job belongs to",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Description for your cron job",
			},
			"destination": schema.SingleNestedAttribute{
				Required: true,
				Validators: []validator.Object{
					&cronjobDestinationValidator{},
				},
				Attributes: map[string]schema.Attribute{
					"url": schema.StringAttribute{
						Optional: true,
					},
					"command": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"interpreter": schema.StringAttribute{
								Required: true,
							},
							"path": schema.StringAttribute{
								Required: true,
							},
							"parameters": schema.ListAttribute{
								Optional:    true,
								ElementType: types.StringType,
							},
						},
					},
				},
			},
			"interval": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The interval of the cron job; this should be a cron expression",
			},
			"email": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The email address to send the cron job's output to",
			},
		},
	}
}

func (r *Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.ClientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	createCronjobBody := mittwaldv2.CronjobCreateCronjobJSONRequestBody{
		Description: data.Description.ValueString(),
		Active:      true,
		AppId:       uuid.MustParse(data.AppID.ValueString()),
		Interval:    data.Interval.ValueString(),
		Destination: mittwaldv2.DeMittwaldV1CronjobCronjobRequest_Destination{},
	}

	dest := data.GetDestination(ctx, resp.Diagnostics)
	if url, ok := dest.GetURL(ctx, resp.Diagnostics); ok {
		err := createCronjobBody.Destination.FromDeMittwaldV1CronjobCronjobUrl(mittwaldv2.DeMittwaldV1CronjobCronjobUrl{
			Url: url,
		})
		if err != nil {
			resp.Diagnostics.AddError("Mapping error while building cron job request", err.Error())
			return
		}
	}

	if cmd, ok := dest.GetCommand(ctx, resp.Diagnostics); ok {
		err := createCronjobBody.Destination.FromDeMittwaldV1CronjobCronjobCommand(mittwaldv2.DeMittwaldV1CronjobCronjobCommand{
			Interpreter: cmd.Interpreter.ValueString(),
			Path:        cmd.Path.ValueString(),
			Parameters:  cmd.ParametersAsStr(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Mapping error while building cron job request", err.Error())
			return
		}
	}

	if !data.Email.IsNull() {
		e := openapi_types.Email(data.Email.ValueString())
		createCronjobBody.Email = &e
	}

	id, err := r.client.Cronjob().CreateCronjob(ctx, data.ProjectID.ValueString(), createCronjobBody)
	if err != nil {
		resp.Diagnostics.AddError("API error while creating cron job", err.Error())
		return
	}

	data.ID = types.StringValue(id)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData, stateData ResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)

	body := mittwaldv2.CronjobUpdateCronjobJSONRequestBody{}

	if !planData.Description.Equal(stateData.Description) {
		if !planData.Description.IsNull() {
			body.Description = ptrutil.To(planData.Description.ValueString())
		} else {
			// no known way to unset a description. :(
		}
	}

	if !planData.Interval.Equal(stateData.Interval) {
		body.Interval = ptrutil.To(planData.Interval.ValueString())
	}

	if !planData.Email.Equal(stateData.Email) {
		if !planData.Email.IsNull() {
			body.Email = ptrutil.To(openapi_types.Email(planData.Email.ValueString()))
		} else {
			// no known way to unset the email address :(
		}
	}

	if !planData.Destination.Equal(stateData.Destination) {
		body.Destination = &mittwaldv2.CronjobUpdateCronjobJSONBody_Destination{}

		dest := planData.GetDestination(ctx, resp.Diagnostics)
		if url, ok := dest.GetURL(ctx, resp.Diagnostics); ok {
			err := body.Destination.FromDeMittwaldV1CronjobCronjobUrl(mittwaldv2.DeMittwaldV1CronjobCronjobUrl{
				Url: url,
			})
			if err != nil {
				resp.Diagnostics.AddError("Mapping error while building cron job request", err.Error())
				return
			}
		}

		if cmd, ok := dest.GetCommand(ctx, resp.Diagnostics); ok {
			err := body.Destination.FromDeMittwaldV1CronjobCronjobCommand(mittwaldv2.DeMittwaldV1CronjobCronjobCommand{
				Interpreter: cmd.Interpreter.ValueString(),
				Path:        cmd.Path.ValueString(),
				Parameters:  cmd.ParametersAsStr(),
			})
			if err != nil {
				resp.Diagnostics.AddError("Mapping error while building cron job request", err.Error())
				return
			}
		}
	}

	if err := r.client.Cronjob().UpdateCronjob(ctx, planData.ID.ValueString(), body); err != nil {
		resp.Diagnostics.AddError("API error while updating cron job", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &stateData)...)
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if err := r.client.Cronjob().DeleteCronjob(ctx, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("API error while deleting cron job", err.Error())
		return
	}
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
