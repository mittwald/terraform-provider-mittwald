package projectresource

import (
	"context"
	"errors"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/contractclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/projectclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/contractv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/orderv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/projectv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiext"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiutils"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/resource/common"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &Resource{}
var _ resource.ResourceWithImportState = &Resource{}
var _ resource.ResourceWithConfigValidators = &Resource{}

// provisioningTimeout bounds how long Create waits for an ordered stand-alone
// project to be provisioned and become ready.
const provisioningTimeout = 30 * time.Minute

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

func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	builder := common.AttributeBuilderFor("project")
	resp.Schema = schema.Schema{
		MarkdownDescription: "This resource models a project on the mittwald cloud platform.\n\n" +
			"A project is either provisioned on an existing server, in which case a `server_id` is required, " +
			"or ordered as a stand-alone project, in which case a `customer_id` and an `article_id` are required.\n\n" +
			"**Note:** Ordering a stand-alone project is a cost-intensive operation and will incur additional costs. " +
			"Projects placed on a server are billed as part of that server's contract.",

		Attributes: map[string]schema.Attribute{
			"server_id": schema.StringAttribute{
				MarkdownDescription: "ID of the server this project should be provisioned on. Must be a full UUID (not a short ID like s-XXXXXX). " +
					"Conflicts with `customer_id` and `article_id`.",
				Optional: true,
				Validators: []validator.String{
					&common.UUIDValidator{},
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"customer_id": schema.StringAttribute{
				MarkdownDescription: "ID of the customer for which the stand-alone project should be ordered. " +
					"Required together with `article_id`, and conflicts with `server_id`. " +
					"For a project on a server, this is populated from the server's customer.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"article_id": schema.StringAttribute{
				MarkdownDescription: "The article ID determining the machine type of a stand-alone project. " +
					"Required together with `customer_id`, and conflicts with `server_id`. " +
					"This may be used to change the machine type at any time. When changing to a lower tier, the change will " +
					"only become active after the contract duration (this may result in undefined behavior in the Terraform plan).",
				Optional: true,
			},
			"contract_id": schema.StringAttribute{
				MarkdownDescription: "The contract ID associated with a stand-alone project. Null for projects on a server, which are billed via the server's contract.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"diskspace_gb": schema.Int64Attribute{
				MarkdownDescription: "The amount of disk space for a stand-alone project, in GiB. Must be at least 20 and a multiple of 20. " +
					"Required together with `article_id`, and can only be set for stand-alone projects; for a project on a server, this " +
					"reports the disk space the project is allotted.",
				Optional: true,
				Computed: true,
				Validators: []validator.Int64{
					diskspaceValidator{},
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"use_free_trial": schema.BoolAttribute{
				MarkdownDescription: "Use a free trial period for the stand-alone project, when available. Only applicable on creation, not on updates.",
				WriteOnly:           true, // This is irretrievable on the API side, so we're treating it as write-only
				Optional:            true,
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
	}
}

func (r *Resource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		projectPlacementValidator{},
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

	if data.IsStandalone() {
		resp.Diagnostics.Append(r.createStandalone(ctx, &data)...)
	} else {
		resp.Diagnostics.Append(r.createOnServer(ctx, &data)...)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.readAfterCreate(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// use_free_trial is write-only and must not be persisted to state.
	data.UseFreeTrial = types.BoolNull()

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// createOnServer creates a project on an existing server, which is billed via
// that server's contract and therefore does not require an order.
func (r *Resource) createOnServer(ctx context.Context, data *ResourceModel) (res diag.Diagnostics) {
	projectResponse := providerutil.
		Try[*projectclientv2.CreateProjectResponse](&res, "error while creating project").
		DoValResp(r.client.Project().CreateProject(ctx, data.ToCreateRequest()))

	if res.HasError() {
		return
	}

	data.ID = types.StringValue(projectResponse.Id)
	data.ContractID = types.StringNull()

	return
}

// createStandalone orders a stand-alone project and waits until it has been
// provisioned and is ready to use.
func (r *Resource) createStandalone(ctx context.Context, data *ResourceModel) (res diag.Diagnostics) {
	ctx, cancel := context.WithTimeout(ctx, provisioningTimeout)
	defer cancel()

	orderRequest := providerutil.
		Try[*contractclientv2.CreateOrderRequest](&res, "error while building project order").
		DoVal(data.ToAPICreateOrderRequest(ctx, r.client))

	if res.HasError() {
		return
	}

	orderResponse := providerutil.
		Try[*contractclientv2.CreateOrderResponse](&res, "error while creating project order").
		DoValResp(r.client.Contract().CreateOrder(ctx, *orderRequest))

	if res.HasError() {
		return
	}

	projectID, contractID := r.resolveProjectFromOrder(ctx, orderResponse.OrderId, data.CustomerID.ValueString(), &res)
	if res.HasError() {
		return
	}

	data.ID = types.StringValue(projectID)
	data.ContractID = types.StringValue(contractID)

	res.Append(r.waitUntilReady(ctx, projectID)...)

	return
}

// resolveProjectFromOrder waits for an order to be executed and resolves the
// resulting project ID and contract ID by matching the order's contract item
// against the customer's contracts.
func (r *Resource) resolveProjectFromOrder(ctx context.Context, orderID, customerID string, diags *diag.Diagnostics) (projectID string, contractID string) {
	type resolved struct {
		projectID  string
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

			return resolved{projectID: item.AggregateReference.Id, contractID: contract.ContractId}, nil
		}

		return resolved{}, apiutils.ErrPollShouldRetry
	}, struct{}{})

	if err != nil {
		diags.AddError("error while resolving project from order", err.Error())
		return "", ""
	}

	return result.projectID, result.contractID
}

// waitUntilReady polls a newly ordered project until it has finished
// provisioning.
func (r *Resource) waitUntilReady(ctx context.Context, projectID string) (res diag.Diagnostics) {
	providerutil.
		Try[*projectv2.Project](&res, "error while waiting for project to become ready").
		DoVal(apiutils.Poll(ctx, apiutils.PollOpts{}, func(ctx context.Context, projectID string) (*projectv2.Project, error) {
			p, _, err := r.client.Project().GetProject(ctx, projectclientv2.GetProjectRequest{ProjectID: projectID})
			if err != nil {
				return nil, err
			}

			if !p.IsReady {
				return nil, apiutils.ErrPollShouldRetry
			}

			return p, nil
		}, projectID))

	return
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
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}

	// The contract lookup gets the outer context, since it is not part of the
	// polling above and should not be bound by its (short) deadline.
	resp.Diagnostics.Append(r.readContractAttributes(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Resource) read(ctx context.Context, data *ResourceModel) (res diag.Diagnostics) {
	client := apiext.NewProjectClient(r.client)

	pr := providerutil.
		Try[*projectv2.Project](&res, "error while reading project").
		IgnoreNotFound().
		DoVal(apiutils.PollRequest(ctx, apiutils.PollOpts{}, client.GetProject, projectclientv2.GetProjectRequest{ProjectID: data.ID.ValueString()}))

	ips := providerutil.
		Try[[]string](&res, "error while reading project ips").
		IgnoreNotFound().
		DoVal(client.GetProjectDefaultIPs(ctx, data.ID.ValueString()))

	if res.HasError() {
		return
	}

	res.Append(data.FromAPIModel(ctx, pr, ips)...)

	return
}

// readAfterCreate is like read but polls for the default IPs to become available.
// This is necessary because immediately after project creation, the default ingress
// may not exist yet.
func (r *Resource) readAfterCreate(ctx context.Context, data *ResourceModel) (res diag.Diagnostics) {
	client := apiext.NewProjectClient(r.client)

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	pr := providerutil.
		Try[*projectv2.Project](&res, "error while reading project").
		IgnoreNotFound().
		DoVal(apiutils.PollRequest(ctx, apiutils.PollOpts{}, client.GetProject, projectclientv2.GetProjectRequest{ProjectID: data.ID.ValueString()}))

	// Wrap GetProjectDefaultIPs to convert ErrNoDefaultIngress to ErrPollShouldRetry
	// so that the Poll function will retry until the default ingress appears.
	getIPsWithRetry := func(ctx context.Context, projectID string) ([]string, error) {
		ips, err := client.GetProjectDefaultIPs(ctx, projectID)
		if errors.Is(err, apiext.ErrNoDefaultIngress) {
			return nil, apiutils.ErrPollShouldRetry
		}
		return ips, err
	}

	ips := providerutil.
		Try[[]string](&res, "error while reading project ips").
		IgnoreNotFound().
		DoVal(apiutils.Poll(ctx, apiutils.PollOpts{}, getIPsWithRetry, data.ID.ValueString()))

	if res.HasError() {
		return
	}

	res.Append(data.FromAPIModel(ctx, pr, ips)...)

	return
}

// readContractAttributes fills in the attributes that are backed by the
// project's contract rather than by the project itself. A project on a server
// has no contract of its own, and for a stand-alone project the values never
// change, so they are only looked up while still missing (after an import).
func (r *Resource) readContractAttributes(ctx context.Context, data *ResourceModel) (res diag.Diagnostics) {
	if !data.IsStandalone() {
		data.ContractID = types.StringNull()
		data.ArticleID = types.StringNull()
		return
	}

	if !data.ContractID.IsNull() && !data.ArticleID.IsNull() {
		return
	}

	contract := r.lookupContract(ctx, data.ID.ValueString(), &res)
	if res.HasError() || contract == nil {
		return
	}

	data.ContractID = types.StringValue(contract.ContractId)

	// The contract keeps reporting the previous article while a tariff change is
	// still pending, so an already-known article ID is never overwritten here.
	if data.ArticleID.IsNull() && len(contract.BaseItem.Articles) > 0 {
		data.ArticleID = types.StringValue(contract.BaseItem.Articles[0].Id)
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

	if !dataPlan.ArticleID.Equal(dataState.ArticleID) || !dataPlan.DiskspaceGB.Equal(dataState.DiskspaceGB) {
		resp.Diagnostics.Append(r.changePlan(ctx, &dataPlan)...)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// A tariff change is applied asynchronously, so the project would still
	// report the previous machine type and disk space here. The planned values
	// are kept to avoid an "inconsistent result after apply" error; the actual
	// values are reconciled on the next read.
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
		Try[*contractclientv2.CreateTariffChangeResponse](&res, "error while requesting project tariff change").
		DoValResp(r.client.Contract().CreateTariffChange(ctx, *changeReq))

	return
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// A stand-alone project exists for as long as its contract does, so it is
	// removed by terminating that contract. A project on a server has no
	// contract of its own and can be deleted directly.
	if data.IsStandalone() {
		resp.Diagnostics.Append(r.terminateContract(ctx, &data)...)
		return
	}

	deleteReq := projectclientv2.DeleteProjectRequest{ProjectID: data.ID.ValueString()}

	providerutil.
		Try[any](&resp.Diagnostics, "error while deleting project").
		IgnoreNotFound().
		DoResp(r.client.Project().DeleteProject(ctx, deleteReq))
}

func (r *Resource) terminateContract(ctx context.Context, data *ResourceModel) (res diag.Diagnostics) {
	contractID := data.ContractID.ValueString()
	if contractID == "" {
		contract := r.lookupContract(ctx, data.ID.ValueString(), &res)
		if res.HasError() {
			return
		}

		if contract == nil {
			// Without a contract there is nothing left to terminate; the
			// project is already gone.
			return
		}

		contractID = contract.ContractId
	}

	providerutil.
		Try[*contractclientv2.TerminateContractResponse](&res, "error while terminating the project contract").
		IgnoreNotFound().
		DoValResp(r.client.Contract().TerminateContract(ctx, contractclientv2.TerminateContractRequest{ContractID: contractID}))

	return
}

// lookupContract resolves the contract of a stand-alone project.
func (r *Resource) lookupContract(ctx context.Context, projectID string, diags *diag.Diagnostics) *contractv2.Contract {
	return providerutil.
		Try[*contractv2.Contract](diags, "error while looking up project contract").
		IgnoreNotFound().
		DoValResp(r.client.Contract().GetDetailOfContractByProject(ctx, contractclientv2.GetDetailOfContractByProjectRequest{ProjectID: projectID}))
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The contract-derived attributes (contract_id, article_id) start out null
	// and are resolved by the subsequent read.
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
