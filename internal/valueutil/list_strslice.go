package valueutil

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func ConvertStringSliceToList(slice []string) types.List {
	values := make([]attr.Value, len(slice))
	for i, v := range slice {
		values[i] = types.StringValue(v)
	}
	list, _ := types.ListValue(types.StringType, values)
	return list
}
