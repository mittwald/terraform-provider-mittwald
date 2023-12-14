package mysqldatabaseresource

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (m *MySQLDatabaseCharsetModel) AsObject(ctx context.Context, diag diag.Diagnostics) types.Object {
	val, d := types.ObjectValueFrom(
		ctx,
		map[string]attr.Type{
			"character_set": types.StringType,
			"collation":     types.StringType,
		},
		m,
	)

	diag.Append(d...)

	return val
}

func (m *MySQLDatabaseUserModel) AsObject(ctx context.Context, diag diag.Diagnostics) types.Object {
	val, d := types.ObjectValueFrom(
		ctx,
		map[string]attr.Type{
			"id":              types.StringType,
			"name":            types.StringType,
			"password":        types.StringType,
			"access_level":    types.StringType,
			"external_access": types.BoolType,
		},
		m,
	)

	diag.Append(d...)

	return val
}
