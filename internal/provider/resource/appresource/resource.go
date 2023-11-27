package appresource

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/terraform-provider-mittwald/api/mittwaldv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &Resource{}
var _ resource.ResourceWithImportState = &Resource{}

var appNames = map[string]string{
	"static":    "d20baefd-81d2-42aa-bfba-9a3220ae839b",
	"php":       "34220303-cb87-4592-8a95-2eb20a97b2ac",
	"node":      "3e7f920b-a711-4d2f-9871-661e1b41a2f0",
	"wordpress": "da3aa3ae-4b6b-4398-a4a8-ee8def827876",
	"typo3":     "352971cc-b96a-4a26-8651-b08d7c8a7357",
	"shopware6": "12d54d05-7e55-4cf3-90c4-093516e0eaf8",
	"shopware5": "a23acf9c-9298-4082-9e7d-25356f9976dc",
}

func New() resource.Resource {
	return &Resource{}
}

type Resource struct {
	client mittwaldv2.ClientBuilder
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
			"project_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the project the app belongs to",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"database_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the database the app uses",
				Optional:            true,
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
				Optional:            true,
			},
			"document_root": schema.StringAttribute{
				MarkdownDescription: "The document root of the app",
				Optional:            true,
			},
			"installation_path": schema.StringAttribute{
				MarkdownDescription: "The installation path of the app",
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
				MarkdownDescription: "The dependencies of the app",
				Optional:            true,
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
			},
		},
	}
}

func (r *Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = providerutil.ClientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	data := ResourceModel{}

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	appClient := r.client.App()
	appInput, appUpdaters := data.ToCreateRequestWithUpdaters(ctx, resp.Diagnostics, appClient)

	if resp.Diagnostics.HasError() {
		return
	}

	installationID := providerutil.
		Try[string](&resp.Diagnostics, "error while requesting app installation").
		DoVal(appClient.RequestAppInstallation(ctx, data.ProjectID.ValueString(), appInput))

	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(installationID)

	try := providerutil.Try[any](&resp.Diagnostics, "error while updating app installation")

	if len(appUpdaters) > 0 {
		try.Do(appClient.UpdateAppInstallation(ctx, installationID, appUpdaters...))
	}

	if !data.DatabaseID.IsNull() {
		try.Do(appClient.LinkAppInstallationToDatabase(
			ctx,
			data.ID.ValueString(),
			data.DatabaseID.ValueString(),
			mittwaldv2.AppLinkDatabaseJSONBodyPurposePrimary,
		))
	}

	try.Do(appClient.WaitUntilAppInstallationIsReady(ctx, installationID))

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
		Try[*mittwaldv2.DeMittwaldV1AppAppInstallation](&res, "error while fetching app installation").
		DoVal(appClient.GetAppInstallation(ctx, data.ID.ValueString()))

	if res.HasError() {
		return
	}

	res.Append(data.FromAPIModel(ctx, appInstallation, appClient)...)

	return
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	planData := ResourceModel{}
	currentData := ResourceModel{}

	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &currentData)...)

	appClient := r.client.App()
	updaters := planData.ToUpdateUpdaters(ctx, resp.Diagnostics, &currentData, appClient)

	try := providerutil.Try[any](&resp.Diagnostics, "error while updating app installation")

	if len(updaters) > 0 {
		try.Do(appClient.UpdateAppInstallation(ctx, planData.ID.ValueString(), updaters...))
	}

	if !planData.DatabaseID.Equal(currentData.DatabaseID) {
		try.Do(appClient.LinkAppInstallationToDatabase(
			ctx,
			planData.ID.ValueString(),
			planData.DatabaseID.ValueString(),
			mittwaldv2.AppLinkDatabaseJSONBodyPurposePrimary,
		))
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
		Do(r.client.App().UninstallApp(ctx, data.ID.ValueString()))
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
