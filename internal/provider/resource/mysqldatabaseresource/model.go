package mysqldatabaseresource

import "github.com/hashicorp/terraform-plugin-framework/types"

// ResourceModel describes the resource data model.
type ResourceModel struct {
	ID          types.String `tfsdk:"id"`
	ProjectID   types.String `tfsdk:"project_id"`
	Version     types.String `tfsdk:"version"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Hostname    types.String `tfsdk:"hostname"`

	CharacterSettings types.Object `tfsdk:"character_settings"`
	User              types.Object `tfsdk:"user"`
}

type MySQLDatabaseUserModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Password          types.String `tfsdk:"password"`
	PasswordWO        types.String `tfsdk:"password_wo"`
	PasswordWOVersion types.Int64  `tfsdk:"password_wo_version"`
	AccessLevel       types.String `tfsdk:"access_level"`
	ExternalAccess    types.Bool   `tfsdk:"external_access"`
}

type MySQLDatabaseCharsetModel struct {
	Charset   types.String `tfsdk:"character_set"`
	Collation types.String `tfsdk:"collation"`
}
