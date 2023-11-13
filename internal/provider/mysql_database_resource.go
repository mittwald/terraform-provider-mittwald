package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/mittwald/terraform-provider-mittwald/api/mittwaldv2"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &MySQLDatabaseResource{}
var _ resource.ResourceWithImportState = &MySQLDatabaseResource{}

func NewMySQLDatabaseResource() resource.Resource {
	return &MySQLDatabaseResource{}
}

type MySQLDatabaseResource struct {
	client mittwaldv2.ClientBuilder
}

// ProjectResourceModel describes the resource data model.
type MySQLDatabaseResourceModel struct {
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
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Password       types.String `tfsdk:"password"`
	AccessLevel    types.String `tfsdk:"access_level"`
	ExternalAccess types.Bool   `tfsdk:"external_access"`
}

type MySQLDatabaseCharsetModel struct {
	Charset   types.String `tfsdk:"character_set"`
	Collation types.String `tfsdk:"collation"`
}

func (d *MySQLDatabaseResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_mysql_database"
}

func (d *MySQLDatabaseResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
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

func (d *MySQLDatabaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	d.client = clientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (d *MySQLDatabaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	data := MySQLDatabaseResourceModel{}
	dataUser := MySQLDatabaseUserModel{}
	dataCharset := MySQLDatabaseCharsetModel{}

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(data.User.As(ctx, &dataUser, basetypes.ObjectAsOptions{})...)
	resp.Diagnostics.Append(data.CharacterSettings.As(ctx, &dataCharset, basetypes.ObjectAsOptions{})...)

	if data.ProjectID.IsNull() {
		resp.Diagnostics.AddError("Invalid Input", "project_id is required")
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}

	createReq := mittwaldv2.DatabaseCreateMysqlDatabaseJSONRequestBody{
		Database: mittwaldv2.DeMittwaldV1DatabaseCreateMySqlDatabase{
			Description: data.Description.ValueString(),
			Version:     data.Version.ValueString(),
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

	dbID, userID, err := d.client.Database().CreateMySQLDatabase(ctx, data.ProjectID.ValueString(), createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	data.ID = types.StringValue(dbID)
	dataUser.ID = types.StringValue(userID)

	resp.Diagnostics.Append(d.read(ctx, &data, &dataCharset)...)
	resp.Diagnostics.Append(d.readUser(ctx, dbID, &dataUser)...)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user"), &dataUser)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("character_settings"), &dataCharset)...)
}

func (d *MySQLDatabaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	data := MySQLDatabaseResourceModel{}
	dataUser := MySQLDatabaseUserModel{}
	dataCharset := MySQLDatabaseCharsetModel{}

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(d.read(ctx, &data, &dataCharset)...)
	resp.Diagnostics.Append(d.readUser(ctx, data.ID.ValueString(), &dataUser)...)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user"), &dataUser)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("character_settings"), &dataCharset)...)
}

func (d *MySQLDatabaseResource) read(ctx context.Context, data *MySQLDatabaseResourceModel, charset *MySQLDatabaseCharsetModel) (res diag.Diagnostics) {
	database, err := d.client.Database().PollMySQLDatabase(ctx, data.ID.ValueString())
	if err != nil {
		res.AddError("Client Error", err.Error())
		return
	}

	data.Name = types.StringValue(database.Name)
	data.Hostname = types.StringValue(database.Hostname)
	data.Description = types.StringValue(database.Description)
	data.Version = types.StringValue(database.Version)
	data.ProjectID = types.StringValue(database.ProjectId.String())

	charset.Charset = types.StringValue(database.CharacterSettings.CharacterSet)
	charset.Collation = types.StringValue(database.CharacterSettings.Collation)

	return
}

func (d *MySQLDatabaseResource) readUser(ctx context.Context, databaseID string, data *MySQLDatabaseUserModel) (res diag.Diagnostics) {
	if data.ID.IsNull() {
		databaseUserList, err := d.client.Database().PollMySQLUsersForDatabase(ctx, databaseID)
		if err != nil {
			res.AddError("Client Error", err.Error())
			return
		}

		for _, user := range databaseUserList {
			if user.MainUser {
				data.ID = types.StringValue(user.Id.String())
				break
			}
		}
	}

	databaseUser, err := d.client.Database().PollMySQLUser(ctx, data.ID.ValueString())
	if err != nil {
		res.AddError("Client Error", err.Error())
		return
	}

	data.Name = types.StringValue(databaseUser.Name)
	data.AccessLevel = types.StringValue(string(databaseUser.AccessLevel))
	data.ExternalAccess = types.BoolValue(databaseUser.ExternalAccess)

	return
}

func (d *MySQLDatabaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData, stateData MySQLDatabaseResourceModel
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
		if err := d.client.Database().SetMySQLDatabaseDescription(ctx, planData.ID.ValueString(), planData.Description.ValueString()); err != nil {
			resp.Diagnostics.AddError("Client Error", err.Error())
			return
		}
	}

	if err := d.client.Database().SetMySQLUserPassword(ctx, dataUser.ID.ValueString(), dataUser.Password.ValueString()); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &planData)...)
	resp.Diagnostics.Append(d.read(ctx, &planData, &dataCharset)...)
	resp.Diagnostics.Append(d.readUser(ctx, planData.ID.ValueString(), &dataUser)...)

	//resp.Diagnostics.Append(resp.State.Set(ctx, &planData)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user"), &dataUser)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("character_settings"), &dataCharset)...)
}

func (d *MySQLDatabaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data MySQLDatabaseResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if err := d.client.Database().DeleteMySQLDatabase(ctx, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
}

func (d *MySQLDatabaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
