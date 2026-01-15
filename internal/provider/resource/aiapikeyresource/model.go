package aiapikeyresource

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/aihostingv2"
)

// ResourceModel describes the resource data model.
type ResourceModel struct {
	ID         types.String `tfsdk:"id"`
	CustomerID types.String `tfsdk:"customer_id"`
	ProjectID  types.String `tfsdk:"project_id"`
	Name       types.String `tfsdk:"name"`
	APIKey     types.String `tfsdk:"api_key"`
}

// FromAPIModel maps from the API model to the Terraform model.
func (m *ResourceModel) FromAPIModel(key *aihostingv2.Key) {
	m.ID = types.StringValue(key.KeyId)
	m.Name = types.StringValue(key.Name)
	m.APIKey = types.StringValue(key.Key)

	if key.CustomerId != nil {
		m.CustomerID = types.StringValue(*key.CustomerId)
	}

	if key.ProjectId != nil {
		m.ProjectID = types.StringValue(*key.ProjectId)
	}
}
