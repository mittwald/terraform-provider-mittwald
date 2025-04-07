package appdatasource

import "github.com/hashicorp/terraform-plugin-framework/types"

// AppDataSourceModel describes the data source data model.
type AppDataSourceModel struct {
	Name        types.String `tfsdk:"name"`
	Recommended types.Bool   `tfsdk:"recommended"`
	Selector    types.String `tfsdk:"selector"`

	Version   types.String `tfsdk:"version"`
	VersionID types.String `tfsdk:"version_id"`
}

func (m *AppDataSourceModel) SelectorOrDefault() string {
	if m.Selector.IsNull() {
		return "*"
	}
	return m.Selector.ValueString()
}
