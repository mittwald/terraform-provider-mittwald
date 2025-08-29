package containerstackresource

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/containerclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/containerv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiext"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
)

// Update reconciles the current state of the resource with the desired state.
//
// Implementation note: There is a difference in implementation between the
// default stack and additional stacks. The default stack is "updated", and the
// implementation respects that there may be containers that are not managed by
// this resource.
// The additional stacks are "declared", and the implementation assumes that all
// containers in the stack are managed by this resource.
func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData, stateData ContainerStackModel

	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "stack_id", stateData.ID.ValueString())
	client := apiext.NewContainerClient(r.client)

	var stack *containerv2.StackResponse

	if stateData.DefaultStack.ValueBool() {
		req := planData.ToUpdateRequest(ctx, &stateData, &resp.Diagnostics)
		if resp.Diagnostics.HasError() || req == nil {
			return
		}

		stack = providerutil.
			Try[*containerv2.StackResponse](&resp.Diagnostics, "API error while updating stack").
			DoValResp(client.UpdateStack(ctx, *req))
	} else {
		req := planData.ToDeclareRequest(ctx, &resp.Diagnostics)
		if resp.Diagnostics.HasError() || req == nil {
			return
		}

		stack = providerutil.
			Try[*containerv2.StackResponse](&resp.Diagnostics, "API error while declaring stack").
			DoValResp(client.DeclareStack(ctx, *req))
	}

	if resp.Diagnostics.HasError() {
		return
	}

	r.recreateContainers(ctx, planData, stack, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	providerutil.Try[any](&resp.Diagnostics, "API error while waiting for stack to be ready").
		Do(client.WaitUntilStackIsReady(ctx, stack.Id, planData.ContainerNames()))

	resp.Diagnostics.Append(r.read(ctx, &stateData)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &stateData)...)
}

// recreateContainers checks if any containers need to be recreated based on the
// current state and the plan data. If so, it sends a request to recreate them.
//
// Recreation is skipped for containers that are present in the current stack,
// but not managed by this resource, as well as for containers whose deployed
// state is equal to the pending state. Additionally, recreation is skipped if
// the no_recreate_on_change flag is set to true.
func (r *Resource) recreateContainers(ctx context.Context, planData ContainerStackModel, stack *containerv2.StackResponse, resp *resource.UpdateResponse) {
	containerModels := planData.ContainerModels(ctx, &resp.Diagnostics)

	for _, service := range stack.Services {
		ctx := tflog.SetField(ctx, "service_id", service.Id)

		serviceConfig, ok := containerModels[service.ServiceName]
		if !ok {
			continue
		}

		if !service.RequiresRecreate {
			tflog.Debug(ctx, "service does not require recreation; skipping")
			continue
		}

		if serviceConfig.NoRecreateOnChange.ValueBool() {
			tflog.Debug(ctx, "recreation would be necessary, but no_recreate_on_change is set; skipping recreation")
			continue
		}

		req := containerclientv2.RecreateServiceRequest{
			StackID:   stack.Id,
			ServiceID: service.Id,
		}

		tflog.Debug(ctx, "recreating service")

		providerutil.
			Try[any](&resp.Diagnostics, "API error while recreating container").
			DoResp(r.client.Container().RecreateService(ctx, req))
	}
}
