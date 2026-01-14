package airesource

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/articleclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/contractclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/contractv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/orderv2"
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
			"**Note:** AI support is an add-on feature and will incur additional costs.",

		Attributes: map[string]schema.Attribute{
			"customer_id": schema.StringAttribute{
				MarkdownDescription: "ID of the customer for which AI support should be enabled",
				Optional:            true,
			},
			"order_id": schema.StringAttribute{
				MarkdownDescription: "The order ID associated with the AI support",
				Computed:            true,
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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
				MarkdownDescription: "The article ID associated with the AI support",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"use_free_trial": schema.BoolAttribute{
				MarkdownDescription: "Use a free trial period for AI support, when available",
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

	monthlyTokens, requestsPerMinute, err := r.getArticleFeatures(ctx, data.ArticleId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error retrieving article features", fmt.Sprintf("Could not retrieve article features for article ID %s: %s", data.ArticleId.ValueString(), err.Error()))
		return
	}

	contractRequest := contractclientv2.GetDetailOfContractByAIHostingRequest{CustomerID: data.CustomerId.ValueString()}
	contractResponse := providerutil.
		Try[*contractv2.Contract](&resp.Diagnostics, "error while checking for existing AI hosting contract").
		IgnoreNotFound().
		DoVal(apiutils.PollRequest(ctx, apiutils.PollOpts{}, r.client.Contract().GetDetailOfContractByAIHosting, contractRequest))

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

		if contractResponse.BaseItem.Articles[0].Id != data.ArticleId.ValueString() {
			resp.Diagnostics.AddError("Unsupported", "Changing hosting plans for AI hosting is currently not supported")
			return
		}

		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

		return
	}

	orderType := contractclientv2.CreateOrderRequestBodyOrderTypeAIHosting
	orderRequest := contractclientv2.CreateOrderRequest{
		Body: contractclientv2.CreateOrderRequestBody{
			OrderType: &orderType,
			OrderData: &contractclientv2.CreateOrderRequestBodyOrderData{
				AlternativeAIHostingOrder: &orderv2.AIHostingOrder{
					CustomerId:        data.CustomerId.ValueString(),
					UseFreeTrial:      data.UseFreeTrial.ValueBoolPointer(),
					MonthlyTokens:     monthlyTokens,
					RequestsPerMinute: requestsPerMinute,
				},
			},
		},
	}

	orderResponse := providerutil.
		Try[*contractclientv2.CreateOrderResponse](&resp.Diagnostics, "error while creating AI hosting order").
		DoValResp(r.client.Contract().CreateOrder(
			ctx,
			orderRequest,
		))

	if resp.Diagnostics.HasError() {
		return
	}

	data.OrderID = types.StringValue(orderResponse.OrderId)

	resp.Diagnostics.Append(r.read(ctx, &data, true)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Resource) getArticleFeatures(ctx context.Context, articleID string) (int64, int64, error) {
	monthlyTokens := int64(0)
	requestsPerMinute := int64(0)

	articleRequest := articleclientv2.GetArticleRequest{ArticleID: articleID}
	article, _, err := r.client.Article().GetArticle(ctx, articleRequest)
	if err != nil {
		return 0, 0, fmt.Errorf("error while retrieving article %s: %w", articleID, err)
	}

	for _, attr := range article.Attributes {
		if attr.Key == "monthlyTokens" {
			monthlyTokens, err = strconv.ParseInt(*attr.Value, 10, 64)
			if err != nil {
				return 0, 0, fmt.Errorf("error while parsing monthlyTokens: %w", err)
			}
		}

		if attr.Key == "requestsPerMinute" {
			requestsPerMinute, err = strconv.ParseInt(*attr.Value, 10, 64)
			if err != nil {
				return 0, 0, fmt.Errorf("error while parsing requestsPerMinute: %w", err)
			}
		}
	}

	return monthlyTokens, requestsPerMinute, nil
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

func (r *Resource) read(ctx context.Context, data *ResourceModel, considerTerminatedAsDeleted bool) (res diag.Diagnostics) {
	client := r.client.Contract()

	contract := providerutil.
		Try[*contractv2.Contract](&res, "error while reading AI hosting usage").
		IgnoreNotFound().
		DoVal(apiutils.PollRequest(ctx, apiutils.PollOpts{}, client.GetDetailOfContractByAIHosting, contractclientv2.GetDetailOfContractByAIHostingRequest{CustomerID: data.CustomerId.ValueString()}))

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

	if !dataPlan.ArticleId.Equal(dataState.ArticleId) {
		monthlyTokens, requestsPerMinute, err := r.getArticleFeatures(ctx, dataPlan.ArticleId.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error retrieving article features", fmt.Sprintf("Could not retrieve article features for article ID %s: %s", dataPlan.ArticleId.ValueString(), err.Error()))
			return
		}

		changeType := contractclientv2.CreateTariffChangeRequestBodyTariffChangeTypeAIHosting
		changeReq := contractclientv2.CreateTariffChangeRequest{
			Body: contractclientv2.CreateTariffChangeRequestBody{
				TariffChangeType: &changeType,
				TariffChangeData: &contractclientv2.CreateTariffChangeRequestBodyTariffChangeData{
					AlternativeAIHostingTariffChange: &orderv2.AIHostingTariffChange{
						ContractId:        dataPlan.ContractID.ValueString(),
						MonthlyTokens:     monthlyTokens,
						RequestsPerMinute: requestsPerMinute,
					},
				},
			},
		}

		planChange := providerutil.
			Try[*contractclientv2.CreateTariffChangeResponse](&resp.Diagnostics, "error while requesting AI plan increase").
			IgnoreNotFound().
			DoValResp(r.client.Contract().CreateTariffChange(ctx, changeReq))

		dataPlan.OrderID = types.StringValue(planChange.OrderId)

		resp.Diagnostics.AddError("Unsupported", "Changing hosting plans for AI hosting is currently not supported")
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

	terminateContractRequest := contractclientv2.TerminateContractRequest{ContractID: data.ContractID.ValueString()}

	providerutil.
		Try[any](&resp.Diagnostics, "error while terminating the AI hosting plan").
		IgnoreNotFound().
		DoValResp(r.client.Contract().TerminateContract(ctx, terminateContractRequest))
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
