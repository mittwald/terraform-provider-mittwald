package containerstackresource

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/containerv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
)

// Delete is responsible for deleting a container stack.
//
// Implementation note: When the default stack is deleted, we will not actually
// delete it, but rather remove all containers and volumes that are known in the
// current state.
//
// This is necessary because additional stacks are not actually supported at the
// moment, and there would be no other way to manage multiple container_stack
// resources in the same configuration.
//
//	TODO(mhelmich): Assess whether this is still the right approach in the future,
//	  as soon as the mStudio API supports multiple stacks in the same project.
func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var stateData ContainerStackModel

	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// The "default" stack has a special role, and we will not delete it even if
	// the user requests it. Instead, we will simply purge all containers and volumes
	// from it that are known in the current state.
	if stateData.DefaultStack.ValueBool() {
		_ = providerutil.
			Try[*containerv2.StackResponse](&resp.Diagnostics, "API error while removing containers from stack").
			DoValResp(r.client.Container().UpdateStack(ctx, *stateData.ToDeletePatchRequest(ctx, &resp.Diagnostics)))
	} else {
		resp.Diagnostics.AddError("not implemented", "removing non-default stacks is not supported, yet")
	}
}
