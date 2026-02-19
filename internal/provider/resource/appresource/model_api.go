package appresource

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/appclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/projectclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/appv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/projectv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiext"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
	"github.com/mittwald/terraform-provider-mittwald/internal/valueutil"
)

func (m *ResourceModel) ToCreateRequestWithUpdaters(ctx context.Context, d *diag.Diagnostics, appClient apiext.AppClient) (appclientv2.RequestAppinstallationRequest, []apiext.AppInstallationUpdater) {
	return m.ToCreateRequest(ctx, d, appClient), m.ToCreateUpdaters(ctx, d, appClient)
}

func (m *ResourceModel) ToCreateUpdaters(ctx context.Context, d *diag.Diagnostics, appClient apiext.AppClient) []apiext.AppInstallationUpdater {
	updaters := make([]apiext.AppInstallationUpdater, 0)

	if !m.DocumentRoot.IsNull() {
		updaters = append(updaters, apiext.UpdateAppInstallationDocumentRoot(m.DocumentRoot.ValueString()))
	}

	if !m.UpdatePolicy.IsNull() {
		updaters = append(updaters, apiext.UpdateAppInstallationUpdatePolicy(appv2.AppUpdatePolicy(m.UpdatePolicy.ValueString())))
	}

	if !m.Dependencies.IsNull() {
		depUpdater := providerutil.
			Try[apiext.AppInstallationUpdater](d, "error while building dependency updaters").
			DoVal(m.dependenciesToUpdater(ctx, appClient, nil))
		updaters = append(updaters, depUpdater)
	}

	return updaters
}

func (m *ResourceModel) ToCreateRequest(ctx context.Context, d *diag.Diagnostics, appClient appclientv2.Client) (r appclientv2.RequestAppinstallationRequest) {
	tflog.Debug(ctx, "building create request for app", map[string]any{"app": m.App.ValueString()})

	appID, ok := apiext.AppNames[m.App.ValueString()]
	if !ok {
		d.AddAttributeError(path.Root("app"), "invalid value", fmt.Sprintf("app %s not found", m.App.ValueString()))
		return
	}

	r.ProjectID = m.ProjectID.ValueString()

	b := &r.Body
	b.Description = m.Description.ValueString()
	b.UpdatePolicy = appv2.AppUpdatePolicy(m.UpdatePolicy.ValueString())

	appVersions := providerutil.
		Try[*[]appv2.AppVersion](d, "error while listing app versions").
		DoValResp(appClient.ListAppversions(ctx, appclientv2.ListAppversionsRequest{AppID: appID}))

	for _, appVersion := range *appVersions {
		if appVersion.InternalVersion == m.Version.ValueString() {
			b.AppVersionId = appVersion.Id
		}
	}

	for key, value := range m.UserInputs.Elements() {
		if s, ok := value.(types.String); ok {
			b.UserInputs = append(b.UserInputs, appv2.SavedUserInput{
				Name:  key,
				Value: s.ValueString(),
			})
		} else {
			d.AddAttributeError(path.Root("user_inputs").AtMapKey(key), "invalid type", fmt.Sprintf("expected string, got %T", value))
		}
	}

	return
}

func (m *ResourceModel) ToUpdateUpdaters(ctx context.Context, d diag.Diagnostics, current *ResourceModel, appClient apiext.AppClient) []apiext.AppInstallationUpdater {
	updaters := make([]apiext.AppInstallationUpdater, 0)

	if !m.Description.Equal(current.Description) {
		updaters = append(updaters, apiext.UpdateAppInstallationDescription(m.Description.ValueString()))
	}

	if !m.DocumentRoot.Equal(current.DocumentRoot) {
		updaters = append(updaters, apiext.UpdateAppInstallationDocumentRoot(m.DocumentRoot.ValueString()))
	}

	if !m.UpdatePolicy.Equal(current.UpdatePolicy) {
		updaters = append(updaters, apiext.UpdateAppInstallationUpdatePolicy(appv2.AppUpdatePolicy(m.UpdatePolicy.ValueString())))
	}

	if !m.Dependencies.Equal(current.Dependencies) {
		depUpdater := providerutil.
			Try[apiext.AppInstallationUpdater](&d, "error while building dependency updaters").
			DoVal(m.dependenciesToUpdater(ctx, appClient, &current.Dependencies))
		updaters = append(updaters, depUpdater)
	}

	return updaters
}

func (m *ResourceModel) ToDeleteRequest() appclientv2.UninstallAppinstallationRequest {
	return appclientv2.UninstallAppinstallationRequest{
		AppInstallationID: m.ID.ValueString(),
	}
}

func (m *ResourceModel) FromAPIModel(ctx context.Context, appInstallation *appv2.AppInstallation, client mittwaldv2.Client) (res diag.Diagnostics) {
	appClient := apiext.NewAppClient(client)
	projectClient := client.Project()

	appDesiredVersion := providerutil.
		Try[*appv2.AppVersion](&res, "error while fetching app version").
		DoValResp(appClient.GetAppversion(ctx, appclientv2.GetAppversionRequest{AppID: appInstallation.AppId, AppVersionID: appInstallation.AppVersion.Desired}))

	project := providerutil.
		Try[*projectv2.Project](&res, "error while fetching project").
		DoValResp(projectClient.GetProject(ctx, projectclientv2.GetProjectRequest{ProjectID: appInstallation.ProjectId}))

	if res.HasError() {
		return
	}

	m.ShortID = types.StringValue(appInstallation.ShortId)
	m.ProjectID = types.StringValue(appInstallation.ProjectId)
	m.InstallationPath = types.StringValue(appInstallation.InstallationPath)
	m.InstallationPathAbsolute = types.StringValue(project.Directories["Web"] + appInstallation.InstallationPath)

	if project.ClusterID != nil && project.ClusterDomain != nil {
		m.SSHHost = types.StringValue(fmt.Sprintf("ssh.%s.%s", *project.ClusterID, *project.ClusterDomain))
	} else {
		m.SSHHost = types.StringNull()
	}

	m.App = func() types.String {
		for key, appID := range apiext.AppNames {
			if appID == appInstallation.AppId {
				return types.StringValue(key)
			}
		}
		return types.StringNull()
	}()

	m.DocumentRoot = valueutil.StringPtrOrNull(appInstallation.CustomDocumentRoot)
	m.Description = valueutil.StringOrNull(appInstallation.Description)
	m.Version = types.StringValue(appDesiredVersion.InternalVersion)
	m.UpdatePolicy = types.StringValue(string(appInstallation.UpdatePolicy))

	if appInstallation.LinkedDatabases != nil {
		databases := make([]DatabaseModel, 0)
		for _, db := range appInstallation.LinkedDatabases {
			model := DatabaseModel{
				ID:      types.StringValue(db.DatabaseId),
				Kind:    types.StringValue(string(db.Kind)),
				Purpose: types.StringValue(string(db.Purpose)),
			}

			if db.DatabaseUserIds != nil {
				userID, ok := db.DatabaseUserIds["admin"]
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
		m.Databases, _ = types.SetValue(&databaseModelAttrType, nil)
	}

	if appInstallation.AppVersion.Current != nil {
		appCurrentVersion := providerutil.
			Try[*appv2.AppVersion](&res, "error while fetching app version").
			DoValResp(appClient.GetAppversion(ctx, appclientv2.GetAppversionRequest{AppID: appInstallation.AppId, AppVersionID: *appInstallation.AppVersion.Current}))
		if appCurrentVersion != nil {
			m.VersionCurrent = types.StringValue(appCurrentVersion.InternalVersion)
		}
	}

	if appInstallation.SystemSoftware != nil {
		m.Dependencies = InstalledSystemSoftwareToDependencyModelMap(ctx, res, appClient, appInstallation.SystemSoftware)
	}

	return
}

func (m *ResourceModel) dependenciesToUpdater(ctx context.Context, appClient apiext.AppClient, currentDependencies *types.Map) (apiext.AppInstallationUpdater, error) {
	updater := make(apiext.AppInstallationUpdaterChain, 0)
	seen := make(map[string]struct{})

	for name, options := range m.Dependencies.Elements() {
		seen[name] = struct{}{}

		dependency, ok, err := appClient.GetSystemsoftwareByName(ctx, name)
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

		versions, err := appClient.SelectSystemsoftwareVersion(ctx, dependency.Id, optionsModel.Version.ValueString())
		if err != nil {
			return nil, err
		}

		recommended, ok := versions.Recommended()
		if !ok {
			return nil, fmt.Errorf("no recommended version found for %s", name)
		}

		updater = append(
			updater,
			apiext.UpdateAppInstallationSystemSoftware(
				dependency.Id,
				recommended.Id,
				appv2.SystemSoftwareUpdatePolicy(optionsModel.UpdatePolicy.ValueString()),
			),
		)
	}

	if currentDependencies != nil {
		for name := range currentDependencies.Elements() {
			if _, ok := seen[name]; !ok {
				dependency, ok, err := appClient.GetSystemsoftwareByName(ctx, name)
				if err != nil {
					return nil, err
				} else if !ok {
					return nil, fmt.Errorf("dependency %s not found", name)
				}

				updater = append(updater, apiext.RemoveAppInstallationSystemSoftware(dependency.Id))
			}
		}
	}

	return updater, nil
}
