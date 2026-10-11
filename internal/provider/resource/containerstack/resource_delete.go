package containerstackresource

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/containerclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/containerv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
)

// Delete is responsible for deleting a container stack.
//
// Implementation note: When a resource manages a project's (legacy) default
// stack, we will not actually delete it, but rather remove all containers and
// volumes that are known in the current state.
//
// This is necessary because multiple container_stack resources may share the
// same default stack, each managing a subset of its containers; deleting the
// whole stack would also remove the containers of all other resources.
//
// All other stacks are owned exclusively by this resource, and are deleted.
func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var stateData ContainerStackModel

	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)

	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, diags := stateData.Timeouts.Delete(ctx, DefaultDeleteTimeout)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	// The "default" stack has a special role, and we will not delete it even if
	// the user requests it. Instead, we will simply purge all containers and volumes
	// from it that are known in the current state.
	if stateData.DefaultStack.ValueBool() {
		_ = providerutil.
			Try[*containerv2.StackResponse](&resp.Diagnostics, "API error while removing containers from stack").
			IgnoreNotFound().
			DoValResp(r.client.Container().UpdateStack(ctx, *stateData.ToDeletePatchRequest(ctx, &resp.Diagnostics)))

		return
	}

	providerutil.
		Try[any](&resp.Diagnostics, "API error while deleting stack").
		IgnoreNotFound().
		DoResp(r.client.Container().DeleteStack(ctx, containerclientv2.DeleteStackRequest{
			StackID: stateData.ID.ValueString(),
		}))
}
