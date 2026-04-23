package cronjobresource

import (
	"context"
	"github.com/alessio/shellescape"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/cronjobv2"
	"strings"
)

var resourceDestinationCommandAttrTypes = map[string]attr.Type{
	"interpreter": types.StringType,
	"path":        types.StringType,
	"parameters":  types.ListType{ElemType: types.StringType},
}

var resourceDestinationAttrTypes = map[string]attr.Type{
	"url":               types.StringType,
	"command":           types.ObjectType{AttrTypes: resourceDestinationCommandAttrTypes},
	"container_command": types.ListType{ElemType: types.StringType},
}

var resourceContainerAttrTypes = map[string]attr.Type{
	"stack_id":   types.StringType,
	"service_id": types.StringType,
}

type ResourceModel struct {
	ID          types.String `tfsdk:"id"`
	ProjectID   types.String `tfsdk:"project_id"`
	AppID       types.String `tfsdk:"app_id"`
	Container   types.Object `tfsdk:"container"`
	Description types.String `tfsdk:"description"`
	Interval    types.String `tfsdk:"interval"`
	Destination types.Object `tfsdk:"destination"`
	Email       types.String `tfsdk:"email"`
	Timezone    types.String `tfsdk:"timezone"`
}

type ResourceDestinationModel struct {
	URL              types.String `tfsdk:"url"`
	Command          types.Object `tfsdk:"command"`
	ContainerCommand types.List   `tfsdk:"container_command"`
}

type ResourceContainerModel struct {
	StackID   types.String `tfsdk:"stack_id"`
	ServiceID types.String `tfsdk:"service_id"`
}

type ResourceDestinationURLModel string

type ResourceDestinationCommandModel struct {
	Interpreter types.String `tfsdk:"interpreter"`
	Path        types.String `tfsdk:"path"`
	Parameters  types.List   `tfsdk:"parameters"`
}

func (m *ResourceModel) GetDestination(ctx context.Context, d *diag.Diagnostics) *ResourceDestinationModel {
	out := ResourceDestinationModel{}
	d.Append(m.Destination.As(ctx, &out, basetypes.ObjectAsOptions{})...)
	return &out
}

func (m *ResourceModel) GetContainer(ctx context.Context, d *diag.Diagnostics) (*ResourceContainerModel, bool) {
	if m.Container.IsNull() || m.Container.IsUnknown() {
		return nil, false
	}

	out := ResourceContainerModel{}
	d.Append(m.Container.As(ctx, &out, basetypes.ObjectAsOptions{})...)
	return &out, true
}

func (m *ResourceDestinationModel) GetURL(ctx context.Context, d *diag.Diagnostics) (ResourceDestinationURLModel, bool) {
	if m == nil {
		return "", false
	}

	if !m.URL.IsNull() && !m.URL.IsUnknown() {
		return ResourceDestinationURLModel(m.URL.ValueString()), true
	}

	return "", false
}

func (m *ResourceDestinationModel) GetCommand(ctx context.Context, d *diag.Diagnostics) (*ResourceDestinationCommandModel, bool) {
	if m == nil {
		return nil, false
	}

	if !m.Command.IsNull() && !m.Command.IsUnknown() {
		out := ResourceDestinationCommandModel{}
		d.Append(m.Command.As(ctx, &out, basetypes.ObjectAsOptions{})...)
		return &out, true
	}

	return nil, false
}

func (m *ResourceDestinationModel) GetContainerCommand(ctx context.Context, d *diag.Diagnostics) (types.List, bool) {
	if m == nil {
		return types.ListNull(types.StringType), false
	}

	if !m.ContainerCommand.IsNull() && !m.ContainerCommand.IsUnknown() {
		return m.ContainerCommand, true
	}

	return types.ListNull(types.StringType), false
}

func (m *ResourceDestinationModel) AsObject(ctx context.Context, d *diag.Diagnostics) types.Object {
	obj, d2 := types.ObjectValueFrom(ctx, map[string]attr.Type{
		"url":               resourceDestinationAttrTypes["url"],
		"command":           resourceDestinationAttrTypes["command"],
		"container_command": resourceDestinationAttrTypes["container_command"],
	}, m)

	d.Append(d2...)
	return obj
}

func (m ResourceDestinationURLModel) AsAPIModel() cronjobv2.CronjobUrl {
	return cronjobv2.CronjobUrl{
		Url: string(m),
	}
}

func (m ResourceDestinationURLModel) AsDestinationModel() *ResourceDestinationModel {
	return &ResourceDestinationModel{
		URL:              types.StringValue(string(m)),
		Command:          types.ObjectNull(resourceDestinationCommandAttrTypes),
		ContainerCommand: types.ListNull(types.StringType),
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

func (m *ResourceDestinationCommandModel) AsAPIModel() cronjobv2.CronjobCommand {
	return cronjobv2.CronjobCommand{
		Interpreter: m.Interpreter.ValueString(),
		Path:        m.Path.ValueString(),
		Parameters:  m.ParametersAsStr(),
	}
}

func (m *ResourceDestinationCommandModel) AsDestinationModel(ctx context.Context, diag *diag.Diagnostics) *ResourceDestinationModel {
	value, d := types.ObjectValueFrom(ctx, map[string]attr.Type{
		"interpreter": resourceDestinationCommandAttrTypes["interpreter"],
		"path":        resourceDestinationCommandAttrTypes["path"],
		"parameters":  resourceDestinationCommandAttrTypes["parameters"],
	}, m)

	diag.Append(d...)

	return &ResourceDestinationModel{
		URL:              types.StringNull(),
		Command:          value,
		ContainerCommand: types.ListNull(types.StringType),
	}
}
