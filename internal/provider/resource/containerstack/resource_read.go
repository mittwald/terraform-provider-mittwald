package containerstackresource

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/containerclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/containerv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
)

// Read updates the state with the latest data from the API.
func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	data := ContainerStackModel{}

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.read(ctx, &data, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Resource) read(ctx context.Context, state, plan *ContainerStackModel) (res diag.Diagnostics) {
	stack := providerutil.
		Try[*containerv2.StackResponse](&res, "API error while fetching stack").
		DoValResp(r.client.Container().GetStack(ctx, containerclientv2.GetStackRequest{StackID: state.ID.ValueString()}))

	if res.HasError() {
		return
	}

	res.Append(state.FromAPIModel(ctx, stack, plan, true)...)

	return
}
