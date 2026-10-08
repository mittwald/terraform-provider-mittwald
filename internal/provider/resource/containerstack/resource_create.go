package containerstackresource

import (
	"context"
	"errors"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/containerclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/containerv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiext"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
)

// Create creates a new container stack.
//
// Implementation note: There are two ways of "creating" a stack; which one is
// used depends on whether the (deprecated) `default_stack` attribute is set to
// true or not.
//
// In the former case, the project's legacy default stack must already exist,
// and we need to "update" it with the new containers. In this case, we also need
// to respect the fact that there may be containers or volumes in the default
// stack that are not part of the current plan. These should not be touched at
// all. New projects do not have a default stack anymore; for these, this case
// fails with an error.
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

	notFound, diags := r.read(readCtx, &data, &data)
	resp.Diagnostics.Append(diags...)
	if notFound {
		resp.Diagnostics.AddError("API error while fetching stack", "the stack could not be found after it was written")
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Resource) createAsNewStack(ctx context.Context, data *ContainerStackModel, resp *resource.CreateResponse) {
	client := apiext.NewContainerClient(r.client)

	description := DefaultStackDescription
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		description = data.Description.ValueString()
	}

	created := providerutil.
		Try[*containerv2.StackResponse](&resp.Diagnostics, "API error while creating stack").
		DoValResp(client.CreateStack(ctx, containerclientv2.CreateStackRequest{
			ProjectID: data.ProjectID.ValueString(),
			Body:      containerv2.CreateStack{Description: description},
		}))
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "stack_id", created.Id)
	tflog.Debug(ctx, "created new stack")

	data.ID = types.StringValue(created.Id)

	// Track the stack right away; if any of the following steps fail, Terraform
	// marks the resource as tainted instead of losing track of the new stack.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), created.Id)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), data.ProjectID)...)

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

	waitUntilStackIsReady(ctx, client, stack.Id, nil, createTimeoutHint, &resp.Diagnostics)

	// DeclareStack has no updateSchedule field, so a schedule set on a brand
	// new stack still requires a separate UpdateStack call.
	if !data.UpdateSchedule.IsNull() && !data.UpdateSchedule.IsUnknown() {
		r.reconcileUpdateSchedule(ctx, data, &resp.Diagnostics)
	}
}

func (r *Resource) createInDefaultStack(ctx context.Context, data *ContainerStackModel, resp *resource.CreateResponse) {
	var current ContainerStackModel

	client := apiext.NewContainerClient(r.client)

	stack, err := client.GetDefaultStack(ctx, data.ProjectID.ValueString())
	if err != nil {
		if noDefaultStack := new(apiext.ErrNoDefaultStack); errors.As(err, &noDefaultStack) {
			resp.Diagnostics.AddAttributeError(
				path.Root("default_stack"),
				"project has no default stack",
				"Project "+data.ProjectID.ValueString()+" does not have a default stack. Projects no longer come "+
					"with a default stack; remove the `default_stack` attribute to create a new stack instead.",
			)
		} else {
			resp.Diagnostics.AddError("failed to get default stack", err.Error())
		}

		return
	}

	resp.Diagnostics.AddAttributeWarning(
		path.Root("default_stack"),
		"using a legacy default stack",
		"This resource manages the legacy default stack of project "+data.ProjectID.ValueString()+". Projects "+
			"no longer come with a default stack; consider removing the `default_stack` attribute to manage a "+
			"stack of its own instead.",
	)

	ctx = tflog.SetField(ctx, "stack_id", stack.Id)
	tflog.Debug(ctx, "using project default stack")

	data.ID = types.StringValue(stack.Id)

	updateRequest := data.ToUpdateRequest(ctx, &current, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		tflog.Debug(ctx, "error while building update request")
		return
	}

	// The default stack already exists (and may already carry an update
	// schedule set outside of this resource), so a config that omits
	// update_schedule must actively clear it rather than leaving it
	// untouched — otherwise the created state would drift from the config
	// until the next Update. UpdateStack already carries an updateSchedule
	// field, so fold this into the same call instead of issuing a second one.
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		description := data.Description.ValueString()
		updateRequest.Body.Description = &description
	}

	var opts []func(req *http.Request) error

	if !data.UpdateSchedule.IsUnknown() {
		schedule, explicitClear, ok := data.resolveUpdateSchedule(ctx, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		if ok {
			updateRequest.Body.UpdateSchedule = schedule
			if explicitClear {
				opts = append(opts, withExplicitNullUpdateSchedule)
			}
		}
	}

	_ = providerutil.
		Try[*containerv2.StackResponse](&resp.Diagnostics, "API error while declaring stack").
		DoValResp(client.UpdateStack(ctx, *updateRequest, opts...))

	// Without this, a failed update would still spend the entire create budget
	// waiting for containers that were never asked to change.
	if resp.Diagnostics.HasError() {
		return
	}

	waitUntilStackIsReady(ctx, client, stack.Id, data.ContainerNames(), createTimeoutHint, &resp.Diagnostics)
}

// reconcileUpdateSchedule calls UpdateStack to set or unset the update
// schedule for the stack, for the cases where the stack's main body update
// went through DeclareStack rather than UpdateStack (which has no
// updateSchedule field of its own). When update_schedule is null, an
// explicit JSON null is forced onto the wire to unset any previously
// configured schedule.
func (r *Resource) reconcileUpdateSchedule(ctx context.Context, data *ContainerStackModel, d *diag.Diagnostics) {
	scheduleRequest, explicitClear := data.ToUpdateScheduleRequest(ctx, d)
	if d.HasError() || scheduleRequest == nil {
		return
	}

	var opts []func(req *http.Request) error
	if explicitClear {
		opts = append(opts, withExplicitNullUpdateSchedule)
	}

	providerutil.Try[*containerv2.StackResponse](d, "API error while setting update schedule").
		DoValResp(r.client.Container().UpdateStack(ctx, *scheduleRequest, opts...))
}
