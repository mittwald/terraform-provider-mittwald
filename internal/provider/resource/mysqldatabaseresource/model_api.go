package mysqldatabaseresource

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/mittwald/terraform-provider-mittwald/api/mittwaldv2"
)

func (m *ResourceModel) ToCreateRequest(ctx context.Context, d diag.Diagnostics) mittwaldv2.DatabaseCreateMysqlDatabaseJSONRequestBody {
	dataCharset := MySQLDatabaseCharsetModel{}
	dataUser := MySQLDatabaseUserModel{}

	d.Append(m.CharacterSettings.As(ctx, &dataCharset, basetypes.ObjectAsOptions{})...)
	d.Append(m.User.As(ctx, &dataUser, basetypes.ObjectAsOptions{})...)

	return mittwaldv2.DatabaseCreateMysqlDatabaseJSONRequestBody{
		Database: mittwaldv2.DeMittwaldV1DatabaseCreateMySqlDatabase{
			Description: m.Description.ValueString(),
			Version:     m.Version.ValueString(),
			CharacterSettings: &mittwaldv2.DeMittwaldV1DatabaseCharacterSettings{
				CharacterSet: dataCharset.Charset.ValueString(),
				Collation:    dataCharset.Collation.ValueString(),
			},
		},
		User: mittwaldv2.DeMittwaldV1DatabaseCreateMySqlUserWithDatabase{
			Password:    dataUser.Password.ValueString(),
			AccessLevel: mittwaldv2.DeMittwaldV1DatabaseCreateMySqlUserWithDatabaseAccessLevel(dataUser.AccessLevel.ValueString()),
		},
	}
}

func (m *ResourceModel) FromAPIModel(ctx context.Context, apiDatabase *mittwaldv2.DeMittwaldV1DatabaseMySqlDatabase, apiUser *mittwaldv2.DeMittwaldV1DatabaseMySqlUser) (res diag.Diagnostics) {
	characterSet := &MySQLDatabaseCharsetModel{}
	user := &MySQLDatabaseUserModel{}

	res.Append(m.CharacterSettings.As(ctx, &characterSet, basetypes.ObjectAsOptions{})...)
	res.Append(m.User.As(ctx, &user, basetypes.ObjectAsOptions{})...)

	m.Name = types.StringValue(apiDatabase.Name)
	m.Hostname = types.StringValue(apiDatabase.Hostname)
	m.Description = types.StringValue(apiDatabase.Description)
	m.Version = types.StringValue(apiDatabase.Version)
	m.ProjectID = types.StringValue(apiDatabase.ProjectId.String())

	characterSet.FromAPIModel(&apiDatabase.CharacterSettings)
	user.FromAPIModel(apiUser)

	m.CharacterSettings = characterSet.AsObject(ctx, res)
	m.User = user.AsObject(ctx, res)

	return
}

func (m *MySQLDatabaseCharsetModel) FromAPIModel(apiCharset *mittwaldv2.DeMittwaldV1DatabaseCharacterSettings) {
	m.Charset = types.StringValue(apiCharset.CharacterSet)
	m.Collation = types.StringValue(apiCharset.Collation)
}

func (m *MySQLDatabaseUserModel) FromAPIModel(apiUser *mittwaldv2.DeMittwaldV1DatabaseMySqlUser) {
	m.Name = types.StringValue(apiUser.Name)
	m.AccessLevel = types.StringValue(string(apiUser.AccessLevel))
	m.ExternalAccess = types.BoolValue(apiUser.ExternalAccess)
}
