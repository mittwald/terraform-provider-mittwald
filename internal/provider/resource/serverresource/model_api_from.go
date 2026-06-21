package serverresource

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/projectv2"
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

	gib, err := ParseStorageGiB(apiModel.Storage)
	if err != nil {
		diags.AddError("error while parsing server storage", err.Error())
		return
	}
	r.DiskspaceGB = types.Int64Value(gib)

	return
}

// ParseStorageGiB parses a storage string such as "50Gi" into its integer GiB
// value. It only accepts whole-gibibyte ("Gi") values; any other unit or format
// is rejected with an error, since silently misinterpreting it could lead to an
// incorrect disk size being recorded.
func ParseStorageGiB(storage string) (int64, error) {
	trimmed, ok := strings.CutSuffix(strings.TrimSpace(storage), "Gi")
	if !ok {
		return 0, fmt.Errorf("expected a gibibyte value with a \"Gi\" suffix, but got %q", storage)
	}

	value, err := strconv.ParseInt(strings.TrimSpace(trimmed), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse storage value %q as a whole number of GiB: %w", storage, err)
	}

	return value, nil
}
