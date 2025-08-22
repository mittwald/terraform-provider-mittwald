package containerstackresource

import (
	"context"

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

	if data.DefaultStack.ValueBool() {
		r.createInDefaultStack(ctx, &data, resp)
	} else {
		r.createAsNewStack(ctx, &data, resp)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.read(ctx, &data)...)
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

	providerutil.Try[any](&resp.Diagnostics, "API error while waiting for stack to be ready").
		Do(client.WaitUntilStackIsReady(ctx, stack.Id, nil))

	data.ID = types.StringValue(stack.Id)
}

func (r *Resource) createInDefaultStack(ctx context.Context, data *ContainerStackModel, resp *resource.CreateResponse) {
	var current ContainerStackModel

	client := apiext.NewContainerClient(r.client)

	stack := providerutil.
		Try[*containerv2.StackResponse](&resp.Diagnostics, "failed to get default stack").
		DoVal(client.GetDefaultStack(ctx, data.ProjectID.ValueString()))

	if resp.Diagnostics.HasError() {
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

	providerutil.Try[any](&resp.Diagnostics, "API error while waiting for stack to be ready").
		Do(client.WaitUntilStackIsReady(ctx, stack.Id, data.ContainerNames()))
}
