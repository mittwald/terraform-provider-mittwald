package appresource

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var databaseModelAttrType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"id":      types.StringType,
		"kind":    types.StringType,
		"user_id": types.StringType,
		"purpose": types.StringType,
	},
}

type DatabaseModel struct {
	ID      types.String `tfsdk:"id"`
	Kind    types.String `tfsdk:"kind"`
	UserID  types.String `tfsdk:"user_id"`
	Purpose types.String `tfsdk:"purpose"`
}

func (m *DatabaseModel) Equals(other *DatabaseModel) bool {
	return m.ID.Equal(other.ID) &&
		m.Kind.Equal(other.Kind) &&
		m.UserID.Equal(other.UserID) &&
		m.Purpose.Equal(other.Purpose)
}
