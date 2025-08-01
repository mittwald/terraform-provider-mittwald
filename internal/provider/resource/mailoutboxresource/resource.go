package mailoutboxresource

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/mailclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/mailv2"
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
	resp.TypeName = req.ProviderTypeName + "_mail_outbox"
}

func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	builder := common.AttributeBuilderFor("mail_outbox")
	resp.Schema = schema.Schema{
		MarkdownDescription: "This resource models a mail outbox.",

		Attributes: map[string]schema.Attribute{
			"id":         builder.Id(),
			"project_id": builder.ProjectId(),
			"description": schema.StringAttribute{
				Description: "The description of the mail outbox.",
				Required:    true,
			},
			"password": schema.StringAttribute{
				Description: "The password for the mail outbox.",
				Required:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.Mail()

	// Create new mail outbox
	createResp, httpResp, err := client.CreateDeliverybox(ctx, data.ToCreateRequest())
	createResp = providerutil.
		Try[*mailclientv2.CreateDeliveryboxResponse](&resp.Diagnostics, "Error creating mail outbox").
		DoValResp(createResp, httpResp, err)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get the created mail outbox to get all the details
	mailOutbox, httpResp, err := client.GetDeliveryBox(ctx, mailclientv2.GetDeliveryBoxRequest{
		DeliveryBoxID: createResp.Id,
	})
	mailOutbox = providerutil.
		Try[*mailv2.Deliverybox](&resp.Diagnostics, "Error retrieving created mail outbox").
		DoValResp(mailOutbox, httpResp, err)

	if resp.Diagnostics.HasError() {
		return
	}

	// Map response body to schema and populate computed values
	data.FromAPIModel(ctx, mailOutbox)

	// Set state to fully populated data
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Trace(ctx, "created mail outbox resource")
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.read(ctx, &data)...)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Resource) read(ctx context.Context, data *ResourceModel) (res diag.Diagnostics) {
	// Get mail outbox
	mailOutbox, httpResp, err := r.client.Mail().GetDeliveryBox(ctx, mailclientv2.GetDeliveryBoxRequest{
		DeliveryBoxID: data.ID.ValueString(),
	})
	mailOutbox = providerutil.
		Try[*mailv2.Deliverybox](&res, "Error reading mail outbox").
		IgnoreNotFound().
		DoValResp(mailOutbox, httpResp, err)

	if res.HasError() {
		return
	}

	// Map response body to schema
	data.FromAPIModel(ctx, mailOutbox)

	return
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ResourceModel
	var state ResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.Mail()

	// Update description if changed
	if !data.Description.Equal(state.Description) {
		httpResp, err := client.UpdateDeliveryBoxDescription(ctx, data.ToUpdateDescriptionRequest())
		providerutil.
			Try[any](&resp.Diagnostics, "Error updating mail outbox").
			DoResp(httpResp, err)

		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Update password if changed
	if !data.Password.Equal(state.Password) {
		httpResp, err := client.UpdateDeliveryBoxPassword(ctx, data.ToUpdatePasswordRequest())
		providerutil.
			Try[any](&resp.Diagnostics, "Error updating mail outbox password").
			DoResp(httpResp, err)

		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Refresh state
	resp.Diagnostics.Append(r.read(ctx, &data)...)

	// Set state to fully populated data
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Trace(ctx, "updated mail outbox resource")
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.Mail()

	// Delete mail outbox
	httpResp, err := client.DeleteDeliveryBox(ctx, data.ToDeleteRequest())
	providerutil.
		Try[any](&resp.Diagnostics, "Error deleting mail outbox").
		DoResp(httpResp, err)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "deleted mail outbox resource")
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
