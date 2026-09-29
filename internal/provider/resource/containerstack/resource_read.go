package containerstackresource

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/containerclientv2"
	"github.com/mittwald/api-client-go/pkg/httperr"
)

// Read updates the state with the latest data from the API.
func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	data := ContainerStackModel{}

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, diags := data.Timeouts.Read(ctx, DefaultReadTimeout)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	readCtx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	notFound, diags := r.read(readCtx, &data, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The stack was deleted outside of Terraform; remove it from the state, so
	// that it will be re-created on the next apply.
	if notFound {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// read updates the state with the stack's current data from the API. notFound
// is true if the stack does not exist (anymore); in this case, the state is
// left untouched.
func (r *Resource) read(ctx context.Context, state, plan *ContainerStackModel) (notFound bool, res diag.Diagnostics) {
	stack, _, err := r.client.Container().GetStack(ctx, containerclientv2.GetStackRequest{StackID: state.ID.ValueString()})
	if err != nil {
		if errNotFound := new(httperr.ErrNotFound); errors.As(err, &errNotFound) {
			return true, res
		}

		if errors.Is(err, context.DeadlineExceeded) {
			res.AddError(
				"API error while fetching stack",
				"the stack "+state.ID.ValueString()+" could not be read in time. "+readTimeoutHint,
			)
		} else {
			res.AddError("API error while fetching stack", err.Error())
		}

		return
	}

	res.Append(state.FromAPIModel(ctx, stack, plan, true)...)

	return
}
