package provider

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
	"github.com/mittwald/terraform-provider-mittwald/internal/mittwaldv2"
	databasev2 "github.com/mittwald/terraform-provider-mittwald/internal/mittwaldv2/models/database"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &MySQLDatabaseResource{}
var _ resource.ResourceWithImportState = &MySQLDatabaseResource{}

func NewMySQLDatabaseResource() resource.Resource {
	return &MySQLDatabaseResource{}
}

type MySQLDatabaseResource struct {
	client *mittwaldv2.Client
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
				MarkdownDescription: "Hostname of the database",
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
						MarkdownDescription: "Charset of the database, e.g. `utf8mb4`",
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
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*mittwaldv2.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
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

	createURL := fmt.Sprintf("/projects/%s/mysql-databases", data.ProjectID.ValueString())
	createReq := databasev2.CreateMySQLDatabaseRequest{
		Database: databasev2.CreateMySQLDatabaseRequestDatabase{
			Description: data.Description.ValueString(),
			Version:     data.Version.ValueString(),
			CharacterSettings: databasev2.CreateMySQLDatabaseRequestDatabaseCharacterSettings{
				CharacterSet: dataCharset.Charset.ValueString(),
				Collation:    dataCharset.Collation.ValueString(),
			},
		},
		User: databasev2.CreateMySQLDatabaseRequestUser{
			Password:    dataUser.Password.ValueString(),
			AccessLevel: dataUser.AccessLevel.ValueString(),
		},
	}
	createRes := databasev2.CreateMySQLDatabaseResponse{}

	if err := d.client.Post(ctx, createURL, &createReq, &createRes); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create database (%s), got error: %s", createURL, err))
		return
	}

	data.ID = types.StringValue(createRes.ID)
	dataUser.ID = types.StringValue(createRes.UserID)

	resp.Diagnostics.Append(d.read(ctx, &data, &dataCharset)...)
	resp.Diagnostics.Append(d.readUser(ctx, createRes.ID, &dataUser)...)

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
	database := databasev2.MySQLDatabase{}
	databaseURL := fmt.Sprintf("/mysql-databases/%s", data.ID.ValueString())

	if err := d.client.Poll(ctx, databaseURL, &database); err != nil {
		res.AddError("Client Error", fmt.Sprintf("Unable to read database (%s), got error: %s", databaseURL, err))
		return
	}

	data.Name = types.StringValue(database.Name)
	data.Hostname = types.StringValue(database.Hostname)
	data.Description = types.StringValue(database.Description)
	data.Version = types.StringValue(database.Version)
	data.ProjectID = types.StringValue(database.ProjectID)

	charset.Charset = types.StringValue(database.CharacterSettings.CharacterSet)
	charset.Collation = types.StringValue(database.CharacterSettings.Collation)

	return
}

func (d *MySQLDatabaseResource) readUser(ctx context.Context, databaseID string, data *MySQLDatabaseUserModel) (res diag.Diagnostics) {
	if data.ID.IsNull() {
		databaseUserList := make([]databasev2.MySQLUser, 0)
		databaseUserListURL := fmt.Sprintf("/mysql-databases/%s/users", databaseID)

		if err := d.client.Poll(ctx, databaseUserListURL, &databaseUserList); err != nil {
			res.AddError("Client Error", fmt.Sprintf("Unable to read database user list (%s), got error: %s", databaseUserListURL, err))
			return
		}

		for _, user := range databaseUserList {
			if user.MainUser {
				data.ID = types.StringValue(user.ID)
				break
			}
		}
	}

	databaseUser := databasev2.MySQLUser{}
	databaseUserURL := fmt.Sprintf("/mysql-users/%s", data.ID.ValueString())

	if err := d.client.Poll(ctx, databaseUserURL, &databaseUser); err != nil {
		res.AddError("Client Error", fmt.Sprintf("Unable to read database user (%s), got error: %s", databaseUserURL, err))
		return
	}

	data.Name = types.StringValue(databaseUser.Name)
	data.AccessLevel = types.StringValue(databaseUser.AccessLevel)
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
		descrURL := fmt.Sprintf("/mysql-databases/%s/description", planData.ID.ValueString())
		descrBody := map[string]any{"description": planData.Description.ValueString()}

		if err := d.client.Patch(ctx, descrURL, &descrBody, nil); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update database description (%s), got error: %s", descrURL, err))
			return
		}
	}

	passwordURL := fmt.Sprintf("/mysql-users/%s/password", dataUser.ID.ValueString())
	passwordBody := map[string]any{"password": dataUser.Password.ValueString()}

	if err := d.client.Patch(ctx, passwordURL, &passwordBody, nil); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update database user password (%s), got error: %s", passwordURL, err))
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

	url := fmt.Sprintf("/mysql-databases/%s", data.ID.ValueString())
	if err := d.client.Delete(ctx, url); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete database (%s), got error: %s", url, err))
		return
	}
}

func (d *MySQLDatabaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
