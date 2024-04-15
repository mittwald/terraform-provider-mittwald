package appresource

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/mittwald/terraform-provider-mittwald/api/mittwaldv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
	"github.com/mittwald/terraform-provider-mittwald/internal/valueutil"
)

func (m *ResourceModel) ToCreateRequestWithUpdaters(ctx context.Context, d diag.Diagnostics, appClient mittwaldv2.AppClient) (mittwaldv2.AppRequestAppinstallationJSONRequestBody, []mittwaldv2.AppInstallationUpdater) {
	return m.ToCreateRequest(ctx, d, appClient), m.ToCreateUpdaters(ctx, d, appClient)
}

func (m *ResourceModel) ToCreateUpdaters(ctx context.Context, d diag.Diagnostics, appClient mittwaldv2.AppClient) []mittwaldv2.AppInstallationUpdater {
	updaters := make([]mittwaldv2.AppInstallationUpdater, 0)

	if !m.DocumentRoot.IsNull() {
		updaters = append(updaters, mittwaldv2.UpdateAppInstallationDocumentRoot(m.DocumentRoot.ValueString()))
	}

	if !m.UpdatePolicy.IsNull() {
		updaters = append(updaters, mittwaldv2.UpdateAppInstallationUpdatePolicy(mittwaldv2.DeMittwaldV1AppAppUpdatePolicy(m.UpdatePolicy.ValueString())))
	}

	if !m.Dependencies.IsNull() {
		depUpdater := providerutil.
			Try[mittwaldv2.AppInstallationUpdater](&d, "error while building dependency updaters").
			DoVal(m.dependenciesToUpdater(ctx, appClient, nil))
		updaters = append(updaters, depUpdater)
	}

	return updaters
}

func (m *ResourceModel) ToCreateRequest(ctx context.Context, d diag.Diagnostics, appClient mittwaldv2.AppClient) (b mittwaldv2.AppRequestAppinstallationJSONRequestBody) {
	appID, ok := appNames[m.App.ValueString()]
	if !ok {
		d.AddError("app", "App not found")
		return
	}

	b.Description = m.Description.ValueString()
	b.UpdatePolicy = mittwaldv2.DeMittwaldV1AppAppUpdatePolicy(m.UpdatePolicy.ValueString())

	appVersions := providerutil.
		Try[[]mittwaldv2.DeMittwaldV1AppAppVersion](&d, "error while listing app versions").
		DoVal(appClient.ListAppVersions(ctx, appID))

	for _, appVersion := range appVersions {
		if appVersion.InternalVersion == m.Version.ValueString() {
			b.AppVersionId = appVersion.Id
		}
	}

	for key, value := range m.UserInputs.Elements() {
		if s, ok := value.(types.String); ok {
			b.UserInputs = append(b.UserInputs, mittwaldv2.DeMittwaldV1AppSavedUserInput{
				Name:  key,
				Value: s.ValueString(),
			})
		} else {
			d.AddAttributeError(path.Root("user_inputs").AtMapKey(key), "invalid type", fmt.Sprintf("expected string, got %T", value))
		}
	}

	return
}

func (m *ResourceModel) ToUpdateUpdaters(ctx context.Context, d diag.Diagnostics, current *ResourceModel, appClient mittwaldv2.AppClient) []mittwaldv2.AppInstallationUpdater {
	updaters := make([]mittwaldv2.AppInstallationUpdater, 0)

	if !m.Description.Equal(current.Description) {
		updaters = append(updaters, mittwaldv2.UpdateAppInstallationDescription(m.Description.ValueString()))
	}

	if !m.DocumentRoot.Equal(current.DocumentRoot) {
		updaters = append(updaters, mittwaldv2.UpdateAppInstallationDocumentRoot(m.DocumentRoot.ValueString()))
	}

	if !m.UpdatePolicy.Equal(current.UpdatePolicy) {
		updaters = append(updaters, mittwaldv2.UpdateAppInstallationUpdatePolicy(mittwaldv2.DeMittwaldV1AppAppUpdatePolicy(m.UpdatePolicy.ValueString())))
	}

	if !m.Dependencies.Equal(current.Dependencies) {
		depUpdater := providerutil.
			Try[mittwaldv2.AppInstallationUpdater](&d, "error while building dependency updaters").
			DoVal(m.dependenciesToUpdater(ctx, appClient, &current.Dependencies))
		updaters = append(updaters, depUpdater)
	}

	return updaters
}

func (m *ResourceModel) FromAPIModel(ctx context.Context, appInstallation *mittwaldv2.DeMittwaldV1AppAppInstallation, clientBuilder mittwaldv2.ClientBuilder) (res diag.Diagnostics) {
	appClient := clientBuilder.App()
	projectClient := clientBuilder.Project()

	appDesiredVersion := providerutil.
		Try[*mittwaldv2.DeMittwaldV1AppAppVersion](&res, "error while fetching app version").
		DoVal(appClient.GetAppVersion(ctx, appInstallation.AppId.String(), appInstallation.AppVersion.Desired))

	project := providerutil.
		Try[*mittwaldv2.DeMittwaldV1ProjectProject](&res, "error while fetching project").
		DoVal(projectClient.GetProject(ctx, appInstallation.ProjectId.String()))

	if res.HasError() {
		return
	}

	m.ShortID = types.StringValue(appInstallation.ShortId)
	m.ProjectID = types.StringValue(appInstallation.ProjectId.String())
	m.InstallationPath = types.StringValue(appInstallation.InstallationPath)
	m.InstallationPathAbsolute = types.StringValue(project.Directories["Web"] + appInstallation.InstallationPath)
	m.App = func() types.String {
		for key, appID := range appNames {
			if appID == appInstallation.AppId.String() {
				return types.StringValue(key)
			}
		}
		return types.StringNull()
	}()

	m.DocumentRoot = valueutil.StringPtrOrNull(appInstallation.CustomDocumentRoot)
	m.Description = valueutil.StringOrNull(appInstallation.Description)
	m.Version = types.StringValue(appDesiredVersion.InternalVersion)
	m.UpdatePolicy = valueutil.StringPtrOrNull(appInstallation.UpdatePolicy)

	if appInstallation.LinkedDatabases != nil {
		databases := make([]DatabaseModel, 0)
		for _, db := range *appInstallation.LinkedDatabases {
			model := DatabaseModel{
				ID:      types.StringValue(db.DatabaseId.String()),
				Kind:    types.StringValue(string(db.Kind)),
				Purpose: types.StringValue(string(db.Purpose)),
			}

			if db.DatabaseUserIds != nil {
				userID, ok := (*db.DatabaseUserIds)["admin"]
				if ok {
					model.UserID = types.StringValue(userID)
				}
			}

			databases = append(databases, model)
		}

		databaseModels, d := types.SetValueFrom(ctx, &databaseModelAttrType, databases)
		res.Append(d...)

		m.Databases = databaseModels
	} else {
		m.Databases = types.SetNull(&databaseModelAttrType)
	}

	if appInstallation.AppVersion.Current != nil {
		appCurrentVersion := providerutil.
			Try[*mittwaldv2.DeMittwaldV1AppAppVersion](&res, "error while fetching app version").
			DoVal(appClient.GetAppVersion(ctx, appInstallation.AppId.String(), *appInstallation.AppVersion.Current))
		if appCurrentVersion != nil {
			m.VersionCurrent = types.StringValue(appCurrentVersion.InternalVersion)
		}
	}

	if appInstallation.SystemSoftware != nil {
		m.Dependencies = InstalledSystemSoftwareToDependencyModelMap(ctx, res, appClient, *appInstallation.SystemSoftware)
	}

	return
}

func (m *ResourceModel) dependenciesToUpdater(ctx context.Context, appClient mittwaldv2.AppClient, currentDependencies *types.Map) (mittwaldv2.AppInstallationUpdater, error) {
	updater := make(mittwaldv2.AppInstallationUpdaterChain, 0)
	seen := make(map[string]struct{})

	for name, options := range m.Dependencies.Elements() {
		seen[name] = struct{}{}

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

		versions, err := appClient.SelectSystemSoftwareVersion(ctx, dependency.Id, optionsModel.Version.ValueString())
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
				mittwaldv2.DeMittwaldV1AppSystemSoftwareUpdatePolicy(optionsModel.UpdatePolicy.ValueString()),
			),
		)
	}

	if currentDependencies != nil {
		for name := range currentDependencies.Elements() {
			if _, ok := seen[name]; !ok {
				dependency, ok, err := appClient.GetSystemSoftwareByName(ctx, name)
				if err != nil {
					return nil, err
				} else if !ok {
					return nil, fmt.Errorf("dependency %s not found", name)
				}

				updater = append(updater, mittwaldv2.RemoveAppInstallationSystemSoftware(dependency.Id))
			}
		}
	}

	return updater, nil
}
