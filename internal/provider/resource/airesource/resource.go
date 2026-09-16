package airesource

import (
	"context"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/aihostingclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/contractclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/aihostingv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/contractv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiutils"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
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
	resp.TypeName = req.ProviderTypeName + "_ai"
}

func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "This resource models mittwald AI support for a specific mittwald customer.\n\n" +
			"**Note:** AI support is an add-on feature and will incur additional costs.\n\n" +
			"**Note:** A customer may have several AI hosting plans; each `mittwald_ai` resource manages exactly one of " +
			"them. Unlike earlier provider versions, this resource will no longer automatically adopt a pre-existing, " +
			"unmanaged AI hosting plan into state; use `terraform import` for that instead.",

		Attributes: map[string]schema.Attribute{
			"customer_id": schema.StringAttribute{
				MarkdownDescription: "ID of the customer for which AI support should be enabled",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"contract_id": schema.StringAttribute{
				MarkdownDescription: "The contract ID associated with the AI support",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"article_id": schema.StringAttribute{
				MarkdownDescription: "The article ID associated with the AI support. This may be used to change the pricing plan to a higher tier at any time. When changing to a lower tier, the change will only become active after the contract duration (this may result in undefined behavior in the Terraform plan).",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "A display name for this AI hosting plan. Useful to tell apart several AI hosting plans booked for the same customer. If not set, a default name will be assigned by the API.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"use_free_trial": schema.BoolAttribute{
				MarkdownDescription: "Use a free trial period for AI support, when available. Only applicable on creation, not on updates.",
				WriteOnly:           true, // This is irretrievable on the API side, so we're treating it as write-only
				Optional:            true,
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

	// use_free_trial is write-only, so its value is only available from the
	// config (it is always null in the plan and state).
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("use_free_trial"), &data.UseFreeTrial)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.createNewContract(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// use_free_trial is write-only and must not be persisted to state.
	data.UseFreeTrial = types.BoolNull()

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// listAIHostingPlanIDs returns the IDs of all AI hosting plans currently
// booked for the given customer.
func (r *Resource) listAIHostingPlanIDs(ctx context.Context, customerID string) ([]string, error) {
	plans, err := apiutils.FetchAllPages(ctx, 100, func(ctx context.Context, limit, page int64) (*[]aihostingv2.CustomerPlan, *http.Response, error) {
		plans, httpResp, err := r.client.AIHosting().CustomerGetPlans(ctx, aihostingclientv2.CustomerGetPlansRequest{
			CustomerID: customerID,
			Limit:      &limit,
			Page:       &page,
		})
		if plans == nil {
			return nil, httpResp, err
		}
		return &plans.Plans, httpResp, err
	})
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(plans))
	for _, plan := range plans {
		ids = append(ids, plan.PlanId)
	}

	return ids, nil
}

// resolvePlanID finds the AI hosting plan ID that belongs to the given
// contract, by scanning all AI hosting plans booked for the customer. It
// returns an empty string (without error) if no matching plan was found.
func (r *Resource) resolvePlanID(ctx context.Context, customerID, contractID string) (string, diag.Diagnostics) {
	var diags diag.Diagnostics

	planIDs, err := r.listAIHostingPlanIDs(ctx, customerID)
	if err != nil {
		diags.AddError("error while listing AI hosting plans", err.Error())
		return "", diags
	}

	for _, planID := range planIDs {
		contract := providerutil.
			Try[*contractv2.Contract](&diags, "error while checking AI hosting contract").
			IgnoreNotFound().
			DoValResp(r.client.Contract().GetDetailOfContractByAIHosting(ctx, contractclientv2.GetDetailOfContractByAIHostingRequest{
				CustomerID:  customerID,
				AIHostingID: planID,
			}))

		if diags.HasError() {
			return "", diags
		}

		if contract != nil && contract.ContractId == contractID {
			return planID, diags
		}
	}

	return "", diags
}

// applyPlanName reads the display name (description) of the given AI hosting
// plan and stores it in data.Name.
func (r *Resource) applyPlanName(ctx context.Context, data *ResourceModel, planID string) (res diag.Diagnostics) {
	plan := providerutil.
		Try[*aihostingv2.CustomerPlan](&res, "error while reading AI hosting plan").
		IgnoreNotFound().
		DoValResp(r.client.AIHosting().CustomerGetPlan(ctx, aihostingclientv2.CustomerGetPlanRequest{
			CustomerID: data.CustomerID.ValueString(),
			PlanID:     planID,
		}))

	if res.HasError() {
		return
	}

	if plan != nil {
		data.Name = types.StringValue(plan.Description)
	} else {
		data.Name = types.StringNull()
	}

	return
}

func (r *Resource) createNewContract(ctx context.Context, data *ResourceModel) (res diag.Diagnostics) {
	customerID := data.CustomerID.ValueString()

	existingPlanIDs, err := r.listAIHostingPlanIDs(ctx, customerID)
	if err != nil {
		res.AddError("error while listing existing AI hosting plans", err.Error())
		return
	}

	existing := make(map[string]bool, len(existingPlanIDs))
	for _, id := range existingPlanIDs {
		existing[id] = true
	}

	orderRequest := providerutil.
		Try[*contractclientv2.CreateOrderRequest](&res, "error while building AI hosting order").
		DoVal(data.ToAPICreateOrderRequest(ctx, r.client))

	if res.HasError() {
		return
	}

	providerutil.
		Try[*contractclientv2.CreateOrderResponse](&res, "error while creating AI hosting order").
		DoValResp(r.client.Contract().CreateOrder(ctx, *orderRequest))

	if res.HasError() {
		return
	}

	// The order API does not return the ID of the resulting AI hosting plan
	// directly, so we determine it by polling the customer's AI hosting
	// plans until a plan appears that wasn't there before.
	newPlanID, err := apiutils.Poll(ctx, apiutils.PollOpts{}, func(ctx context.Context, _ struct{}) (string, error) {
		planIDs, err := r.listAIHostingPlanIDs(ctx, customerID)
		if err != nil {
			return "", err
		}

		for _, id := range planIDs {
			if !existing[id] {
				return id, nil
			}
		}

		return "", apiutils.ErrPollShouldRetry
	}, struct{}{})

	if err != nil {
		res.AddError("error while waiting for the new AI hosting plan to become available", err.Error())
		return
	}

	contract := providerutil.
		Try[*contractv2.Contract](&res, "error while reading AI hosting contract").
		DoVal(apiutils.PollRequest(ctx, apiutils.PollOpts{}, r.client.Contract().GetDetailOfContractByAIHosting, contractclientv2.GetDetailOfContractByAIHostingRequest{
			CustomerID:  customerID,
			AIHostingID: newPlanID,
		}))

	if res.HasError() {
		return
	}

	res.Append(data.FromAPIModel(ctx, contract)...)
	if res.HasError() {
		return
	}

	res.Append(r.applyPlanName(ctx, data, newPlanID)...)

	return
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// The timeout is larger than a single request would need, since reading
	// a plan's name requires scanning all of the customer's AI hosting plans
	// (see resolvePlanID), which may take several sequential requests.
	readCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	resp.Diagnostics.Append(r.read(readCtx, &data, true)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If the contract no longer exists, remove the resource from state.
	if data.ContractID.IsNull() || data.ContractID.ValueString() == "" {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Resource) changePlan(ctx context.Context, data *ResourceModel) (res diag.Diagnostics) {
	changeReq := providerutil.
		Try[*contractclientv2.CreateTariffChangeRequest](&res, "error while creating API request").
		DoVal(data.ToAPIChangePlanRequest(ctx, r.client))

	if res.HasError() {
		return
	}

	providerutil.
		Try[*contractclientv2.CreateTariffChangeResponse](&res, "error while requesting AI plan change").
		DoValResp(r.client.Contract().CreateTariffChange(ctx, *changeReq))

	return
}

func (r *Resource) renamePlan(ctx context.Context, data *ResourceModel) (res diag.Diagnostics) {
	planID, diags := r.resolvePlanID(ctx, data.CustomerID.ValueString(), data.ContractID.ValueString())
	res.Append(diags...)
	if res.HasError() {
		return
	}

	if planID == "" {
		res.AddError("error while renaming AI hosting plan", "could not determine the AI hosting plan ID for contract "+data.ContractID.ValueString())
		return
	}

	description := data.Name.ValueString()
	providerutil.
		Try[any](&res, "error while renaming AI hosting plan").
		DoResp(r.client.AIHosting().CustomerUpdatePlan(ctx, aihostingclientv2.CustomerUpdatePlanRequest{
			CustomerID: data.CustomerID.ValueString(),
			PlanID:     planID,
			Body: aihostingclientv2.CustomerUpdatePlanRequestBody{
				Description: &description,
			},
		}))

	return
}

func (r *Resource) read(ctx context.Context, data *ResourceModel, considerTerminatedAsDeleted bool) (res diag.Diagnostics) {
	client := r.client.Contract()

	contract := providerutil.
		Try[*contractv2.Contract](&res, "error while reading AI hosting contract").
		IgnoreNotFound().
		DoVal(apiutils.PollRequest(ctx, apiutils.PollOpts{}, client.GetDetailOfContract, contractclientv2.GetDetailOfContractRequest{ContractID: data.ContractID.ValueString()}))

	// Consider contract as deleted if it has a termination date
	if contract != nil && contract.Termination != nil && considerTerminatedAsDeleted {
		contract = nil
	}

	if res.HasError() {
		return
	}

	res.Append(data.FromAPIModel(ctx, contract)...)
	if res.HasError() || contract == nil {
		return
	}

	planID, diags := r.resolvePlanID(ctx, data.CustomerID.ValueString(), contract.ContractId)
	res.Append(diags...)
	if res.HasError() {
		return
	}

	if planID != "" {
		res.Append(r.applyPlanName(ctx, data, planID)...)
	} else {
		data.Name = types.StringNull()
	}

	return
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var dataPlan, dataState ResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &dataPlan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &dataState)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !dataPlan.ArticleID.Equal(dataState.ArticleID) {
		resp.Diagnostics.Append(r.changePlan(ctx, &dataPlan)...)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	if !dataPlan.Name.Equal(dataState.Name) && !dataPlan.Name.IsUnknown() {
		resp.Diagnostics.Append(r.renamePlan(ctx, &dataPlan)...)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(r.read(ctx, &dataPlan, true)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &dataPlan)...)
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	terminateContractRequest := contractclientv2.TerminateContractRequest{ContractID: data.ContractID.ValueString()}

	providerutil.
		Try[any](&resp.Diagnostics, "error while terminating the AI hosting plan").
		IgnoreNotFound().
		DoValResp(r.client.Contract().TerminateContract(ctx, terminateContractRequest))
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	contractID := req.ID

	// Fetch the contract details to get the customer_id
	contract := providerutil.
		Try[*contractv2.Contract](&resp.Diagnostics, "error while fetching contract details for import").
		DoValResp(r.client.Contract().GetDetailOfContract(ctx, contractclientv2.GetDetailOfContractRequest{
			ContractID: contractID,
		}))

	if resp.Diagnostics.HasError() {
		return
	}

	// Create a model with the contract_id and customer_id
	var data ResourceModel
	resp.Diagnostics.Append(data.FromAPIModel(ctx, contract)...)
	data.CustomerID = types.StringValue(contract.CustomerId)

	if resp.Diagnostics.HasError() {
		return
	}

	planID, diags := r.resolvePlanID(ctx, data.CustomerID.ValueString(), data.ContractID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if planID != "" {
		resp.Diagnostics.Append(r.applyPlanName(ctx, &data, planID)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Set the state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
