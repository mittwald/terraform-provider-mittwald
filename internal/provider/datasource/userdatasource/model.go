package userdatasource

import "github.com/hashicorp/terraform-plugin-framework/types"

type DataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	Email        types.String `tfsdk:"email"`
	FirstName    types.String `tfsdk:"first_name"`
	LastName     types.String `tfsdk:"last_name"`
	Title        types.String `tfsdk:"title"`
	RegisteredAt types.String `tfsdk:"registered_at"`
}
