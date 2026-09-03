package containerstackresource

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/containerclientv2"
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

	resp.Diagnostics.Append(r.read(readCtx, &data, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Resource) read(ctx context.Context, state, plan *ContainerStackModel) (res diag.Diagnostics) {
	stack, _, err := r.client.Container().GetStack(ctx, containerclientv2.GetStackRequest{StackID: state.ID.ValueString()})
	if err != nil {
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
