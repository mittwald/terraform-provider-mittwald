package redisdatabaseresource

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var redisConfigurationAttrs = map[string]attr.Type{
	"additional_flags": types.ListType{
		ElemType: types.StringType,
	},
	"max_memory_mb":     types.Int64Type,
	"max_memory_policy": types.StringType,
	"persistent":        types.BoolType,
}

func (m *RedisConfigurationModel) AsObject(ctx context.Context, diag diag.Diagnostics) types.Object {
	val, d := types.ObjectValueFrom(ctx, redisConfigurationAttrs, m)
	diag.Append(d...)

	return val
}
