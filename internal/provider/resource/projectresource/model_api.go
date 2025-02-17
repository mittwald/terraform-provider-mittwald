package projectresource

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/projectclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/projectv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
	"github.com/mittwald/terraform-provider-mittwald/internal/valueutil"
)

func (m *ResourceModel) Reset() {
	m.ID = types.StringNull()
	m.ServerID = types.StringNull()
	m.Description = types.StringNull()
	m.Directories = types.MapNull(types.StringType)
	m.DefaultIPs = types.ListNull(types.StringType)
}

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
	m.DefaultIPs = providerutil.EmbedDiag(types.ListValueFrom(ctx, types.StringType, ips))(&res)

	return
}

func (m *ResourceModel) ToCreateRequest() projectclientv2.CreateProjectRequest {
	return projectclientv2.CreateProjectRequest{
		ServerID: m.ServerID.ValueString(),
		Body: projectclientv2.CreateProjectRequestBody{
			Description: m.Description.ValueString(),
		},
	}
}
