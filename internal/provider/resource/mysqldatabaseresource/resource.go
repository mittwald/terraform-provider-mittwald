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
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/databaseclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/databasev2"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiutils"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/resource/common"
	"time"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &Resource{}
var _ resource.ResourceWithImportState = &Resource{}

func New() resource.Resource {
	return &Resource{}
}

type Resource struct {
	client mittwaldv2.Client
}

func (d *Resource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_mysql_database"
}

func (d *Resource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	builder := common.AttributeBuilderFor("database")
	response.Schema = schema.Schema{
		MarkdownDescription: "Models a MySQL database on the mittwald plattform",

		Attributes: map[string]schema.Attribute{
			"id": builder.Id(),
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
			"project_id":  builder.ProjectId(),
			"description": builder.Description(),
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
				Validators: []validator.Object{
					&mysqlPasswordValidator{},
				},
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
						Optional:            true,
						Sensitive:           true,
						MarkdownDescription: "Password for the database user",
						DeprecationMessage:  "This attribute is deprecated and will be removed in a future version. Use `password_wo` instead.",
					},
					"password_wo": schema.StringAttribute{
						Optional:            true,
						Sensitive:           true,
						MarkdownDescription: "Password for the database user; this field is mutually exclusive with `password` and will be used instead of it. The password is not stored in the database, but only used to create the user.",
						WriteOnly:           true,
					},
					"password_wo_version": schema.Int64Attribute{
						Optional:            true,
						MarkdownDescription: "Version of the password for the database user; this is required when using `password_wo`.",
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

	createRes, _, err := d.client.Database().CreateMysqlDatabase(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	dataUser.ID = types.StringValue(createRes.UserId)

	data.ID = types.StringValue(createRes.Id)
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

	readCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	resp.Diagnostics.Append(d.read(readCtx, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *Resource) read(ctx context.Context, data *ResourceModel) (res diag.Diagnostics) {
	client := d.client.Database()

	dataUser := MySQLDatabaseUserModel{}

	if !data.User.IsNull() {
		res.Append(data.User.As(ctx, &dataUser, basetypes.ObjectAsOptions{})...)
	}

	if res.HasError() {
		return
	}

	database := providerutil.
		Try[*databasev2.MySqlDatabase](&res, "error while reading database").
		IgnoreNotFound().
		DoVal(apiutils.Poll(ctx, apiutils.PollOpts{}, client.GetMysqlDatabase, databaseclientv2.GetMysqlDatabaseRequest{MysqlDatabaseID: data.ID.ValueString()}))
	databaseUser := providerutil.
		Try[*databasev2.MySqlUser](&res, "error while reading database user").
		DoVal(d.findDatabaseUser(ctx, data.ID.ValueString(), &dataUser))

	if res.HasError() {
		return
	}

	res.Append(data.FromAPIModel(ctx, database, databaseUser)...)

	return
}

func (d *Resource) findDatabaseUser(ctx context.Context, databaseID string, data *MySQLDatabaseUserModel) (*databasev2.MySqlUser, error) {
	client := d.client.Database()

	// This should be the regular case, in which we can simply look up the user by ID.
	if !data.ID.IsNull() {
		return apiutils.Poll(ctx, apiutils.PollOpts{}, client.GetMysqlUser, databaseclientv2.GetMysqlUserRequest{MysqlUserID: data.ID.ValueString()})
	}

	// If the user ID is not set, we need to look up the user by database ID and check which one is the main user.
	databaseUserList, _, err := client.ListMysqlUsers(ctx, databaseclientv2.ListMysqlUsersRequest{MysqlDatabaseID: databaseID})
	if err != nil {
		return nil, err
	}

	for _, user := range *databaseUserList {
		if user.MainUser {
			data.ID = types.StringValue(user.Id)
			return &user, nil
		}
	}

	return nil, fmt.Errorf("could not find main user for database %s", databaseID)
}

func (d *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	planData, planUser, _ := d.unpack(ctx, req.Plan, &resp.Diagnostics)
	stateData, stateUser, _ := d.unpack(ctx, req.Plan, &resp.Diagnostics)

	if resp.Diagnostics.HasError() {
		return
	}

	// TODO: Update charsets
	d.updateDescription(ctx, &planData, &stateData, resp)
	d.updatePasswordDeprecated(ctx, &planUser, resp)
	d.updatePassword(ctx, &planUser, &stateUser, resp)
}

func (d *Resource) unpack(ctx context.Context, planOrState interface {
	Get(context.Context, any) diag.Diagnostics
}, diags *diag.Diagnostics) (data ResourceModel, user MySQLDatabaseUserModel, charset MySQLDatabaseCharsetModel) {
	diags.Append(planOrState.Get(ctx, &data)...)
	diags.Append(data.User.As(ctx, &user, basetypes.ObjectAsOptions{})...)
	diags.Append(data.CharacterSettings.As(ctx, &charset, basetypes.ObjectAsOptions{})...)

	return
}

func (d *Resource) updateDescription(ctx context.Context, planData, stateData *ResourceModel, resp *resource.UpdateResponse) {
	if planData.Description.Equal(stateData.Description) {
		return
	}

	client := d.client.Database()

	providerutil.
		Try[any](&resp.Diagnostics, "error while updating database").
		DoResp(client.UpdateMysqlDatabaseDescription(ctx, databaseclientv2.UpdateMysqlDatabaseDescriptionRequest{
			MysqlDatabaseID: planData.ID.ValueString(),
			Body: databaseclientv2.UpdateMysqlDatabaseDescriptionRequestBody{
				Description: planData.Description.ValueString(),
			},
		}))
}

func (d *Resource) updatePassword(ctx context.Context, planUser, stateUser *MySQLDatabaseUserModel, resp *resource.UpdateResponse) {
	hasWriteOnlyPassword := !planUser.PasswordWO.IsNull()
	isChanged := !planUser.PasswordWOVersion.Equal(stateUser.PasswordWOVersion)

	if !hasWriteOnlyPassword || !isChanged {
		return
	}

	d.updatePasswordInternal(ctx, planUser, planUser.PasswordWO.ValueString(), resp)
}

func (d *Resource) updatePasswordDeprecated(ctx context.Context, planUser *MySQLDatabaseUserModel, resp *resource.UpdateResponse) {
	if planUser.Password.IsNull() {
		return
	}

	d.updatePasswordInternal(ctx, planUser, planUser.Password.ValueString(), resp)
}

func (d *Resource) updatePasswordInternal(ctx context.Context, planUser *MySQLDatabaseUserModel, password string, resp *resource.UpdateResponse) {
	client := d.client.Database()

	providerutil.
		Try[any](&resp.Diagnostics, "error while setting database user password").
		DoResp(client.UpdateMysqlUserPassword(ctx, databaseclientv2.UpdateMysqlUserPasswordRequest{
			MysqlUserID: planUser.ID.ValueString(),
			Body: databaseclientv2.UpdateMysqlUserPasswordRequestBody{
				Password: password,
			},
		}))
}

func (d *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() {
		tflog.Debug(ctx, "database is null, skipping deletion")
		return
	}

	providerutil.
		Try[any](&resp.Diagnostics, "error while deleting database").
		IgnoreNotFound().
		DoResp(d.client.Database().DeleteMysqlDatabase(ctx, data.ToDeleteRequest()))
}

func (d *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
