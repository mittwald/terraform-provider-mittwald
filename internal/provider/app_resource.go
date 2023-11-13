package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/mittwald/terraform-provider-mittwald/api/mittwaldv2"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &AppResource{}
var _ resource.ResourceWithImportState = &AppResource{}

var appNames = map[string]string{
	"static":    "d20baefd-81d2-42aa-bfba-9a3220ae839b",
	"php":       "34220303-cb87-4592-8a95-2eb20a97b2ac",
	"node":      "3e7f920b-a711-4d2f-9871-661e1b41a2f0",
	"wordpress": "da3aa3ae-4b6b-4398-a4a8-ee8def827876",
	"typo3":     "352971cc-b96a-4a26-8651-b08d7c8a7357",
	"shopware6": "12d54d05-7e55-4cf3-90c4-093516e0eaf8",
	"shopware5": "a23acf9c-9298-4082-9e7d-25356f9976dc",
}

func NewAppResource() resource.Resource {
	return &AppResource{}
}

type AppResource struct {
	client mittwaldv2.ClientBuilder
}

type AppResourceModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	DatabaseID       types.String `tfsdk:"database_id"` // TODO: There may theoretically be multiple database links
	Description      types.String `tfsdk:"description"`
	App              types.String `tfsdk:"app"`
	Version          types.String `tfsdk:"version"`
	VersionCurrent   types.String `tfsdk:"version_current"`
	DocumentRoot     types.String `tfsdk:"document_root"`
	InstallationPath types.String `tfsdk:"installation_path"`
	UpdatePolicy     types.String `tfsdk:"update_policy"`
	UserInputs       types.Map    `tfsdk:"user_inputs"`
	Dependencies     types.Map    `tfsdk:"dependencies"`
}

type DependencyModel struct {
	Version string `tfsdk:"version"`
	//VersionCurrent string `tfsdk:"version_current"`
	UpdatePolicy string `tfsdk:"update_policy"`
}

func (r *AppResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app"
}

func (r *AppResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
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

func (r *AppResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFromProviderData(&req, resp)
}

func (r *AppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	data := AppResourceModel{}

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	appID, ok := appNames[data.App.ValueString()]
	if !ok {
		resp.Diagnostics.AddError("app", "App not found")
		return
	}

	appClient := r.client.App()
	appInput := mittwaldv2.AppRequestAppinstallationJSONRequestBody{
		Description:  data.Description.ValueString(),
		UpdatePolicy: mittwaldv2.DeMittwaldV1AppAppUpdatePolicy(data.UpdatePolicy.ValueString()),
	}

	appVersions := ErrorValueToDiag(appClient.ListAppVersions(ctx, appID))(&resp.Diagnostics, "API Error")
	for _, appVersion := range appVersions {
		if appVersion.InternalVersion == data.Version.ValueString() {
			appInput.AppVersionId = appVersion.Id
		}
	}

	for key, value := range data.UserInputs.Elements() {
		appInput.UserInputs = append(appInput.UserInputs, mittwaldv2.DeMittwaldV1AppSavedUserInput{
			Name:  key,
			Value: value.String(),
		})
	}

	if resp.Diagnostics.HasError() {
		return
	}

	appID, err := appClient.RequestAppInstallation(ctx, data.ProjectID.ValueString(), appInput)
	if err != nil {
		resp.Diagnostics.AddError("API Error", err.Error())
		return
	}

	data.ID = types.StringValue(appID)

	updaters := make([]mittwaldv2.AppInstallationUpdater, 0)

	if !data.DocumentRoot.IsNull() {
		updaters = append(updaters, mittwaldv2.UpdateAppInstallationDocumentRoot(data.DocumentRoot.ValueString()))
	}

	if !data.UpdatePolicy.IsNull() {
		updaters = append(updaters, mittwaldv2.UpdateAppInstallationUpdatePolicy(mittwaldv2.DeMittwaldV1AppAppUpdatePolicy(data.UpdatePolicy.ValueString())))
	}

	if !data.Dependencies.IsNull() {
		depUpdater := ErrorValueToDiag(r.appDependenciesToUpdater(ctx, &data))(&resp.Diagnostics, "Dependency version error")
		updaters = append(updaters, depUpdater)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	if len(updaters) > 0 {
		ErrorToDiag(appClient.UpdateAppInstallation(ctx, data.ID.ValueString(), updaters...))(&resp.Diagnostics, "API Error")
	}

	if !data.DatabaseID.IsNull() {
		ErrorToDiag(appClient.LinkAppInstallationToDatabase(
			ctx,
			data.ID.ValueString(),
			data.DatabaseID.ValueString(),
			mittwaldv2.AppLinkDatabaseJSONBodyPurposePrimary,
		))(&resp.Diagnostics, "API Error")
	}

	ErrorToDiag(appClient.WaitUntilAppInstallationIsReady(ctx, appID))(&resp.Diagnostics, "API Error")

	resp.Diagnostics.Append(r.read(ctx, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppResource) appDependenciesToUpdater(ctx context.Context, d *AppResourceModel) (mittwaldv2.AppInstallationUpdater, error) {
	appClient := r.client.App()
	updater := make(mittwaldv2.AppInstallationUpdaterChain, 0)
	for name, options := range d.Dependencies.Elements() {
		dependency, ok, err := appClient.GetSystemSoftwareByName(ctx, name)
		if err != nil {
			return nil, err
		} else if !ok {
			return nil, fmt.Errorf("dependency %s not found", name)
		}

		optionsObj, ok := options.(types.Object)
		if !ok {
			return nil, fmt.Errorf("expected types.Object, got %T", options)
		}

		optionsModel := DependencyModel{}
		optionsObj.As(ctx, &optionsModel, basetypes.ObjectAsOptions{})

		versions, err := appClient.SelectSystemSoftwareVersion(ctx, dependency.Id, optionsModel.Version)
		if err != nil {
			return nil, err
		}

		recommended, ok := versions.Recommended()
		if !ok {
			return nil, fmt.Errorf("no recommended version found for %s", name)
		}

		updater = append(
			updater,
			mittwaldv2.UpdateAppInstallationSystemSoftware(
				dependency.Id,
				recommended.Id.String(),
				mittwaldv2.DeMittwaldV1AppSystemSoftwareUpdatePolicy(optionsModel.UpdatePolicy),
			),
		)
	}

	return updater, nil
}

func (r *AppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	data := AppResourceModel{}

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.read(ctx, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppResource) read(ctx context.Context, data *AppResourceModel) (res diag.Diagnostics) {
	appClient := r.client.App()

	appInstallation := ErrorValueToDiag(appClient.GetAppInstallation(ctx, data.ID.ValueString()))(&res, "API Error")
	if res.HasError() {
		return
	}

	appDesiredVersion := ErrorValueToDiag(appClient.GetAppVersion(ctx, appInstallation.AppId.String(), appInstallation.AppVersion.Desired))(&res, "API Error")
	if res.HasError() {
		return
	}

	data.ProjectID = types.StringValue(appInstallation.ProjectId.String())
	data.InstallationPath = types.StringValue(appInstallation.InstallationPath)
	data.App = func() types.String {
		for key, appID := range appNames {
			if appID == appInstallation.AppId.String() {
				return types.StringValue(key)
			}
		}
		return types.StringNull()
	}()

	if appInstallation.CustomDocumentRoot != nil {
		data.DocumentRoot = types.StringValue(*appInstallation.CustomDocumentRoot)
	} else {
		data.DocumentRoot = types.StringNull()
	}

	if appInstallation.Description != "" {
		data.Description = types.StringValue(appInstallation.Description)
	} else {
		data.Description = types.StringNull()
	}

	data.Version = types.StringValue(appDesiredVersion.InternalVersion)

	if appInstallation.UpdatePolicy != nil {
		data.UpdatePolicy = types.StringValue(string(*appInstallation.UpdatePolicy))
	} else {
		data.UpdatePolicy = types.StringNull()
	}

	data.DatabaseID = func() types.String {
		if appInstallation.LinkedDatabases == nil {
			return types.StringNull()
		}

		for _, link := range *appInstallation.LinkedDatabases {
			if link.Purpose == "primary" {
				return types.StringValue(link.DatabaseId.String())
			}
		}
		return types.StringNull()
	}()

	if appInstallation.AppVersion.Current != nil {
		if appDesiredVersion := ErrorValueToDiag(appClient.GetAppVersion(ctx, appInstallation.AppId.String(), appInstallation.AppVersion.Desired))(&res, "API Error"); appDesiredVersion != nil {
			data.VersionCurrent = types.StringValue(appDesiredVersion.InternalVersion)
		}
	}

	modType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"version":       types.StringType,
			"update_policy": types.StringType,
		},
	}

	if appInstallation.SystemSoftware != nil {
		dependencyMapValues := make(map[string]attr.Value)
		for _, dep := range *appInstallation.SystemSoftware {
			systemSoftware, version, err := appClient.GetSystemSoftwareAndVersion(
				ctx,
				dep.SystemSoftwareId.String(),
				dep.SystemSoftwareVersion.Desired,
			)

			if err != nil {
				ErrorToDiag(err)(&res, "API Error")
				return
			}

			mod := types.Object{}

			tfsdk.ValueFrom(ctx, DependencyModel{
				Version:      version.InternalVersion,
				UpdatePolicy: string(dep.UpdatePolicy),
			}, modType, &mod)

			dependencyMapValues[systemSoftware.Name] = mod
		}

		dependencyMap, d := basetypes.NewMapValue(modType, dependencyMapValues)
		if d.HasError() {
			res.Append(d...)
			return
		}

		data.Dependencies = dependencyMap
	}

	return
}

func (r *AppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	updaters := make([]mittwaldv2.AppInstallationUpdater, 0)
	planData := AppResourceModel{}
	currentData := AppResourceModel{}

	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &currentData)...)

	appClient := r.client.App()

	if !planData.DocumentRoot.Equal(currentData.DocumentRoot) {
		updaters = append(updaters, mittwaldv2.UpdateAppInstallationDocumentRoot(planData.DocumentRoot.ValueString()))
	}

	if !planData.UpdatePolicy.Equal(currentData.UpdatePolicy) {
		updaters = append(updaters, mittwaldv2.UpdateAppInstallationUpdatePolicy(mittwaldv2.DeMittwaldV1AppAppUpdatePolicy(planData.UpdatePolicy.ValueString())))
	}

	if len(updaters) > 0 {
		ErrorToDiag(appClient.UpdateAppInstallation(ctx, planData.ID.ValueString(), updaters...))(&resp.Diagnostics, "API Error")
	}

	if !planData.DatabaseID.Equal(currentData.DatabaseID) {
		ErrorToDiag(appClient.LinkAppInstallationToDatabase(
			ctx,
			planData.ID.ValueString(),
			planData.DatabaseID.ValueString(),
			mittwaldv2.AppLinkDatabaseJSONBodyPurposePrimary,
		))(&resp.Diagnostics, "API Error")
	}

	resp.Diagnostics.Append(r.read(ctx, &planData)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &planData)...)
}

func (r *AppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AppResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ErrorToDiag(r.client.App().UninstallApp(ctx, data.ID.ValueString()))(&resp.Diagnostics, "API Error")
}

func (r *AppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
