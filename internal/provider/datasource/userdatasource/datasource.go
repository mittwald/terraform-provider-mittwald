package userdatasource

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/userclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/userv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
	"github.com/mittwald/terraform-provider-mittwald/internal/valueutil"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &DataSource{}

func New() datasource.DataSource {
	return &DataSource{}
}

// DataSource defines the data source implementation.
type DataSource struct {
	client mittwaldv2.Client
}

func (d *DataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *DataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `This data source selects information about the authenticated user`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The user ID; if omitted, the authenticated user is assumed",
				Optional:            true,
				Computed:            true,
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "The users email",
				Computed:            true,
			},
			"first_name": schema.StringAttribute{
				MarkdownDescription: "The users first name",
				Computed:            true,
			},
			"last_name": schema.StringAttribute{
				MarkdownDescription: "The users last name",
				Computed:            true,
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "The salutation of the user (`mr`, `ms` or `other`)",
				Computed:            true,
			},
			"registered_at": schema.StringAttribute{
				MarkdownDescription: "The registration timestamp of the user in RFC3339 format",
				Computed:            true,
			},
		},
	}
}

func (d *DataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = providerutil.ClientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (d *DataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	userID := "self"
	if !data.ID.IsNull() && !data.ID.IsUnknown() {
		userID = data.ID.ValueString()
	}

	user := providerutil.
		Try[*userv2.User](&resp.Diagnostics, "error while fetching user").
		DoValResp(d.client.User().GetUser(ctx, userclientv2.GetUserRequest{UserID: userID}))

	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(user.UserId)
	data.Email = valueutil.StringPtrOrNull(user.Email)
	data.FirstName = types.StringValue(user.Person.FirstName)
	data.LastName = types.StringValue(user.Person.LastName)
	data.Title = valueutil.StringPtrOrNull(user.Person.Title)
	if user.RegisteredAt != nil {
		data.RegisteredAt = types.StringValue(user.RegisteredAt.Format(time.RFC3339))
	} else {
		data.RegisteredAt = types.StringNull()
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
