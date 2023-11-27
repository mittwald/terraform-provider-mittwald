package cronjobresource

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/terraform-provider-mittwald/api/mittwaldv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
	"github.com/mittwald/terraform-provider-mittwald/internal/ptrutil"
	"github.com/mittwald/terraform-provider-mittwald/internal/valueutil"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"strings"
	"text/scanner"
)

func (m *ResourceModel) FromAPIModel(ctx context.Context, apiModel *mittwaldv2.DeMittwaldV1CronjobCronjob) (res diag.Diagnostics) {
	m.AppID = valueutil.StringerOrNull(apiModel.AppId)
	m.Description = types.StringValue(apiModel.Description)
	m.Email = valueutil.StringPtrOrNull(apiModel.Email)
	m.Interval = types.StringValue(apiModel.Interval)

	asURL, err := apiModel.Destination.AsDeMittwaldV1CronjobCronjobUrl()
	if err != nil {
		res.AddError("error mapping cron job destination: %w", err.Error())
	}

	asCommand, err := apiModel.Destination.AsDeMittwaldV1CronjobCronjobCommand()
	if err != nil {
		res.AddError("error mapping cron job destination: %w", err.Error())
	}

	if asURL.Url != "" {
		m.Destination = ResourceDestinationURLModel(asURL.Url).AsDestinationModel().AsObject(ctx, res)
	} else {
		cmdModel := ResourceDestinationCommandModel{}

		res.Append(cmdModel.FromAPIModel(ctx, &asCommand)...)
		m.Destination = cmdModel.AsDestinationModel(ctx, res).AsObject(ctx, res)
	}

	return
}

func (m *ResourceModel) ToCreateRequest(ctx context.Context, d diag.Diagnostics) mittwaldv2.CronjobCreateCronjobJSONRequestBody {
	createCronjobBody := mittwaldv2.CronjobCreateCronjobJSONRequestBody{
		Description: m.Description.ValueString(),
		Active:      true,
		Interval:    m.Interval.ValueString(),
		AppId:       providerutil.ParseUUID(m.AppID.ValueString(), &d),
		Destination: mittwaldv2.DeMittwaldV1CronjobCronjobRequest_Destination{},
	}

	try := providerutil.Try[any](&d, "Mapping error while building cron job request")

	dest := m.GetDestination(ctx, d)
	if url, ok := dest.GetURL(ctx, d); ok {
		try.Do(createCronjobBody.Destination.FromDeMittwaldV1CronjobCronjobUrl(url.AsAPIModel()))
	}

	if cmd, ok := dest.GetCommand(ctx, d); ok {
		try.Do(createCronjobBody.Destination.FromDeMittwaldV1CronjobCronjobCommand(cmd.AsAPIModel()))
	}

	if !m.Email.IsNull() {
		e := openapi_types.Email(m.Email.ValueString())
		createCronjobBody.Email = &e
	}

	return createCronjobBody
}

func (m *ResourceModel) ToUpdateRequest(ctx context.Context, d diag.Diagnostics, current *ResourceModel) mittwaldv2.CronjobUpdateCronjobJSONRequestBody {
	body := mittwaldv2.CronjobUpdateCronjobJSONRequestBody{}
	try := providerutil.Try[any](&d, "Mapping error while building cron job request")

	if !m.Description.Equal(current.Description) {
		if !m.Description.IsNull() {
			body.Description = ptrutil.To(m.Description.ValueString())
		} else {
			// no known way to unset a description. :(
		}
	}

	if !m.Interval.Equal(current.Interval) {
		body.Interval = ptrutil.To(m.Interval.ValueString())
	}

	if !m.Email.Equal(current.Email) {
		if !m.Email.IsNull() {
			body.Email = ptrutil.To(openapi_types.Email(m.Email.ValueString()))
		} else {
			// no known way to unset the email address :(
		}
	}

	if !m.Destination.Equal(current.Destination) {
		body.Destination = &mittwaldv2.CronjobUpdateCronjobJSONBody_Destination{}

		dest := m.GetDestination(ctx, d)
		if url, ok := dest.GetURL(ctx, d); ok {
			try.Do(body.Destination.FromDeMittwaldV1CronjobCronjobUrl(url.AsAPIModel()))
		}

		if cmd, ok := dest.GetCommand(ctx, d); ok {
			try.Do(body.Destination.FromDeMittwaldV1CronjobCronjobCommand(cmd.AsAPIModel()))
		}
	}

	return body
}

func (m *ResourceDestinationCommandModel) FromAPIModel(ctx context.Context, apiModel *mittwaldv2.DeMittwaldV1CronjobCronjobCommand) (res diag.Diagnostics) {
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
	}

	return
}
