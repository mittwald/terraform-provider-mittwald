package serverresource

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/projectv2"
)

func (r *ResourceModel) FromAPIModel(_ context.Context, apiModel *projectv2.Server) diag.Diagnostics {
	if apiModel == nil {
		r.ID = types.StringNull()
		r.ContractID = types.StringNull()
		r.ShortID = types.StringNull()
		r.MachineType = types.StringNull()
		r.Status = types.StringNull()
		r.ClusterName = types.StringNull()
		r.CreatedAt = types.StringNull()
		return nil
	}

	r.ID = types.StringValue(apiModel.Id)
	r.CustomerID = types.StringValue(apiModel.CustomerId)
	r.Description = types.StringValue(apiModel.Description)
	r.ShortID = types.StringValue(apiModel.ShortId)
	r.MachineType = types.StringValue(apiModel.MachineType.Name)
	r.Status = types.StringValue(string(apiModel.Status))
	r.ClusterName = types.StringValue(apiModel.ClusterName)
	r.CreatedAt = types.StringValue(apiModel.CreatedAt.Format(time.RFC3339))

	if gib, ok := parseStorageGiB(apiModel.Storage); ok {
		r.DiskspaceGB = types.Int64Value(gib)
	}

	return nil
}

// parseStorageGiB parses a storage string such as "50Gi" into its integer GiB
// value. Returns false if the value cannot be parsed.
func parseStorageGiB(storage string) (int64, bool) {
	trimmed := strings.TrimSuffix(strings.TrimSpace(storage), "Gi")
	if trimmed == storage {
		// no "Gi" suffix; nothing we can reliably interpret
		return 0, false
	}

	value, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return 0, false
	}

	return value, true
}
