package serverresource

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/projectv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/valueutil"
)

func (r *ResourceModel) FromAPIModel(_ context.Context, apiModel *projectv2.Server) (diags diag.Diagnostics) {
	if apiModel == nil {
		r.ID = types.StringNull()
		r.ContractID = types.StringNull()
		r.ShortID = types.StringNull()
		r.MachineType = types.StringNull()
		r.Status = types.StringNull()
		r.ClusterName = types.StringNull()
		r.CreatedAt = types.StringNull()
		return
	}

	r.ID = types.StringValue(apiModel.Id)
	r.CustomerID = types.StringValue(apiModel.CustomerId)
	r.Description = types.StringValue(apiModel.Description)
	r.ShortID = types.StringValue(apiModel.ShortId)
	r.MachineType = types.StringValue(apiModel.MachineType.Name)
	r.Status = types.StringValue(string(apiModel.Status))
	r.ClusterName = types.StringValue(apiModel.ClusterName)
	r.CreatedAt = types.StringValue(apiModel.CreatedAt.Format(time.RFC3339))

	gib, err := valueutil.ParseStorageGiB(apiModel.Storage)
	if err != nil {
		diags.AddError("error while parsing server storage", err.Error())
		return
	}
	r.DiskspaceGB = types.Int64Value(gib)

	return
}
