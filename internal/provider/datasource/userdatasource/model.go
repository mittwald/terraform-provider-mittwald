package userdatasource

import "github.com/hashicorp/terraform-plugin-framework/types"

type DataSourceModel struct {
	ID    types.String `tfsdk:"id"`
	Email types.String `tfsdk:"email"`
}
