package mysqldatabaseresource

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var charsetAttrs = map[string]attr.Type{
	"character_set": types.StringType,
	"collation":     types.StringType,
}

var userAttrs = map[string]attr.Type{
	"id":                  types.StringType,
	"name":                types.StringType,
	"password":            types.StringType,
	"password_wo":         types.StringType,
	"password_wo_version": types.Int64Type,
	"access_level":        types.StringType,
	"external_access":     types.BoolType,
}

func (m *MySQLDatabaseCharsetModel) AsObject(ctx context.Context, diag diag.Diagnostics) types.Object {
	val, d := types.ObjectValueFrom(ctx, charsetAttrs, m)
	diag.Append(d...)

	return val
}

func (m *MySQLDatabaseUserModel) AsObject(ctx context.Context, diag diag.Diagnostics) types.Object {
	val, d := types.ObjectValueFrom(ctx, userAttrs, m)
	diag.Append(d...)

	return val
}
