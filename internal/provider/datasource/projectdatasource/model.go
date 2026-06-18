package projectdatasource

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/projectv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
	"github.com/mittwald/terraform-provider-mittwald/internal/valueutil"
)

// DataSourceModel describes the data model for the mittwald_project data source.
type DataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	ShortID     types.String `tfsdk:"short_id"`
	ServerID    types.String `tfsdk:"server_id"`
	Description types.String `tfsdk:"description"`
	Directories types.Map    `tfsdk:"directories"`
	DefaultIPs  types.List   `tfsdk:"default_ips"`
}

// FromAPIModel populates the data source model from the API project model and
// its default IP addresses.
func (m *DataSourceModel) FromAPIModel(ctx context.Context, project *projectv2.Project, ips []string) (res diag.Diagnostics) {
	m.ID = types.StringValue(project.Id)
	m.ShortID = types.StringValue(project.ShortId)
	m.Description = types.StringValue(project.Description)
	m.Directories = providerutil.EmbedDiag(types.MapValueFrom(ctx, types.StringType, project.Directories))(&res)
	m.ServerID = valueutil.StringPtrOrNull(project.ServerId)
	m.DefaultIPs = providerutil.EmbedDiag(types.ListValueFrom(ctx, types.StringType, ips))(&res)

	return
}
