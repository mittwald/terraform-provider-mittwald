package projectresource

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/projectv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
	"github.com/mittwald/terraform-provider-mittwald/internal/valueutil"
)

func (m *ResourceModel) Reset() {
	m.ID = types.StringNull()
	m.ShortID = types.StringNull()
	m.ServerID = types.StringNull()
	m.CustomerID = types.StringNull()
	m.Description = types.StringNull()
	m.Directories = types.MapNull(types.StringType)
	m.DefaultIPs = types.ListNull(types.StringType)
	m.DiskspaceGB = types.Int64Null()
}

// FromAPIModel maps an API project into the model.
//
// The article ID and contract ID are deliberately not derived here: they are
// not part of the project representation, and the contract keeps reporting the
// previous article while a tariff change is still pending. Both are resolved
// explicitly where they are needed (on read, import and for the data source).
func (m *ResourceModel) FromAPIModel(ctx context.Context, project *projectv2.Project, ips []string) (res diag.Diagnostics) {
	if project == nil {
		m.Reset()
		return
	}

	m.ID = types.StringValue(project.Id)
	m.ShortID = types.StringValue(project.ShortId)
	m.Description = types.StringValue(project.Description)
	m.Directories = providerutil.EmbedDiag(types.MapValueFrom(ctx, types.StringType, project.Directories))(&res)
	m.ServerID = valueutil.StringPtrOrNull(project.ServerId)
	m.CustomerID = types.StringValue(project.CustomerId)
	m.DefaultIPs = providerutil.EmbedDiag(types.ListValueFrom(ctx, types.StringType, ips))(&res)

	storage, ok := specStorage(project.Spec)
	if !ok {
		m.DiskspaceGB = types.Int64Null()
		return
	}

	gib, err := valueutil.ParseStorageGiB(storage)
	if err != nil {
		res.AddError("error while parsing project storage", err.Error())
		return
	}

	m.DiskspaceGB = types.Int64Value(gib)

	return
}

// specStorage extracts the storage quantity from a project spec, which the API
// models as a union of the possible tariff shapes.
func specStorage(spec *projectv2.ProjectSpec) (string, bool) {
	switch {
	case spec == nil:
		return "", false
	case spec.AlternativeHardwareSpec != nil:
		return spec.AlternativeHardwareSpec.Storage, true
	case spec.AlternativeVisitorSpec != nil:
		return spec.AlternativeVisitorSpec.Storage, true
	default:
		return "", false
	}
}
