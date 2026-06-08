package cronjobresource

import (
	"context"
	"fmt"
	"github.com/alessio/shellescape"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/cronjobclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/cronjobv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/ptrutil"
	"github.com/mittwald/terraform-provider-mittwald/internal/valueutil"
	"strings"
	"text/scanner"
)

func (m *ResourceModel) FromAPIModel(ctx context.Context, apiModel *cronjobv2.Cronjob) (res diag.Diagnostics) {
	m.ProjectID = valueutil.StringPtrOrNull(apiModel.ProjectId)
	m.Description = types.StringValue(apiModel.Description)
	m.Email = valueutil.StringPtrOrNull(apiModel.Email)
	m.Interval = types.StringValue(apiModel.Interval)
	m.Timezone = valueutil.StringPtrOrNull(apiModel.TimeZone)
	m.AppID = types.StringNull()
	m.Container = types.ObjectNull(resourceContainerAttrTypes)
	m.Destination = types.ObjectNull(resourceDestinationAttrTypes)

	if apiModel.Target != nil {
		if appTarget := apiModel.Target.AlternativeAppInstallationTarget; appTarget != nil {
			m.AppID = types.StringValue(appTarget.AppInstallationId)
			m.Destination = destinationFromAppTarget(ctx, &res, appTarget.Destination)
			return
		}

		if svcTarget := apiModel.Target.AlternativeServiceTargetResponse; svcTarget != nil {
			m.Container = containerObjectFromAPI(ctx, &res, svcTarget.StackId, svcTarget.ServiceShortId)
			m.Destination = destinationFromContainerCommand(ctx, &res, svcTarget.Command)
			return
		}
	}

	// Fallback for older API responses still using deprecated fields.
	m.AppID = types.StringValue(apiModel.AppId)
	if apiModel.Destination != nil {
		if u := apiModel.Destination.AlternativeCronjobUrl; u != nil {
			m.Destination = ResourceDestinationURLModel(u.Url).AsDestinationModel().AsObject(ctx, &res)
		}

		if c := apiModel.Destination.AlternativeCronjobCommand; c != nil {
			cmdModel := ResourceDestinationCommandModel{}

			res.Append(cmdModel.FromAPIModel(ctx, c)...)
			m.Destination = cmdModel.AsDestinationModel(ctx, &res).AsObject(ctx, &res)
		}
	}
	return
}

func (m *ResourceModel) ToCreateRequest(ctx context.Context, d *diag.Diagnostics) cronjobclientv2.CreateCronjobRequest {
	createCronjobBody := cronjobv2.CronjobRequest{
		Description: m.Description.ValueString(),
		Active:      true,
		Interval:    m.Interval.ValueString(),
		Target:      m.toRequestTarget(ctx, d),
	}

	if !m.Email.IsNull() {
		createCronjobBody.Email = m.Email.ValueStringPointer()
	}

	if !m.Timezone.IsNull() {
		createCronjobBody.TimeZone = m.Timezone.ValueStringPointer()
	}

	return cronjobclientv2.CreateCronjobRequest{
		ProjectID: m.ProjectID.ValueString(),
		Body:      createCronjobBody,
	}
}

func (m *ResourceModel) ToUpdateRequest(ctx context.Context, d *diag.Diagnostics, current *ResourceModel) cronjobclientv2.UpdateCronjobRequest {
	body := cronjobclientv2.UpdateCronjobRequestBody{}

	if !m.Description.Equal(current.Description) && !m.Description.IsNull() {
		body.Description = ptrutil.To(m.Description.ValueString())
	}

	if !m.Interval.Equal(current.Interval) {
		body.Interval = ptrutil.To(m.Interval.ValueString())
	}

	if !m.Email.Equal(current.Email) && !m.Email.IsNull() {
		body.Email = m.Email.ValueStringPointer()
	}

	if !m.Timezone.Equal(current.Timezone) && !m.Timezone.IsNull() {
		body.TimeZone = m.Timezone.ValueStringPointer()
	}

	if !m.Destination.Equal(current.Destination) || !m.AppID.Equal(current.AppID) || !m.Container.Equal(current.Container) {
		body.Target = m.toUpdateRequestTarget(ctx, d)
	}

	return cronjobclientv2.UpdateCronjobRequest{
		Body:      body,
		CronjobID: m.ID.ValueString(),
	}
}

func (m *ResourceModel) ToDeleteRequest() cronjobclientv2.DeleteCronjobRequest {
	return cronjobclientv2.DeleteCronjobRequest{
		CronjobID: m.ID.ValueString(),
	}
}

func (m *ResourceDestinationCommandModel) FromAPIModel(ctx context.Context, apiModel *cronjobv2.CronjobCommand) (res diag.Diagnostics) {
	m.Interpreter = types.StringValue(apiModel.Interpreter)
	m.Path = types.StringValue(apiModel.Path)

	if apiModel.Parameters != nil {
		paramScanner := scanner.Scanner{}
		paramScanner.Init(strings.NewReader(*apiModel.Parameters))

		var paramValues []string
		for tok := paramScanner.Scan(); tok != scanner.EOF; tok = paramScanner.Scan() {
			paramValues = append(paramValues, paramScanner.TokenText())
		}

		params, d := types.ListValueFrom(ctx, types.StringType, paramValues)

		res.Append(d...)
		m.Parameters = params
	} else {
		m.Parameters = types.ListNull(types.StringType)
	}

	return
}

func (m *ResourceModel) toRequestTarget(ctx context.Context, d *diag.Diagnostics) *cronjobv2.CronjobRequestTarget {
	dest := m.GetDestination(ctx, d)
	if container, ok := m.GetContainer(ctx, d); ok {
		serviceTarget := cronjobv2.ServiceTarget{
			StackId:           container.StackID.ValueString(),
			ServiceIdentifier: container.ServiceID.ValueString(),
			Command:           renderContainerCommand(ctx, d, dest.ContainerCommand),
		}

		return &cronjobv2.CronjobRequestTarget{AlternativeServiceTarget: &serviceTarget}
	}

	appTarget := cronjobv2.AppInstallationTarget{
		AppInstallationId: m.AppID.ValueString(),
	}
	if url, ok := dest.GetURL(ctx, d); ok {
		urlModel := url.AsAPIModel()
		appTarget.Destination.AlternativeCronjobUrl = &urlModel
	}
	if cmd, ok := dest.GetCommand(ctx, d); ok {
		cmdModel := cmd.AsAPIModel()
		appTarget.Destination.AlternativeCronjobCommand = &cmdModel
	}

	return &cronjobv2.CronjobRequestTarget{AlternativeAppInstallationTarget: &appTarget}
}

func (m *ResourceModel) toUpdateRequestTarget(ctx context.Context, d *diag.Diagnostics) *cronjobclientv2.UpdateCronjobRequestBodyTarget {
	dest := m.GetDestination(ctx, d)
	if container, ok := m.GetContainer(ctx, d); ok {
		serviceTarget := cronjobv2.ServiceTarget{
			StackId:           container.StackID.ValueString(),
			ServiceIdentifier: container.ServiceID.ValueString(),
			Command:           renderContainerCommand(ctx, d, dest.ContainerCommand),
		}

		return &cronjobclientv2.UpdateCronjobRequestBodyTarget{AlternativeServiceTarget: &serviceTarget}
	}

	appTarget := cronjobv2.AppInstallationTarget{
		AppInstallationId: m.AppID.ValueString(),
	}
	if url, ok := dest.GetURL(ctx, d); ok {
		urlModel := url.AsAPIModel()
		appTarget.Destination.AlternativeCronjobUrl = &urlModel
	}
	if cmd, ok := dest.GetCommand(ctx, d); ok {
		cmdModel := cmd.AsAPIModel()
		appTarget.Destination.AlternativeCronjobCommand = &cmdModel
	}

	return &cronjobclientv2.UpdateCronjobRequestBodyTarget{AlternativeAppInstallationTarget: &appTarget}
}

func renderContainerCommand(ctx context.Context, d *diag.Diagnostics, commandList types.List) string {
	var commandParts []string
	d.Append(commandList.ElementsAs(ctx, &commandParts, false)...)
	if len(commandParts) == 0 {
		return ""
	}

	quoted := make([]string, 0, len(commandParts))
	for _, p := range commandParts {
		quoted = append(quoted, shellescape.Quote(p))
	}

	return strings.Join(quoted, " ")
}

func destinationFromContainerCommand(ctx context.Context, d *diag.Diagnostics, command string) types.Object {
	commandParts, err := splitShellCommand(command)
	if err != nil {
		d.AddWarning(
			"Could not parse container command from API response",
			fmt.Sprintf("The command %q could not be split into individual arguments (%s). The full command is kept as a single list element.", command, err),
		)
		// Keep provider state stable even for unexpected command strings returned by the API.
		commandParts = []string{command}
	}

	out := ResourceDestinationModel{
		URL:     types.StringNull(),
		Command: types.ObjectNull(resourceDestinationCommandAttrTypes),
	}
	out.ContainerCommand, _ = types.ListValueFrom(ctx, types.StringType, commandParts)
	return out.AsObject(ctx, d)
}

func destinationFromAppTarget(ctx context.Context, d *diag.Diagnostics, destination cronjobv2.AppInstallationTargetDestination) types.Object {
	if u := destination.AlternativeCronjobUrl; u != nil {
		return ResourceDestinationURLModel(u.Url).AsDestinationModel().AsObject(ctx, d)
	}

	if c := destination.AlternativeCronjobCommand; c != nil {
		cmdModel := ResourceDestinationCommandModel{}
		d.Append(cmdModel.FromAPIModel(ctx, c)...)
		return cmdModel.AsDestinationModel(ctx, d).AsObject(ctx, d)
	}

	return types.ObjectNull(resourceDestinationAttrTypes)
}

func containerObjectFromAPI(ctx context.Context, d *diag.Diagnostics, stackID, serviceID string) types.Object {
	obj, d2 := types.ObjectValueFrom(ctx, resourceContainerAttrTypes, ResourceContainerModel{
		StackID:   types.StringValue(stackID),
		ServiceID: types.StringValue(serviceID),
	})
	d.Append(d2...)
	return obj
}

func splitShellCommand(input string) ([]string, error) {
	var (
		out          []string
		current      strings.Builder
		inSingle     bool
		inDouble     bool
		escapeActive bool
	)

	flush := func() {
		if current.Len() > 0 {
			out = append(out, current.String())
			current.Reset()
		}
	}

	for _, r := range input {
		switch {
		case escapeActive:
			current.WriteRune(r)
			escapeActive = false
		case inSingle:
			if r == '\'' {
				inSingle = false
				continue
			}
			current.WriteRune(r)
		case inDouble:
			switch r {
			case '"':
				inDouble = false
			case '\\':
				escapeActive = true
			default:
				current.WriteRune(r)
			}
		default:
			switch r {
			case '\'':
				inSingle = true
			case '"':
				inDouble = true
			case '\\':
				escapeActive = true
			case ' ', '\n', '\t':
				flush()
			default:
				current.WriteRune(r)
			}
		}
	}

	if escapeActive {
		return nil, fmt.Errorf("invalid shell command: trailing escape")
	}
	if inSingle {
		return nil, fmt.Errorf("invalid shell command: unclosed single quote")
	}
	if inDouble {
		return nil, fmt.Errorf("invalid shell command: unclosed double quote")
	}

	flush()
	return out, nil
}
