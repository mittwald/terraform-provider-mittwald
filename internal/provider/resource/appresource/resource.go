package appresource

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/appclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/appv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiext"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
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

func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app"
}

func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Models an app installation on the mittwald platform",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the app",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"short_id": schema.StringAttribute{
				MarkdownDescription: "The short ID of the app",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the project the app belongs to",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"databases": schema.SetNestedAttribute{
				MarkdownDescription: "The databases the app uses.\n\n" +
					"    You can use this field to link databases to the app. The database resources must be created before the app resource, and the database resources must have the same project ID as the app resource.\n\n" +
					"    This is only necessary if the specific app is not implicitly linked to a database by the backend. This is the case for apps like WordPress or TYPO3, which are always linked to a database. In these cases, you can (or should, even) omit the `databases` attribute. You can still retrieve the linked databases from the observed state, but you should not manage them manually.",
				Optional: true,
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "The ID of the database",
							Required:            true,
						},
						"user_id": schema.StringAttribute{
							MarkdownDescription: "The ID of the database user that the app should use",
							Required:            true,
						},
						"purpose": schema.StringAttribute{
							MarkdownDescription: "The purpose of the database; use 'primary' for the primary data storage, or 'cache' for a cache database",
							Required:            true,
						},
						"kind": schema.StringAttribute{
							MarkdownDescription: "The kind of the database; one of `mysql` or `redis`",
							Required:            true,
						},
					},
				},
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"app": schema.StringAttribute{
				MarkdownDescription: "The name of the app",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "The desired version of the app",
				Required:            true,
			},
			"version_current": schema.StringAttribute{
				MarkdownDescription: "The current version of the app",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the app",
				Required:            true,
			},
			"document_root": schema.StringAttribute{
				MarkdownDescription: "The document root of the app",
				Optional:            true,
			},
			"installation_path": schema.StringAttribute{
				MarkdownDescription: "The installation path of the app, relative to the web root",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"installation_path_absolute": schema.StringAttribute{
				MarkdownDescription: "The absolute installation path of the app, including the web root",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"update_policy": schema.StringAttribute{
				MarkdownDescription: "The update policy of the app; one of `none`, `patchLevel` or `all`",
				Required:            true,
			},
			"user_inputs": schema.MapAttribute{
				MarkdownDescription: "The user inputs of the app",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"dependencies": schema.MapNestedAttribute{
				MarkdownDescription: "The dependencies of the app.\n\n" +
					"    You can omit these to use the suggested dependencies for the app (in which case you can later select the dependencies from the resource state).\n\n" +
					"    If you specify dependencies, you must specify the exact version of the dependency. To select a version using a semantic versioning constraint, use the `mittwald_systemsoftware` data source.",
				Optional: true,
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"version": schema.StringAttribute{
							MarkdownDescription: "The version of the dependency; please take note that this must be an *exact* version string; to select a version using a semantic versioning constraint, use the `mittwald_systemsoftware` data source.",
							Required:            true,
						},
						"update_policy": schema.StringAttribute{
							MarkdownDescription: "The update policy of the dependency; one of `none`, `patchLevel` or `all`",
							Required:            true,
						},
					},
				},
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"ssh_host": schema.StringAttribute{
				MarkdownDescription: "The SSH host of the app; this will be populated after the app has been installed. You can use it for declaring a [provisioner](https://developer.hashicorp.com/terraform/language/resources/provisioners/connection) for your app.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.ClientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	data := ResourceModel{}
	databases := make([]DatabaseModel, 0)

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	// Databases may be unknown, in cases where linked database resources are determined by backend logic
	if !data.Databases.IsUnknown() {
		resp.Diagnostics.Append(data.Databases.ElementsAs(ctx, &databases, false)...)
	}

	appClient := apiext.NewAppClient(r.client)
	appInput, appUpdaters := data.ToCreateRequestWithUpdaters(ctx, &resp.Diagnostics, appClient)

	if resp.Diagnostics.HasError() {
		return
	}

	installation := providerutil.
		Try[*appclientv2.RequestAppinstallationResponse](&resp.Diagnostics, "error while requesting app installation").
		DoValResp(appClient.RequestAppinstallation(ctx, appInput))

	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(installation.Id)

	try := providerutil.Try[any](&resp.Diagnostics, "error while updating app installation")
	try.Do(appClient.UpdateAppinstallation(ctx, installation.Id, appUpdaters...))

	for _, database := range databases {
		try.DoResp(appClient.LinkDatabase(
			ctx,
			appclientv2.LinkDatabaseRequest{
				AppInstallationID: data.ID.ValueString(),
				Body: appclientv2.LinkDatabaseRequestBody{
					DatabaseId: database.ID.ValueString(),
					DatabaseUserIds: map[string]string{
						"admin": database.UserID.ValueString(),
					},
					Purpose: appclientv2.LinkDatabaseRequestBodyPurposePrimary,
				},
			},
		))
	}

	try.Do(appClient.WaitUntilAppInstallationIsReady(ctx, installation.Id))

	resp.Diagnostics.Append(r.read(ctx, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	data := ResourceModel{}

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.read(ctx, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Resource) read(ctx context.Context, data *ResourceModel) (res diag.Diagnostics) {
	appClient := r.client.App()

	appInstallation := providerutil.
		Try[*appv2.AppInstallation](&res, "error while fetching app installation").
		DoValResp(appClient.GetAppinstallation(ctx, appclientv2.GetAppinstallationRequest{AppInstallationID: data.ID.ValueString()}))

	if res.HasError() {
		return
	}

	res.Append(data.FromAPIModel(ctx, appInstallation, r.client)...)

	return
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	planData := ResourceModel{}
	currentData := ResourceModel{}

	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &currentData)...)

	appClient := apiext.NewAppClient(r.client)
	updaters := planData.ToUpdateUpdaters(ctx, resp.Diagnostics, &currentData, appClient)

	try := providerutil.Try[any](&resp.Diagnostics, "error while updating app installation")
	try.Do(appClient.UpdateAppinstallation(ctx, planData.ID.ValueString(), updaters...))

	linkedDatabasesInState := make([]DatabaseModel, 0)
	linkedDatabasesInPlan := make([]DatabaseModel, 0)

	resp.Diagnostics.Append(planData.Databases.ElementsAs(ctx, &linkedDatabasesInPlan, false)...)
	resp.Diagnostics.Append(currentData.Databases.ElementsAs(ctx, &linkedDatabasesInState, false)...)

	linkedDatabasesInStateByID := make(map[string]DatabaseModel)
	linkedDatabasesInPlanByID := make(map[string]DatabaseModel)

	for _, database := range linkedDatabasesInState {
		linkedDatabasesInStateByID[database.ID.ValueString()] = database
	}

	for _, database := range linkedDatabasesInPlan {
		linkedDatabasesInPlanByID[database.ID.ValueString()] = database

		existing, exists := linkedDatabasesInStateByID[database.ID.ValueString()]
		if exists && !existing.Equals(&database) {
			tflog.Debug(ctx, "database link changed; dropping", map[string]any{"database_id": database.ID.String()})
			try.DoResp(appClient.UnlinkDatabase(
				ctx,
				appclientv2.UnlinkDatabaseRequest{
					AppInstallationID: planData.ID.ValueString(),
					DatabaseID:        database.ID.ValueString(),
				},
			))
			exists = false
		}

		if !exists {
			tflog.Debug(ctx, "creating database link", map[string]any{"database_id": database.ID.String()})
			try.DoResp(appClient.LinkDatabase(
				ctx,
				appclientv2.LinkDatabaseRequest{
					AppInstallationID: planData.ID.ValueString(),
					Body: appclientv2.LinkDatabaseRequestBody{
						DatabaseId:      database.ID.ValueString(),
						DatabaseUserIds: map[string]string{"admin": database.UserID.ValueString()},
						Purpose:         appclientv2.LinkDatabaseRequestBodyPurpose(database.Purpose.ValueString()),
					},
				},
			))
		}
	}

	for _, database := range linkedDatabasesInState {
		_, planned := linkedDatabasesInPlanByID[database.ID.ValueString()]
		if !planned {
			tflog.Debug(ctx, "dropping database link", map[string]any{"database_id": database.ID.String()})
			try.DoResp(appClient.UnlinkDatabase(
				ctx,
				appclientv2.UnlinkDatabaseRequest{
					AppInstallationID: planData.ID.ValueString(),
					DatabaseID:        database.ID.ValueString(),
				},
			))
		}
	}

	resp.Diagnostics.Append(r.read(ctx, &planData)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &planData)...)
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	providerutil.
		Try[any](&resp.Diagnostics, "error while uninstalling app").
		DoResp(r.client.App().UninstallAppinstallation(ctx, data.ToDeleteRequest()))
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
