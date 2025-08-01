package mailoutboxresource

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/mailclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/mailv2"
)

// ToCreateRequest converts the resource model to an API create request
func (m *ResourceModel) ToCreateRequest() mailclientv2.CreateDeliveryboxRequest {
	return mailclientv2.CreateDeliveryboxRequest{
		Body: mailclientv2.CreateDeliveryboxRequestBody{
			Description: m.Description.ValueString(),
			Password:    m.Password.ValueString(),
		},
		ProjectID: m.ProjectID.ValueString(),
	}
}

// ToUpdateDescriptionRequest converts the resource model to an API update description request
func (m *ResourceModel) ToUpdateDescriptionRequest() mailclientv2.UpdateDeliveryBoxDescriptionRequest {
	return mailclientv2.UpdateDeliveryBoxDescriptionRequest{
		Body: mailclientv2.UpdateDeliveryBoxDescriptionRequestBody{
			Description: m.Description.ValueString(),
		},
		DeliveryBoxID: m.ID.ValueString(),
	}
}

// ToUpdatePasswordRequest converts the resource model to an API update password request
func (m *ResourceModel) ToUpdatePasswordRequest() mailclientv2.UpdateDeliveryBoxPasswordRequest {
	return mailclientv2.UpdateDeliveryBoxPasswordRequest{
		Body: mailclientv2.UpdateDeliveryBoxPasswordRequestBody{
			Password: m.Password.ValueString(),
		},
		DeliveryBoxID: m.ID.ValueString(),
	}
}

// ToDeleteRequest converts the resource model to an API delete request
func (m *ResourceModel) ToDeleteRequest() mailclientv2.DeleteDeliveryBoxRequest {
	return mailclientv2.DeleteDeliveryBoxRequest{
		DeliveryBoxID: m.ID.ValueString(),
	}
}

// ToGetRequest converts the resource model to an API get request
func (m *ResourceModel) ToGetRequest() mailclientv2.GetDeliveryBoxRequest {
	return mailclientv2.GetDeliveryBoxRequest{
		DeliveryBoxID: m.ID.ValueString(),
	}
}

// FromAPIModel converts an API response to the resource model
func (m *ResourceModel) FromAPIModel(_ context.Context, apiModel *mailv2.Deliverybox) diag.Diagnostics {
	var diags diag.Diagnostics

	if apiModel == nil {
		return diags
	}

	m.ID = types.StringValue(apiModel.Id)
	m.ProjectID = types.StringValue(apiModel.ProjectId)
	m.Description = types.StringValue(apiModel.Description)
	// Password is not returned from the API for security reasons

	return diags
}
