package mysqlpassword

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ ephemeral.EphemeralResource = &Resource{}

func New() ephemeral.EphemeralResource {
	return &Resource{}
}

type Resource struct {
}

type MySQLPasswordModel struct {
	Length   types.Int32  `tfsdk:"length"`
	Password types.String `tfsdk:"password"`
}

func (m *MySQLPasswordModel) LengthOrDefault() int {
	if m.Length.IsNull() {
		return 16
	}

	return int(m.Length.ValueInt32())
}

func (r *Resource) Metadata(_ context.Context, request ephemeral.MetadataRequest, response *ephemeral.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_mysql_password"
}

func (r *Resource) Schema(_ context.Context, _ ephemeral.SchemaRequest, response *ephemeral.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Generate a random MySQL password compliant with the MySQL password policy.",
		Attributes: map[string]schema.Attribute{
			"length": schema.Int32Attribute{
				Description: "The desired length of the password. The default is 16.",
				Optional:    true,
			},
			"password": schema.StringAttribute{
				Description: "The generated password.",
				Computed:    true,
				Sensitive:   true,
			},
		},
	}
}

func (r *Resource) Open(ctx context.Context, request ephemeral.OpenRequest, response *ephemeral.OpenResponse) {
	model := MySQLPasswordModel{}

	response.Diagnostics.Append(request.Config.Get(ctx, &model)...)

	password, err := generatePassword(model.LengthOrDefault())
	if err != nil {
		response.Diagnostics.AddAttributeError(path.Root("password"), "Error generating password", "Unable to generate password: "+err.Error())
		return
	}

	model.Password = types.StringValue(password)

	response.Result.Set(ctx, &model)
}
