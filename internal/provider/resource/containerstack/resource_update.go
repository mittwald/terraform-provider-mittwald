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
// (legacy) default stack and other stacks. The default stack is "updated", and the
// implementation respects that there may be containers that are not managed by
// this resource.
// The additional stacks are "declared", and the implementation assumes that all
// containers in the stack are managed by this resource.
//
// Which of the two is used depends on the planned value of `default_stack`.
// When it is removed from a resource that manages the default stack, the
// default stack is declared from then on; this way, the resource takes over the
// default stack exclusively, removing all containers and volumes that it does
// not manage.
func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData, stateData ContainerStackModel

	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, diags := planData.Timeouts.Update(ctx, DefaultUpdateTimeout)
	resp.Diagnostics.Append(diags...)

	readTimeout, diags := planData.Timeouts.Read(ctx, DefaultReadTimeout)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// The timeouts block is part of the configuration, not of the remote object;
	// carry the planned value over, or Terraform will complain about a changed
	// value once the state below is written from stateData.
	stateData.Timeouts = planData.Timeouts

	// The same applies to default_stack, which is a client-side attribute that
	// controls how the stack is managed. Switching it to true forces a
	// replacement, so the only in-place change is a take-over of the default
	// stack (true to false or null).
	stateData.DefaultStack = planData.DefaultStack
	defaultStack := planData.DefaultStack.ValueBool()

	ctx = tflog.SetField(ctx, "stack_id", stateData.ID.ValueString())
	client := apiext.NewContainerClient(r.client)

	updateCtx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	var stack *containerv2.StackResponse

	if defaultStack {
		req := planData.ToUpdateRequest(updateCtx, &stateData, &resp.Diagnostics)
		if resp.Diagnostics.HasError() || req == nil {
			return
		}

		// The default stack is already updated via UpdateStack, so its settings
		// can be folded into the same call instead of issuing a second one.
		opts, _ := planData.applyStackSettings(updateCtx, &stateData, &req.Body, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}

		stack = providerutil.
			Try[*containerv2.StackResponse](&resp.Diagnostics, "API error while updating stack").
			DoValResp(client.UpdateStack(updateCtx, *req, opts...))
	} else {
		req := planData.ToDeclareRequest(updateCtx, &resp.Diagnostics)
		if resp.Diagnostics.HasError() || req == nil {
			return
		}

		stack = providerutil.
			Try[*containerv2.StackResponse](&resp.Diagnostics, "API error while declaring stack").
			DoValResp(client.DeclareStack(updateCtx, *req))
	}

	if resp.Diagnostics.HasError() {
		return
	}

	r.recreateContainers(updateCtx, planData, stack, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	waitUntilStackIsReady(updateCtx, client, stack.Id, planData.ContainerNames(), updateTimeoutHint, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// For the default stack, the settings were already folded into the
	// UpdateStack call above. Declared stacks go through DeclareStack instead,
	// which has no fields for these settings, so they still need their own
	// UpdateStack call here.
	if !defaultStack {
		settingsReq := containerclientv2.UpdateStackRequest{StackID: stateData.ID.ValueString()}

		opts, changed := planData.applyStackSettings(updateCtx, &stateData, &settingsReq.Body, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}

		if changed {
			providerutil.
				Try[*containerv2.StackResponse](&resp.Diagnostics, "API error while updating stack settings").
				DoValResp(client.UpdateStack(updateCtx, settingsReq, opts...))
		}
	}

	// Like on create, the read-back gets its own budget, so that an exhausted
	// update timeout does not also fail the read and leave a stale state behind.
	readCtx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	notFound, diags := r.read(readCtx, &stateData, &planData)
	resp.Diagnostics.Append(diags...)
	if notFound {
		resp.Diagnostics.AddError("API error while fetching stack", "the stack could not be found after it was written")
	}
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
