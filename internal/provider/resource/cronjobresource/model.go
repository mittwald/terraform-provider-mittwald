package cronjobresource

import (
	"context"
	"github.com/alessio/shellescape"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/mittwald/terraform-provider-mittwald/api/mittwaldv2"
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

type ResourceDestinationURLModel string

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

func (m *ResourceDestinationModel) GetURL(ctx context.Context, d diag.Diagnostics) (ResourceDestinationURLModel, bool) {
	if m == nil {
		return "", false
	}

	if !m.URL.IsNull() {
		return ResourceDestinationURLModel(m.URL.ValueString()), true
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

func (m *ResourceDestinationModel) AsObject(ctx context.Context, d diag.Diagnostics) types.Object {
	obj, d2 := types.ObjectValueFrom(ctx, map[string]attr.Type{
		"url": types.StringType,
		"command": types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"interpreter": types.StringType,
				"path":        types.StringType,
				"parameters":  types.ListType{ElemType: types.StringType},
			},
		},
	}, m)

	d.Append(d2...)
	return obj
}

func (m ResourceDestinationURLModel) AsAPIModel() mittwaldv2.DeMittwaldV1CronjobCronjobUrl {
	return mittwaldv2.DeMittwaldV1CronjobCronjobUrl{
		Url: string(m),
	}
}

func (m ResourceDestinationURLModel) AsDestinationModel() *ResourceDestinationModel {
	return &ResourceDestinationModel{
		URL: types.StringValue(string(m)),
	}
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

func (m *ResourceDestinationCommandModel) AsAPIModel() mittwaldv2.DeMittwaldV1CronjobCronjobCommand {
	return mittwaldv2.DeMittwaldV1CronjobCronjobCommand{
		Interpreter: m.Interpreter.ValueString(),
		Path:        m.Path.ValueString(),
		Parameters:  m.ParametersAsStr(),
	}
}

func (m *ResourceDestinationCommandModel) AsDestinationModel(ctx context.Context, diag diag.Diagnostics) *ResourceDestinationModel {
	value, d := types.ObjectValueFrom(ctx, map[string]attr.Type{
		"interpreter": types.StringType,
		"path":        types.StringType,
		"parameters":  types.ListType{ElemType: types.StringType},
	}, m)

	diag.Append(d...)

	return &ResourceDestinationModel{
		Command: value,
	}
}
