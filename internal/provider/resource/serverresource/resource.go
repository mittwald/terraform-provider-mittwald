package serverresource

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/contractclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/projectclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/contractv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/orderv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/projectv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiutils"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &Resource{}
var _ resource.ResourceWithImportState = &Resource{}

// provisioningTimeout bounds how long Create/Update wait for an ordered server
// to be provisioned and become ready.
const provisioningTimeout = 30 * time.Minute

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
	resp.Schema = schema.Schema{
		MarkdownDescription: "This resource models a (virtual) server for a specific mittwald customer.\n\n" +
			"**Note:** Ordering a server is a cost-intensive operation and will incur additional costs.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the server",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"contract_id": schema.StringAttribute{
				MarkdownDescription: "The contract ID associated with the server",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"customer_id": schema.StringAttribute{
				MarkdownDescription: "ID of the customer for which the server should be ordered",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"article_id": schema.StringAttribute{
				MarkdownDescription: "The article ID determining the machine type of the server. This may be used to change the machine type at any time. When changing to a lower tier, the change will only become active after the contract duration (this may result in undefined behavior in the Terraform plan).",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A human-readable description for the server",
				Required:            true,
			},
			"diskspace_gb": schema.Int64Attribute{
				MarkdownDescription: "The amount of disk space for the server, in GiB",
				Required:            true,
			},
			"use_free_trial": schema.BoolAttribute{
				MarkdownDescription: "Use a free trial period for the server, when available. Only applicable on creation, not on updates.",
				WriteOnly:           true, // This is irretrievable on the API side, so we're treating it as write-only
				Optional:            true,
			},
			"short_id": schema.StringAttribute{
				MarkdownDescription: "The short ID of the server (for example `s-4e7tz3`)",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"machine_type": schema.StringAttribute{
				MarkdownDescription: "The machine type of the server, as resolved from the article",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The current status of the server",
				Computed:            true,
			},
			"cluster_name": schema.StringAttribute{
				MarkdownDescription: "The name of the cluster the server is running on",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The time at which the server was created",
				Computed:            true,
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

	ctx, cancel := context.WithTimeout(ctx, provisioningTimeout)
	defer cancel()

	orderRequest := providerutil.
		Try[*contractclientv2.CreateOrderRequest](&resp.Diagnostics, "error while building server order").
		DoVal(data.ToAPICreateOrderRequest(ctx, r.client))

	if resp.Diagnostics.HasError() {
		return
	}

	orderResponse := providerutil.
		Try[*contractclientv2.CreateOrderResponse](&resp.Diagnostics, "error while creating server order").
		DoValResp(r.client.Contract().CreateOrder(ctx, *orderRequest))

	if resp.Diagnostics.HasError() {
		return
	}

	serverID, contractID := r.resolveServerFromOrder(ctx, orderResponse.OrderId, data.CustomerID.ValueString(), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(serverID)
	data.ContractID = types.StringValue(contractID)

	resp.Diagnostics.Append(r.waitUntilReady(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// resolveServerFromOrder waits for an order to be executed and resolves the
// resulting server ID and contract ID by matching the order's contract item
// against the customer's contracts.
func (r *Resource) resolveServerFromOrder(ctx context.Context, orderID, customerID string, diags *diag.Diagnostics) (serverID string, contractID string) {
	type resolved struct {
		serverID   string
		contractID string
	}

	result, err := apiutils.Poll(ctx, apiutils.PollOpts{}, func(ctx context.Context, _ struct{}) (resolved, error) {
		order, _, err := r.client.Contract().GetOrder(ctx, contractclientv2.GetOrderRequest{OrderID: orderID})
		if err != nil {
			return resolved{}, err
		}

		if order.Status != orderv2.OrderStatusEXECUTED {
			return resolved{}, apiutils.ErrPollShouldRetry
		}

		contracts, _, err := r.client.Contract().ListContracts(ctx, contractclientv2.ListContractsRequest{CustomerID: customerID})
		if err != nil {
			return resolved{}, err
		}
		if contracts == nil {
			return resolved{}, apiutils.ErrPollShouldRetry
		}

		for _, contract := range *contracts {
			item := contract.BaseItem
			matches := item.OrderId != nil && *item.OrderId == orderID
			if !matches {
				continue
			}

			if item.AggregateReference == nil || item.AggregateReference.Id == "" {
				return resolved{}, apiutils.ErrPollShouldRetry
			}

			return resolved{serverID: item.AggregateReference.Id, contractID: contract.ContractId}, nil
		}

		return resolved{}, apiutils.ErrPollShouldRetry
	}, struct{}{})

	if err != nil {
		diags.AddError("error while resolving server from order", err.Error())
		return "", ""
	}

	return result.serverID, result.contractID
}

// waitUntilReady polls the server until it reaches the ready state and maps the
// resulting API model into data.
func (r *Resource) waitUntilReady(ctx context.Context, data *ResourceModel) (res diag.Diagnostics) {
	server := providerutil.
		Try[*projectv2.Server](&res, "error while waiting for server to become ready").
		DoVal(apiutils.Poll(ctx, apiutils.PollOpts{}, func(ctx context.Context, serverID string) (*projectv2.Server, error) {
			s, _, err := r.client.Project().GetServer(ctx, projectclientv2.GetServerRequest{ServerID: serverID})
			if err != nil {
				return nil, err
			}

			if s.Status != projectv2.ServerStatusReady {
				return nil, apiutils.ErrPollShouldRetry
			}

			return s, nil
		}, data.ID.ValueString()))

	if res.HasError() {
		return
	}

	res.Append(data.FromAPIModel(ctx, server)...)
	return
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	server := providerutil.
		Try[*projectv2.Server](&resp.Diagnostics, "error while reading server").
		IgnoreNotFound().
		DoValResp(r.client.Project().GetServer(ctx, projectclientv2.GetServerRequest{ServerID: data.ID.ValueString()}))

	if resp.Diagnostics.HasError() {
		return
	}

	if server == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(data.FromAPIModel(ctx, server)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The contract ID is not part of the server representation; refresh it from
	// the API if it is missing (e.g. after an import).
	if data.ContractID.IsNull() || data.ContractID.ValueString() == "" {
		data.ContractID = types.StringValue(r.lookupContractID(ctx, data.ID.ValueString(), &resp.Diagnostics))
		if resp.Diagnostics.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var dataPlan, dataState ResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &dataPlan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &dataState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Carry over computed values from the prior state.
	dataPlan.ID = dataState.ID
	dataPlan.ContractID = dataState.ContractID

	if !dataPlan.Description.Equal(dataState.Description) {
		providerutil.
			Try[any](&resp.Diagnostics, "error while updating server description").
			DoResp(r.client.Project().UpdateServerDescription(ctx, projectclientv2.UpdateServerDescriptionRequest{
				ServerID: dataPlan.ID.ValueString(),
				Body:     projectclientv2.UpdateServerDescriptionRequestBody{Description: dataPlan.Description.ValueString()},
			}))
	}

	if !dataPlan.ArticleID.Equal(dataState.ArticleID) || !dataPlan.DiskspaceGB.Equal(dataState.DiskspaceGB) {
		resp.Diagnostics.Append(r.changePlan(ctx, &dataPlan)...)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	server := providerutil.
		Try[*projectv2.Server](&resp.Diagnostics, "error while reading server").
		DoValResp(r.client.Project().GetServer(ctx, projectclientv2.GetServerRequest{ServerID: dataPlan.ID.ValueString()}))

	if resp.Diagnostics.HasError() {
		return
	}

	// A disk resize is applied asynchronously via the tariff change, so the
	// server may still report the previous storage here. Keep the requested
	// value to avoid an "inconsistent result after apply" error; the actual
	// value is reconciled on the next read.
	plannedDiskspace := dataPlan.DiskspaceGB

	resp.Diagnostics.Append(dataPlan.FromAPIModel(ctx, server)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dataPlan.DiskspaceGB = plannedDiskspace

	resp.Diagnostics.Append(resp.State.Set(ctx, &dataPlan)...)
}

func (r *Resource) changePlan(ctx context.Context, data *ResourceModel) (res diag.Diagnostics) {
	changeReq := providerutil.
		Try[*contractclientv2.CreateTariffChangeRequest](&res, "error while building tariff change request").
		DoVal(data.ToAPIChangePlanRequest(ctx, r.client))

	if res.HasError() {
		return
	}

	providerutil.
		Try[*contractclientv2.CreateTariffChangeResponse](&res, "error while requesting server tariff change").
		DoValResp(r.client.Contract().CreateTariffChange(ctx, *changeReq))

	return
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	contractID := data.ContractID.ValueString()
	if contractID == "" {
		contractID = r.lookupContractID(ctx, data.ID.ValueString(), &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	providerutil.
		Try[*contractclientv2.TerminateContractResponse](&resp.Diagnostics, "error while terminating the server contract").
		IgnoreNotFound().
		DoValResp(r.client.Contract().TerminateContract(ctx, contractclientv2.TerminateContractRequest{ContractID: contractID}))
}

// lookupContractID resolves the contract ID for a given server.
func (r *Resource) lookupContractID(ctx context.Context, serverID string, diags *diag.Diagnostics) string {
	contract := providerutil.
		Try[*contractv2.Contract](diags, "error while looking up server contract").
		DoValResp(r.client.Contract().GetDetailOfContractByServer(ctx, contractclientv2.GetDetailOfContractByServerRequest{ServerID: serverID}))

	if diags.HasError() || contract == nil {
		return ""
	}

	return contract.ContractId
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	serverID := req.ID

	server := providerutil.
		Try[*projectv2.Server](&resp.Diagnostics, "error while reading server for import").
		DoValResp(r.client.Project().GetServer(ctx, projectclientv2.GetServerRequest{ServerID: serverID}))

	if resp.Diagnostics.HasError() {
		return
	}

	contract := providerutil.
		Try[*contractv2.Contract](&resp.Diagnostics, "error while reading server contract for import").
		DoValResp(r.client.Contract().GetDetailOfContractByServer(ctx, contractclientv2.GetDetailOfContractByServerRequest{ServerID: serverID}))

	if resp.Diagnostics.HasError() {
		return
	}

	var data ResourceModel
	resp.Diagnostics.Append(data.FromAPIModel(ctx, server)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ContractID = types.StringValue(contract.ContractId)
	if len(contract.BaseItem.Articles) > 0 {
		data.ArticleID = types.StringValue(contract.BaseItem.Articles[0].Id)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
