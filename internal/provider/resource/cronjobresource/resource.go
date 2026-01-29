package cronjobresource

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/cronjobclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/cronjobv2"
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
	resp.TypeName = req.ProviderTypeName + "_cronjob"
}

func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	builder := common.AttributeBuilderFor("cronjob")
	resp.Schema = schema.Schema{
		MarkdownDescription: "This resource models a cron job.",

		Attributes: map[string]schema.Attribute{
			"id":          builder.Id(),
			"project_id":  builder.ProjectId(),
			"app_id":      builder.AppId(),
			"description": builder.Description(),
			"destination": modelDestinationSchema,
			"interval": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The interval of the cron job; this should be a cron expression",
			},
			"email": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The email address to send the cron job's output to",
			},
			"timezone": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The timezone to use for the cron job execution schedule (e.g., `Europe/Berlin`, `America/New_York`)",
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

	if resp.Diagnostics.HasError() {
		return
	}

	cronjob := providerutil.
		Try[*cronjobclientv2.CreateCronjobResponse](&resp.Diagnostics, "API error while updating cron job").
		DoValResp(r.client.Cronjob().CreateCronjob(ctx, data.ToCreateRequest(ctx, &resp.Diagnostics)))

	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(cronjob.Id)

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
	cronjob := providerutil.
		Try[*cronjobv2.Cronjob](&res, "API error while fetching cron job").
		DoValResp(r.client.Cronjob().GetCronjob(ctx, cronjobclientv2.GetCronjobRequest{CronjobID: data.ID.ValueString()}))

	if res.HasError() {
		return
	}

	res.Append(data.FromAPIModel(ctx, cronjob)...)

	return
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData, stateData ResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)

	if resp.Diagnostics.HasError() {
		return
	}

	providerutil.
		Try[any](&resp.Diagnostics, "API error while updating cron job").
		DoResp(r.client.Cronjob().UpdateCronjob(ctx, planData.ToUpdateRequest(ctx, &resp.Diagnostics, &stateData)))

	resp.Diagnostics.Append(r.read(ctx, &stateData)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &stateData)...)
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	providerutil.
		Try[any](&resp.Diagnostics, "API error while deleting cron job").
		DoResp(r.client.Cronjob().DeleteCronjob(ctx, data.ToDeleteRequest()))
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
