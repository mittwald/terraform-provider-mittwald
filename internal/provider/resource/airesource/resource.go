package airesource

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/contractclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/contractv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiutils"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &Resource{}
var _ resource.ResourceWithImportState = &Resource{}
var _ resource.ResourceWithModifyPlan = &Resource{}

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
			"**Note:** AI support is an add-on feature and will incur additional costs.",

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

func (r *Resource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var data ResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	contractRequest := contractclientv2.GetDetailOfContractByAIHostingRequest{CustomerID: data.CustomerID.ValueString()}
	contractResponse := providerutil.
		Try[*contractv2.Contract](&resp.Diagnostics, "error while checking for existing AI hosting contract").
		IgnoreNotFound().
		DoValResp(r.client.Contract().GetDetailOfContractByAIHosting(ctx, contractRequest))

	if resp.Diagnostics.HasError() {
		return
	}

	// TODO: This behavior might need changing in the future; currently, there
	//  is a fixed limit of one AI hosting contract per customer, but this may
	//  change in the future.
	//  If that should ever happen, we should just look up the contract by the
	//  contract ID in the state, and not attempt to "adopt" any unknown
	//  contracts.
	if contractResponse != nil && data.ContractID.IsUnknown() {
		resp.Diagnostics.AddAttributeWarning(path.Root("contract_id"), "Existing AI hosting contract detected", "An existing AI hosting contract was detected for this customer. The existing contract will be adopted into management by this resource. Note that certain changes (e.g., changing to a lower-tier plan) may not be possible until the end of the current contract duration.")

		resp.Diagnostics.Append(data.FromAPIModel(ctx, contractResponse)...)
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &data)...)

		return
	}
}

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	contractRequest := contractclientv2.GetDetailOfContractByAIHostingRequest{CustomerID: data.CustomerID.ValueString()}
	contractResponse := providerutil.
		Try[*contractv2.Contract](&resp.Diagnostics, "error while checking for existing AI hosting contract").
		IgnoreNotFound().
		DoValResp(r.client.Contract().GetDetailOfContractByAIHosting(ctx, contractRequest))

	if resp.Diagnostics.HasError() {
		return
	}

	if contractResponse != nil {
		resp.Diagnostics.Append(data.FromAPIModel(ctx, contractResponse)...)

		if contractResponse.Termination != nil {
			cancelTerminationRequest := contractclientv2.CancelContractTerminationRequest{ContractID: contractResponse.ContractId}
			providerutil.
				Try[any](&resp.Diagnostics, "error while cancelling AI hosting contract termination").
				DoValResp(r.client.Contract().CancelContractTermination(ctx, cancelTerminationRequest))

			if resp.Diagnostics.HasError() {
				return
			}
		}

		if contractResponse.BaseItem.Articles[0].Id != data.ArticleID.ValueString() {
			resp.Diagnostics.Append(r.changePlan(ctx, &data)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}

		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

		return
	}

	orderRequest := providerutil.
		Try[*contractclientv2.CreateOrderRequest](&resp.Diagnostics, "error while building AI hosting order").
		DoVal(data.ToAPICreateOrderRequest(ctx, r.client))

	providerutil.
		Try[*contractclientv2.CreateOrderResponse](&resp.Diagnostics, "error while creating AI hosting order").
		DoValResp(r.client.Contract().CreateOrder(ctx, *orderRequest))

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.read(ctx, &data, true)...)
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

	resp.Diagnostics.Append(r.read(readCtx, &data, true)...)
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

func (r *Resource) read(ctx context.Context, data *ResourceModel, considerTerminatedAsDeleted bool) (res diag.Diagnostics) {
	client := r.client.Contract()

	contract := providerutil.
		Try[*contractv2.Contract](&res, "error while reading AI hosting usage").
		IgnoreNotFound().
		DoVal(apiutils.PollRequest(ctx, apiutils.PollOpts{}, client.GetDetailOfContractByAIHosting, contractclientv2.GetDetailOfContractByAIHostingRequest{CustomerID: data.CustomerID.ValueString()}))

	// Consider contract as deleted if it has a termination date
	if contract.Termination != nil && considerTerminatedAsDeleted {
		contract = nil
	}

	if res.HasError() {
		return
	}

	res.Append(data.FromAPIModel(ctx, contract)...)

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
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
