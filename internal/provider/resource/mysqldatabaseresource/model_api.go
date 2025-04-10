package mysqldatabaseresource

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/databaseclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/databasev2"
)

func (m *ResourceModel) ToCreateRequest(ctx context.Context, d diag.Diagnostics) databaseclientv2.CreateMysqlDatabaseRequest {
	dataCharset := MySQLDatabaseCharsetModel{}
	dataUser := MySQLDatabaseUserModel{}

	d.Append(m.CharacterSettings.As(ctx, &dataCharset, basetypes.ObjectAsOptions{})...)
	d.Append(m.User.As(ctx, &dataUser, basetypes.ObjectAsOptions{})...)

	password := dataUser.Password.ValueString()
	if !dataUser.PasswordWO.IsNull() {
		password = dataUser.PasswordWO.ValueString()
	}

	return databaseclientv2.CreateMysqlDatabaseRequest{
		ProjectID: m.ProjectID.ValueString(),
		Body: databaseclientv2.CreateMysqlDatabaseRequestBody{
			Database: databasev2.CreateMySqlDatabase{
				Description: m.Description.ValueString(),
				Version:     m.Version.ValueString(),
				CharacterSettings: &databasev2.CharacterSettings{
					CharacterSet: dataCharset.Charset.ValueString(),
					Collation:    dataCharset.Collation.ValueString(),
				},
			},
			User: databasev2.CreateMySqlUserWithDatabase{
				Password:    password,
				AccessLevel: databasev2.CreateMySqlUserWithDatabaseAccessLevel(dataUser.AccessLevel.ValueString()),
			},
		},
	}
}

func (m *ResourceModel) ToDeleteRequest() databaseclientv2.DeleteMysqlDatabaseRequest {
	return databaseclientv2.DeleteMysqlDatabaseRequest{
		MysqlDatabaseID: m.ID.ValueString(),
	}
}

func (m *ResourceModel) Reset() {
	m.Name = types.StringNull()
	m.Hostname = types.StringNull()
	m.Description = types.StringNull()
	m.Version = types.StringNull()
	m.ProjectID = types.StringNull()
	m.CharacterSettings = types.ObjectNull(charsetAttrs)
	m.User = types.ObjectNull(userAttrs)
}

func (m *ResourceModel) FromAPIModel(ctx context.Context, apiDatabase *databasev2.MySqlDatabase, apiUser *databasev2.MySqlUser) (res diag.Diagnostics) {
	if apiDatabase == nil {
		m.Reset()
		return
	}

	characterSet := MySQLDatabaseCharsetModel{}
	user := MySQLDatabaseUserModel{}

	if !m.CharacterSettings.IsNull() {
		res.Append(m.CharacterSettings.As(ctx, &characterSet, basetypes.ObjectAsOptions{})...)
	}

	if !m.User.IsNull() {
		res.Append(m.User.As(ctx, &user, basetypes.ObjectAsOptions{})...)
	}

	m.Name = types.StringValue(apiDatabase.Name)
	m.Hostname = types.StringValue(apiDatabase.Hostname)
	m.Description = types.StringValue(apiDatabase.Description)
	m.Version = types.StringValue(apiDatabase.Version)
	m.ProjectID = types.StringValue(apiDatabase.ProjectId)

	characterSet.FromAPIModel(&apiDatabase.CharacterSettings)
	m.CharacterSettings = characterSet.AsObject(ctx, res)

	if apiUser != nil {
		user.FromAPIModel(apiUser)
		m.User = user.AsObject(ctx, res)
	} else {
		m.User = types.ObjectNull(userAttrs)
	}

	return
}

func (m *MySQLDatabaseCharsetModel) FromAPIModel(apiCharset *databasev2.CharacterSettings) {
	m.Charset = types.StringValue(apiCharset.CharacterSet)
	m.Collation = types.StringValue(apiCharset.Collation)
}

func (m *MySQLDatabaseUserModel) FromAPIModel(apiUser *databasev2.MySqlUser) {
	m.ID = types.StringValue(apiUser.Id)
	m.Name = types.StringValue(apiUser.Name)
	m.AccessLevel = types.StringValue(string(apiUser.AccessLevel))
	m.ExternalAccess = types.BoolValue(apiUser.ExternalAccess)
}
