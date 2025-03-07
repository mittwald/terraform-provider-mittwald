package redisdatabaseresource

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/databaseclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/databasev2"
	"github.com/mittwald/terraform-provider-mittwald/internal/valueutil"
)

func (m *ResourceModel) ToCreateRequest(ctx context.Context, d *diag.Diagnostics) databaseclientv2.CreateRedisDatabaseRequest {
	return databaseclientv2.CreateRedisDatabaseRequest{
		ProjectID: m.ProjectID.ValueString(),
		Body: databaseclientv2.CreateRedisDatabaseRequestBody{
			Description:   m.Description.ValueString(),
			Version:       m.Version.ValueString(),
			Configuration: m.mapConfiguration(ctx, d),
		},
	}
}

func (m *ResourceModel) mapConfiguration(ctx context.Context, d *diag.Diagnostics) *databasev2.RedisDatabaseConfiguration {
	configurationModel := RedisConfigurationModel{}

	d.Append(m.Configuration.As(ctx, &configurationModel, basetypes.ObjectAsOptions{})...)

	additionalFlags := make([]string, 0, 8)
	for _, v := range configurationModel.AdditionalFlags.Elements() {
		if str, ok := v.(basetypes.StringValue); ok {
			additionalFlags = append(additionalFlags, str.ValueString())
		}
	}

	maxMemory := fmt.Sprintf("%dMi", configurationModel.MaxMemoryMB.ValueInt64())

	return &databasev2.RedisDatabaseConfiguration{
		AdditionalFlags: additionalFlags,
		MaxMemory:       &maxMemory,
		MaxMemoryPolicy: configurationModel.MaxMemoryPolicy.ValueStringPointer(),
		Persistent:      configurationModel.Persistent.ValueBoolPointer(),
	}
}

func (m *ResourceModel) ToUpdateDescriptionRequest() databaseclientv2.UpdateRedisDatabaseDescriptionRequest {
	return databaseclientv2.UpdateRedisDatabaseDescriptionRequest{
		RedisDatabaseID: m.ID.ValueString(),
		Body: databaseclientv2.UpdateRedisDatabaseDescriptionRequestBody{
			Description: m.Description.ValueString(),
		},
	}
}

func (m *ResourceModel) ToUpdateConfigurationRequest(ctx context.Context, d *diag.Diagnostics) databaseclientv2.UpdateRedisDatabaseConfigurationRequest {
	return databaseclientv2.UpdateRedisDatabaseConfigurationRequest{
		RedisDatabaseID: m.ID.ValueString(),
		Body: databaseclientv2.UpdateRedisDatabaseConfigurationRequestBody{
			Configuration: m.mapConfiguration(ctx, d),
		},
	}
}

func (m *ResourceModel) ToDeleteRequest() databaseclientv2.DeleteRedisDatabaseRequest {
	return databaseclientv2.DeleteRedisDatabaseRequest{
		RedisDatabaseID: m.ID.ValueString(),
	}
}

func (m *ResourceModel) FromAPIModel(ctx context.Context, database *databasev2.RedisDatabase) (res diag.Diagnostics) {

	m.Name = types.StringValue(database.Name)
	m.Hostname = types.StringValue(database.Hostname)
	m.Description = types.StringValue(database.Description)
	m.Version = types.StringValue(database.Version)
	m.ProjectID = types.StringValue(database.ProjectId)

	if database.Configuration != nil {
		configuration := RedisConfigurationModel{}
		res.Append(configuration.FromAPIModel(ctx, database.Configuration)...)

		m.Configuration = configuration.AsObject(ctx, res)
	} else {
		m.Configuration = types.ObjectNull(redisConfigurationAttrs)
	}

	return
}

func (m *RedisConfigurationModel) FromAPIModel(ctx context.Context, config *databasev2.RedisDatabaseConfiguration) (res diag.Diagnostics) {
	if maxmem := config.MaxMemory; maxmem != nil {
		maxMemoryBytes := valueutil.Int64FromByteQuantity(*maxmem, &res)
		m.MaxMemoryMB = types.Int64Value(maxMemoryBytes.ValueInt64() / 1024 / 1024)
	} else {
		m.MaxMemoryMB = types.Int64Null()
	}

	m.MaxMemoryPolicy = valueutil.StringPtrOrNull(config.MaxMemoryPolicy)
	m.Persistent = valueutil.BoolPtrOrNull(config.Persistent)
	m.AdditionalFlags, res = types.ListValueFrom(ctx, types.StringType, config.AdditionalFlags)
	return
}
