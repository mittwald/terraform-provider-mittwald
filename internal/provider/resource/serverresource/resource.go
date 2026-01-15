package serverresource

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/contractclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/contractv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/orderv2"
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
	resp.TypeName = req.ProviderTypeName + "_server"
}

func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	builder := common.AttributeBuilderFor("server")
	resp.Schema = schema.Schema{
		MarkdownDescription: "This resource models a server on the mittwald cloud platform; a server is always provisioned for a customer, and usually incurs costs to the user.",

		Attributes: map[string]schema.Attribute{
			"customer_id": schema.StringAttribute{
				MarkdownDescription: "ID of the customer this server belongs to",
				Optional:            true,
			},
			"id": builder.Id(),
			"short_id": schema.StringAttribute{
				MarkdownDescription: "The short ID of the server",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": builder.Description(),
			"machine_type": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "The machine type name",
						Required:            true,
					},
					"cpu": schema.Float64Attribute{
						MarkdownDescription: "The number of CPUs; this is derived from the machine type",
						Computed:            true,
					},
					"ram": schema.Float64Attribute{
						MarkdownDescription: "The amount of RAM in GB; this is derived from the machine type",
						Computed:            true,
					},
				},
			},
			"volume_size": schema.Int64Attribute{
				MarkdownDescription: "The size of the server's volume in GB",
				Required:            true,
			},
			"use_free_trial": schema.BoolAttribute{
				MarkdownDescription: "Whether to use a free trial for this server, when available",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				WriteOnly:           true,
			},
		},
	}
}

func (r *Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.ClientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ResourceModel

	client := r.client.Contract()

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(data.Validate()...)

	if resp.Diagnostics.HasError() {
		return
	}

	orderResponse := providerutil.
		Try[*contractclientv2.CreateOrderResponse](&resp.Diagnostics, "error while creating server order").
		DoValResp(client.CreateOrder(
			ctx,
			data.ToCreateOrderRequest(),
		))

	if resp.Diagnostics.HasError() {
		return
	}

	serverID, extractDiags := waitForOrderAndExtractServerID(ctx, client, orderResponse.OrderId)
	resp.Diagnostics.Append(extractDiags...)

	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(serverID)

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
	client := r.client.Contract()

	contract := providerutil.
		Try[*contractv2.Contract](&res, "error while reading server contract").
		IgnoreNotFound().
		DoValResp(client.GetDetailOfContractByServer(ctx, contractclientv2.GetDetailOfContractByServerRequest{ServerID: data.ID.ValueString()}))

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

	dataPlan.ID = dataState.ID

	resp.Diagnostics.Append(r.read(ctx, &dataPlan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &dataPlan)...)
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.Contract()

	contract := providerutil.
		Try[*contractv2.Contract](&resp.Diagnostics, "error while getting server contract for deletion").
		IgnoreNotFound().
		DoValResp(client.GetDetailOfContractByServer(ctx, contractclientv2.GetDetailOfContractByServerRequest{ServerID: data.ID.ValueString()}))

	if resp.Diagnostics.HasError() {
		return
	}

	if contract == nil {
		return
	}

	terminateReq := contractclientv2.TerminateContractRequest{
		ContractID: contract.ContractId,
		Body:       contractclientv2.TerminateContractRequestBody{},
	}

	providerutil.
		Try[*contractclientv2.TerminateContractResponse](&resp.Diagnostics, "error while terminating server contract").
		IgnoreNotFound().
		DoValResp(client.TerminateContract(ctx, terminateReq))
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func waitForOrderAndExtractServerID(ctx context.Context, client contractclientv2.Client, orderID string) (string, diag.Diagnostics) {
	var diags diag.Diagnostics

	order := providerutil.
		Try[*orderv2.CustomerOrder](&diags, "error while waiting for server order to complete").
		DoVal(apiutils.PollRequest(
			ctx,
			apiutils.PollOpts{},
			client.GetOrder,
			contractclientv2.GetOrderRequest{OrderID: orderID},
		))

	if diags.HasError() {
		return "", diags
	}

	if order.Status != orderv2.OrderStatusEXECUTED {
		diags.AddError("Order not completed", fmt.Sprintf("Order status is %s, expected EXECUTED", order.Status))
		return "", diags
	}

	if len(order.Items) == 0 {
		diags.AddError("Invalid order", "Order has no items")
		return "", diags
	}

	baseItem := order.Items[0]
	if baseItem.Reference == nil || baseItem.Reference.ContractItemId == nil {
		diags.AddError("Order not ready", "Order item does not have a contract item reference")
		return "", diags
	}

	contractItemID := *baseItem.Reference.ContractItemId

	contractItem := providerutil.
		Try[*contractv2.ContractItem](&diags, "error while getting server contract item").
		DoValResp(client.GetDetailOfContractItem(ctx, contractclientv2.GetDetailOfContractItemRequest{ContractItemID: contractItemID}))

	if diags.HasError() {
		return "", diags
	}

	if contractItem.AggregateReference == nil {
		diags.AddError("Invalid contract item", "Contract item does not have an aggregate reference")
		return "", diags
	}

	return contractItem.AggregateReference.Id, diags
}
