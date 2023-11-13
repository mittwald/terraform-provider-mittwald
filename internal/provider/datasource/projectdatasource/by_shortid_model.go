package projectdatasource

import "github.com/hashicorp/terraform-plugin-framework/types"

// ByShortIdDataSourceModel describes the data source data model.
type ByShortIdDataSourceModel struct {
	Id      types.String `tfsdk:"id"`
	ShortId types.String `tfsdk:"short_id"`
}
