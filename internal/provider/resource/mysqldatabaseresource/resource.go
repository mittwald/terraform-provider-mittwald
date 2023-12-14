package mysqldatabaseresource

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/mittwald/terraform-provider-mittwald/api/mittwaldv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &Resource{}
var _ resource.ResourceWithImportState = &Resource{}

func New() resource.Resource {
	return &Resource{}
}

type Resource struct {
	client mittwaldv2.ClientBuilder
}

func (d *Resource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_mysql_database"
}

func (d *Resource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Models a MySQL database on the mittwald plattform",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "ID of this database",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Version of the database, e.g. `5.7`",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Name of the database, e.g. `db-XXXXX`",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "ID of the project this database belongs to",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Description for your database",
			},
			"hostname": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Hostname of the database; this is the hostname that you should use within the platform to connect to the database.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"character_settings": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"collation": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Collation of the database, e.g. `utf8mb4_general_ci`",
					},
					"character_set": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Character set of the database, e.g. `utf8mb4`",
					},
				},
			},
			"user": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "ID of the database user",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"name": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Name of the database user, e.g. `dbu-XXXXX`",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"password": schema.StringAttribute{
						Required:            true,
						Sensitive:           true,
						MarkdownDescription: "Password for the database user",
					},
					"access_level": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Access level for the database user, e.g. `full` or `readonly`",
					},
					"external_access": schema.BoolAttribute{
						Required:            true,
						MarkdownDescription: "Whether the database user should be accessible from outside the cluster",
					},
				},
			},
		},
	}
}

func (d *Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	d.client = providerutil.ClientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (d *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	data := ResourceModel{}
	dataUser := MySQLDatabaseUserModel{}
	dataCharset := MySQLDatabaseCharsetModel{}

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(data.User.As(ctx, &dataUser, basetypes.ObjectAsOptions{})...)
	resp.Diagnostics.Append(data.CharacterSettings.As(ctx, &dataCharset, basetypes.ObjectAsOptions{})...)

	if resp.Diagnostics.HasError() {
		return
	}

	createReq := data.ToCreateRequest(ctx, resp.Diagnostics)

	dbID, userID, err := d.client.Database().CreateMySQLDatabase(ctx, data.ProjectID.ValueString(), createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	dataUser.ID = types.StringValue(userID)

	data.ID = types.StringValue(dbID)
	data.User = dataUser.AsObject(ctx, resp.Diagnostics)

	resp.Diagnostics.Append(d.read(ctx, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	data := ResourceModel{}

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(d.read(ctx, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *Resource) read(ctx context.Context, data *ResourceModel) (res diag.Diagnostics) {
	client := d.client.Database()

	database := providerutil.
		Try[*mittwaldv2.DeMittwaldV1DatabaseMySqlDatabase](&res, "error while reading database").
		DoVal(client.PollMySQLDatabase(ctx, data.ID.ValueString()))
	databaseUser := providerutil.
		Try[*mittwaldv2.DeMittwaldV1DatabaseMySqlUser](&res, "error while reading database user").
		DoVal(d.findDatabaseUser(ctx, data.ID.ValueString(), &MySQLDatabaseUserModel{}))

	res.Append(data.FromAPIModel(ctx, database, databaseUser)...)

	return
}

func (d *Resource) findDatabaseUser(ctx context.Context, databaseID string, data *MySQLDatabaseUserModel) (*mittwaldv2.DeMittwaldV1DatabaseMySqlUser, error) {
	client := d.client.Database()

	// This should be the regular case, in which we can simply look up the user by ID.
	if !data.ID.IsNull() {
		return client.PollMySQLUser(ctx, data.ID.ValueString())
	}

	// If the user ID is not set, we need to look up the user by database ID and check which one is the main user.
	databaseUserList, err := client.PollMySQLUsersForDatabase(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	for _, user := range databaseUserList {
		if user.MainUser {
			data.ID = types.StringValue(user.Id.String())
			return &user, nil
		}
	}

	return nil, fmt.Errorf("could not find main user for database %s", databaseID)
}

func (d *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData, stateData ResourceModel

	client := d.client.Database()

	dataUser := MySQLDatabaseUserModel{}
	dataCharset := MySQLDatabaseCharsetModel{}

	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	resp.Diagnostics.Append(planData.User.As(ctx, &dataUser, basetypes.ObjectAsOptions{})...)
	resp.Diagnostics.Append(planData.CharacterSettings.As(ctx, &dataCharset, basetypes.ObjectAsOptions{})...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !planData.Description.Equal(stateData.Description) {
		providerutil.
			Try[any](&resp.Diagnostics, "error while updating database").
			Do(client.SetMySQLDatabaseDescription(ctx, planData.ID.ValueString(), planData.Description.ValueString()))
	}

	providerutil.
		Try[any](&resp.Diagnostics, "error while setting database user password").
		Do(client.SetMySQLUserPassword(ctx, dataUser.ID.ValueString(), dataUser.Password.ValueString()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &planData)...)
}

func (d *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	providerutil.
		Try[any](&resp.Diagnostics, "error while deleting database").
		Do(d.client.Database().DeleteMySQLDatabase(ctx, data.ID.ValueString()))
}

func (d *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
