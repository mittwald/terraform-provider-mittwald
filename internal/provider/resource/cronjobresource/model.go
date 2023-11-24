package cronjobresource

import (
	"context"
	"github.com/alessio/shellescape"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"strings"
)

type ResourceModel struct {
	ID          types.String `tfsdk:"id"`
	ProjectID   types.String `tfsdk:"project_id"`
	AppID       types.String `tfsdk:"app_id"`
	Description types.String `tfsdk:"description"`
	Interval    types.String `tfsdk:"interval"`
	Destination types.Object `tfsdk:"destination"`
	Email       types.String `tfsdk:"email"`
}

type ResourceDestinationModel struct {
	URL     types.String `tfsdk:"url"`
	Command types.Object `tfsdk:"command"`
}

type ResourceDestinationCommandModel struct {
	Interpreter types.String `tfsdk:"interpreter"`
	Path        types.String `tfsdk:"path"`
	Parameters  types.List   `tfsdk:"parameters"`
}

func (m *ResourceModel) GetDestination(ctx context.Context, d diag.Diagnostics) *ResourceDestinationModel {
	out := ResourceDestinationModel{}
	d.Append(m.Destination.As(ctx, &out, basetypes.ObjectAsOptions{})...)
	return &out
}

func (m *ResourceDestinationModel) GetURL(ctx context.Context, d diag.Diagnostics) (string, bool) {
	if m == nil {
		return "", false
	}

	if !m.URL.IsNull() {
		return m.URL.ValueString(), true
	}

	return "", false
}

func (m *ResourceDestinationModel) GetCommand(ctx context.Context, d diag.Diagnostics) (*ResourceDestinationCommandModel, bool) {
	if m == nil {
		return nil, false
	}

	if !m.Command.IsNull() {
		out := ResourceDestinationCommandModel{}
		d.Append(m.Command.As(ctx, &out, basetypes.ObjectAsOptions{})...)
		return &out, true
	}

	return nil, false
}

func (m *ResourceDestinationCommandModel) ParametersAsStrSlice() []string {
	elements := m.Parameters.Elements()
	out := make([]string, 0, len(elements))
	for _, v := range elements {
		out = append(out, v.String())
	}
	return out
}

func (m *ResourceDestinationCommandModel) ParametersAsStr() *string {
	if m.Parameters.IsNull() {
		return nil
	}

	elements := m.Parameters.Elements()
	out := make([]string, 0, len(elements))
	for _, v := range elements {
		out = append(out, shellescape.Quote(v.String()))
	}
	outAsStr := strings.Join(out, " ")
	return &outAsStr
}
