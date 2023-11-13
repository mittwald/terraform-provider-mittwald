package systemsoftwaredatasource

import "github.com/hashicorp/terraform-plugin-framework/types"

// DataSourceModel describes the data source data model.
type DataSourceModel struct {
	Name        types.String `tfsdk:"name"`
	Recommended types.Bool   `tfsdk:"recommended"`
	Selector    types.String `tfsdk:"selector"`

	Version   types.String `tfsdk:"version"`
	VersionID types.String `tfsdk:"version_id"`
}

func (m *DataSourceModel) SelectorOrDefault() string {
	if m.Selector.IsNull() {
		return "*"
	}
	return m.Selector.ValueString()
}
