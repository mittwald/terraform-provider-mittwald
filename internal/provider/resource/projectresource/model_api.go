package projectresource

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/terraform-provider-mittwald/api/mittwaldv2"
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

func (m *ResourceModel) FromAPIModel(ctx context.Context, project *mittwaldv2.DeMittwaldV1ProjectProject, ips []string) (res diag.Diagnostics) {
	if project == nil {
		m.Reset()
		return
	}

	m.ID = types.StringValue(project.Id.String())
	m.Description = types.StringValue(project.Description)
	m.Directories = providerutil.EmbedDiag(types.MapValueFrom(ctx, types.StringType, project.Directories))(&res)
	m.ServerID = valueutil.StringerOrNull(project.ServerId)
	m.DefaultIPs = providerutil.EmbedDiag(types.ListValueFrom(ctx, types.StringType, ips))(&res)

	return
}

func (m *ResourceModel) ToCreateRequest() mittwaldv2.ProjectCreateProjectJSONRequestBody {
	return mittwaldv2.ProjectCreateProjectJSONRequestBody{
		Description: m.Description.ValueString(),
	}
}
