package cronjobresource

import (
	"context"
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
	m.AppID = types.StringValue(apiModel.AppId)
	m.ProjectID = valueutil.StringPtrOrNull(apiModel.ProjectId)
	m.Description = types.StringValue(apiModel.Description)
	m.Email = valueutil.StringPtrOrNull(apiModel.Email)
	m.Interval = types.StringValue(apiModel.Interval)

	if u := apiModel.Destination.AlternativeCronjobUrl; u != nil {
		m.Destination = ResourceDestinationURLModel(u.Url).AsDestinationModel().AsObject(ctx, &res)
	}

	if c := apiModel.Destination.AlternativeCronjobCommand; c != nil {
		cmdModel := ResourceDestinationCommandModel{}

		res.Append(cmdModel.FromAPIModel(ctx, c)...)
		m.Destination = cmdModel.AsDestinationModel(ctx, &res).AsObject(ctx, &res)
	}

	return
}

func (m *ResourceModel) ToCreateRequest(ctx context.Context, d *diag.Diagnostics) cronjobclientv2.CreateCronjobRequest {
	createCronjobBody := cronjobv2.CronjobRequest{
		Description: m.Description.ValueString(),
		Active:      true,
		Interval:    m.Interval.ValueString(),
		AppId:       m.AppID.ValueString(),
		Destination: cronjobv2.CronjobRequestDestination{},
	}

	dest := m.GetDestination(ctx, d)
	if url, ok := dest.GetURL(ctx, d); ok {
		urlModel := url.AsAPIModel()
		createCronjobBody.Destination.AlternativeCronjobUrl = &urlModel
	}

	if cmd, ok := dest.GetCommand(ctx, d); ok {
		cmdModel := cmd.AsAPIModel()
		createCronjobBody.Destination.AlternativeCronjobCommand = &cmdModel
	}

	if !m.Email.IsNull() {
		createCronjobBody.Email = m.Email.ValueStringPointer()
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

	if !m.Destination.Equal(current.Destination) {
		body.Destination = &cronjobclientv2.UpdateCronjobRequestBodyDestination{}

		dest := m.GetDestination(ctx, d)
		if url, ok := dest.GetURL(ctx, d); ok {
			urlModel := url.AsAPIModel()
			body.Destination.AlternativeCronjobUrl = &urlModel
		}

		if cmd, ok := dest.GetCommand(ctx, d); ok {
			cmdModel := cmd.AsAPIModel()
			body.Destination.AlternativeCronjobCommand = &cmdModel
		}
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
