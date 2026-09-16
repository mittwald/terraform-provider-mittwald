package aiapikeyresource

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/aihostingclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/contractclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/projectclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/aihostingv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/contractv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/projectv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &Resource{}
var _ resource.ResourceWithImportState = &Resource{}
var _ resource.ResourceWithValidateConfig = &Resource{}
var _ resource.ResourceWithModifyPlan = &Resource{}

// New creates a new AI API key resource.
func New() resource.Resource {
	return &Resource{}
}

// Resource defines the AI API key resource implementation.
type Resource struct {
	client mittwaldv2.Client
}

// Metadata returns the resource type name.
func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ai_api_key"
}

// Schema defines the resource schema.
func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "This resource manages an AI API key for the mittwald AI hosting service.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The generated API key ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"customer_id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The ID of the customer to create the API key for. Either `customer_id` or `project_id` must be set.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The ID of the project to create the API key for. Either `customer_id` or `project_id` must be set.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"contract_id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The contract ID of the AI hosting plan (see `mittwald_ai`'s `contract_id` attribute) that this key should be scoped to. If not set, the customer's only AI hosting plan is used; if the customer has multiple AI hosting plans, this attribute is required.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the API key.",
			},
			"api_key": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "The generated API key value. This is only available after creation and is sensitive.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// ValidateConfig validates the resource configuration.
func (r *Resource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data ResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	customerIDSet := !data.CustomerID.IsNull()
	projectIDSet := !data.ProjectID.IsNull()

	if !customerIDSet && !projectIDSet {
		resp.Diagnostics.AddAttributeError(
			path.Root("customer_id"),
			"Missing required argument",
			"At least one of \"customer_id\" or \"project_id\" must be set.",
		)
		return
	}
}

func (r *Resource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var data ResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	customerIDIsNullOrUnknown := data.CustomerID.IsNull() || data.CustomerID.IsUnknown()
	projectIDIsNullOrUnknown := data.ProjectID.IsNull() || data.ProjectID.IsUnknown()
	if customerIDIsNullOrUnknown && !projectIDIsNullOrUnknown {
		project, err := r.getProject(ctx, data.ProjectID.ValueString())
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("project_id"), "Error retrieving project", fmt.Sprintf("Error retrieving project with ID %s: %s", data.ProjectID.ValueString(), err.Error()))
			return
		}

		data.CustomerID = types.StringValue(project.CustomerId)

		resp.Diagnostics.Append(resp.Plan.Set(ctx, data)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
}

// Configure sets up the resource with the provider client.
func (r *Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.ClientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

// Create creates a new AI API key.
func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine the customer ID - either from customer_id attribute or by looking up the project
	customerID, projectID := r.resolveCustomerAndProjectID(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	planID, contractID, diags := r.resolvePlan(ctx, customerID, data.ContractID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build the request body
	body := aihostingclientv2.CustomerCreateKeyRequestBody{
		Name:   data.Name.ValueString(),
		PlanId: planID,
	}

	if projectID != "" {
		body.ProjectId = &projectID
	}

	// Create the API key
	createReq := aihostingclientv2.CustomerCreateKeyRequest{
		CustomerID: customerID,
		Body:       body,
	}

	key, httpResp, err := r.client.AIHosting().CustomerCreateKey(ctx, createReq)
	key = providerutil.
		Try[*aihostingv2.Key](&resp.Diagnostics, "Error creating AI API key").
		DoValResp(key, httpResp, err)

	if resp.Diagnostics.HasError() {
		return
	}

	// Map response to model
	data.FromAPIModel(key)
	data.ContractID = types.StringValue(contractID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Trace(ctx, "created AI API key resource")
}

// resolvePlan determines the AI hosting plan (and its associated contract)
// that a key should be scoped to. If configuredContractID is empty, the
// customer's only AI hosting plan is used; if the customer has more than one
// plan, an error is returned since the plan cannot be determined
// unambiguously. If configuredContractID is set, the plan whose contract
// matches it is used.
func (r *Resource) resolvePlan(ctx context.Context, customerID, configuredContractID string) (planID, contractID string, diags diag.Diagnostics) {
	limit := int64(100)
	plans := providerutil.
		Try[*aihostingv2.CustomerPlans](&diags, "Error listing AI hosting plans").
		DoValResp(r.client.AIHosting().CustomerGetPlans(ctx, aihostingclientv2.CustomerGetPlansRequest{
			CustomerID: customerID,
			Limit:      &limit,
		}))

	if diags.HasError() || plans == nil {
		return
	}

	if configuredContractID == "" {
		switch len(plans.Plans) {
		case 0:
			diags.AddAttributeError(
				path.Root("contract_id"),
				"No AI hosting plan found",
				fmt.Sprintf("Customer %s has no AI hosting plan. Create a mittwald_ai resource for this customer first, or set contract_id explicitly.", customerID),
			)
			return
		case 1:
			planID = plans.Plans[0].PlanId
		default:
			diags.AddAttributeError(
				path.Root("contract_id"),
				"Ambiguous AI hosting plan",
				fmt.Sprintf("Customer %s has %d AI hosting plans. Set contract_id to the contract_id of the mittwald_ai resource this key should be scoped to.", customerID, len(plans.Plans)),
			)
			return
		}

		contractID, diags = r.contractIDForPlan(ctx, customerID, planID)
		return
	}

	for _, plan := range plans.Plans {
		var id string
		id, diags = r.contractIDForPlan(ctx, customerID, plan.PlanId)
		if diags.HasError() {
			return
		}

		if id == configuredContractID {
			planID = plan.PlanId
			contractID = id
			return
		}
	}

	diags.AddAttributeError(
		path.Root("contract_id"),
		"AI hosting plan not found",
		fmt.Sprintf("No AI hosting plan found for contract %s on customer %s.", configuredContractID, customerID),
	)

	return
}

// contractIDForPlan resolves the contract ID that belongs to the given AI
// hosting plan.
func (r *Resource) contractIDForPlan(ctx context.Context, customerID, planID string) (string, diag.Diagnostics) {
	var diags diag.Diagnostics

	contract := providerutil.
		Try[*contractv2.Contract](&diags, "Error reading AI hosting contract").
		DoValResp(r.client.Contract().GetDetailOfContractByAIHosting(ctx, contractclientv2.GetDetailOfContractByAIHostingRequest{
			CustomerID:  customerID,
			AIHostingID: planID,
		}))

	if diags.HasError() || contract == nil {
		return "", diags
	}

	return contract.ContractId, diags
}

// Read reads the current state of the AI API key.
func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	customerID, _ := r.resolveCustomerAndProjectID(ctx, &data, &resp.Diagnostics)
	if customerID == "" {
		resp.Diagnostics.AddError(
			"Unable to determine customer ID",
			"Could not determine the customer ID for reading the AI API key.",
		)
		return
	}

	// Get the API key
	getReq := aihostingclientv2.CustomerGetKeyRequest{
		CustomerID: customerID,
		KeyID:      data.ID.ValueString(),
	}

	key, httpResp, err := r.client.AIHosting().CustomerGetKey(ctx, getReq)
	key = providerutil.
		Try[*aihostingv2.Key](&resp.Diagnostics, "Error reading AI API key").
		IgnoreNotFound().
		DoValResp(key, httpResp, err)

	if resp.Diagnostics.HasError() {
		return
	}

	if key == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Preserve the API key value from state since the API doesn't return it on read
	apiKey := data.APIKey

	data.FromAPIModel(key)

	// Restore the API key from state
	data.APIKey = apiKey

	contractID, diags := r.contractIDForPlan(ctx, customerID, key.PlanId)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.ContractID = types.StringValue(contractID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the AI API key.
func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ResourceModel
	var state ResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	customerID, _ := r.resolveCustomerAndProjectID(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update the API key name if changed
	if !data.Name.Equal(state.Name) {
		updateReq := aihostingclientv2.CustomerUpdateKeyRequest{
			CustomerID: customerID,
			KeyID:      data.ID.ValueString(),
			Body: aihostingclientv2.CustomerUpdateKeyRequestBody{
				Name: data.Name.ValueStringPointer(),
			},
		}

		key, httpResp, err := r.client.AIHosting().CustomerUpdateKey(ctx, updateReq)
		key = providerutil.
			Try[*aihostingv2.Key](&resp.Diagnostics, "Error updating AI API key").
			DoValResp(key, httpResp, err)

		if resp.Diagnostics.HasError() {
			return
		}

		// Preserve the API key value from state
		apiKey := state.APIKey

		data.FromAPIModel(key)
		data.APIKey = apiKey
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Trace(ctx, "updated AI API key resource")
}

// Delete deletes the AI API key.
func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	customerID, _ := r.resolveCustomerAndProjectID(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if customerID == "" {
		resp.Diagnostics.AddError(
			"Unable to determine customer ID",
			"Could not determine the customer ID for deleting the AI API key.",
		)
		return
	}

	// Delete the API key
	deleteReq := aihostingclientv2.CustomerDeleteKeyRequest{
		CustomerID: customerID,
		KeyID:      data.ID.ValueString(),
	}

	httpResp, err := r.client.AIHosting().CustomerDeleteKey(ctx, deleteReq)
	providerutil.
		Try[any](&resp.Diagnostics, "Error deleting AI API key").
		DoResp(httpResp, err)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "deleted AI API key resource")
}

// ImportState imports an existing AI API key into Terraform state.
func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// resolveCustomerAndProjectID determines the customer ID and project ID from the resource data.
func (r *Resource) resolveCustomerAndProjectID(ctx context.Context, data *ResourceModel, d *diag.Diagnostics) (customerID, projectID string) {
	if !data.CustomerID.IsNull() && !data.CustomerID.IsUnknown() {
		customerID = data.CustomerID.ValueString()
	}

	if !data.ProjectID.IsNull() && !data.ProjectID.IsUnknown() {
		projectID = data.ProjectID.ValueString()

		if customerID == "" {
			// Look up the customer ID from the project
			project, err := r.getProject(ctx, projectID)
			if err != nil {
				d.AddError("Error retrieving project", fmt.Sprintf("Error retrieving project with ID %s: %s", projectID, err.Error()))
				return
			}

			customerID = project.CustomerId
		}
	}

	return
}

// getProject retrieves a project by ID.
func (r *Resource) getProject(ctx context.Context, projectID string) (*projectv2.Project, error) {
	getReq := projectclientv2.GetProjectRequest{
		ProjectID: projectID,
	}

	project, _, err := r.client.Project().GetProject(ctx, getReq)
	if err != nil {
		return nil, err
	}

	return project, nil
}
