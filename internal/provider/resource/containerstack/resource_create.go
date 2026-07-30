package containerstackresource

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/containerv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiext"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
)

// Create creates a new container stack.
//
// Implementation note: There are two ways of "creating" a stack; which one is
// used depends on whether the `default_stack` attribute is set to true or not.
//
// In the former case, the actual stack in the API will already exist, and we
// need to "update" it with the new containers. In this case, we also need to
// respect the fact that there may be containers or volumes in the default stack
// that are not part of the current plan. These should not be touched at all.
//
// In the latter case, we create a new stack in the API (and assume that we have
// exclusive ownership of it).
func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ContainerStackModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := data.Timeouts.Create(ctx, DefaultCreateTimeout)
	resp.Diagnostics.Append(diags...)

	readTimeout, diags := data.Timeouts.Read(ctx, DefaultReadTimeout)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	createCtx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	if data.DefaultStack.ValueBool() {
		r.createInDefaultStack(createCtx, &data, resp)
	} else {
		r.createAsNewStack(createCtx, &data, resp)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// The read-back gets its own budget, so that an exhausted create timeout
	// does not also fail the read; that would leave the state unwritten and the
	// stack we just created untracked.
	readCtx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	resp.Diagnostics.Append(r.read(readCtx, &data, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Resource) createAsNewStack(ctx context.Context, data *ContainerStackModel, resp *resource.CreateResponse) {
	client := apiext.NewContainerClient(r.client)

	declareRequest := data.ToDeclareRequest(ctx, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		tflog.Debug(ctx, "error while building declare request")
		return
	}

	stack := providerutil.
		Try[*containerv2.StackResponse](&resp.Diagnostics, "API error while declaring stack").
		DoValResp(client.DeclareStack(ctx, *declareRequest))
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(stack.Id)

	waitUntilStackIsReady(ctx, client, stack.Id, nil, createTimeoutHint, &resp.Diagnostics)

	if !data.UpdateSchedule.IsNull() && !data.UpdateSchedule.IsUnknown() {
		r.reconcileUpdateSchedule(ctx, data, &resp.Diagnostics)
	}
}

func (r *Resource) createInDefaultStack(ctx context.Context, data *ContainerStackModel, resp *resource.CreateResponse) {
	var current ContainerStackModel

	client := apiext.NewContainerClient(r.client)

	stack, err := client.PollDefaultStack(ctx, data.ProjectID.ValueString())
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			resp.Diagnostics.AddError(
				"failed to get default stack",
				"the default stack of project "+data.ProjectID.ValueString()+" did not become available in time. "+
					createTimeoutHint,
			)
		} else {
			resp.Diagnostics.AddError("failed to get default stack", err.Error())
		}

		return
	}

	ctx = tflog.SetField(ctx, "stack_id", stack.Id)
	tflog.Debug(ctx, "using project default stack")

	data.ID = types.StringValue(stack.Id)

	updateRequest := data.ToUpdateRequest(ctx, &current, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		tflog.Debug(ctx, "error while building update request")
		return
	}

	_ = providerutil.
		Try[*containerv2.StackResponse](&resp.Diagnostics, "API error while declaring stack").
		DoValResp(client.UpdateStack(ctx, *updateRequest))

	// Without this, a failed update would still spend the entire create budget
	// waiting for containers that were never asked to change.
	if resp.Diagnostics.HasError() {
		return
	}

	waitUntilStackIsReady(ctx, client, stack.Id, data.ContainerNames(), createTimeoutHint, &resp.Diagnostics)

	if !data.UpdateSchedule.IsNull() && !data.UpdateSchedule.IsUnknown() {
		r.reconcileUpdateSchedule(ctx, data, &resp.Diagnostics)
	}
}

// reconcileUpdateSchedule calls SetStackUpdateSchedule to set or unset the
// update schedule for the stack. When update_schedule is null, an empty body
// is sent to unset any previously configured schedule.
func (r *Resource) reconcileUpdateSchedule(ctx context.Context, data *ContainerStackModel, d *diag.Diagnostics) {
	scheduleRequest := data.ToUpdateScheduleRequest(ctx, d)
	if d.HasError() || scheduleRequest == nil {
		return
	}

	providerutil.Try[any](d, "API error while setting update schedule").
		DoResp(r.client.Container().SetStackUpdateSchedule(ctx, *scheduleRequest))
}
