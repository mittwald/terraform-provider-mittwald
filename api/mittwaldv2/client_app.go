package mittwaldv2

import (
	"context"
	"github.com/google/uuid"
)

type AppClient interface {
	GetAppVersion(ctx context.Context, appID string, versionID string) (*DeMittwaldV1AppAppVersion, error)
	ListAppVersions(ctx context.Context, appID string) ([]DeMittwaldV1AppAppVersion, error)
	ListApps(ctx context.Context) ([]DeMittwaldV1AppApp, error)
	RequestAppInstallation(ctx context.Context, projectID string, body AppRequestAppinstallationJSONRequestBody) (string, error)
	GetAppInstallation(ctx context.Context, appInstallationID string) (*DeMittwaldV1AppAppInstallation, error)
	WaitUntilAppInstallationIsReady(ctx context.Context, appID string) error
	UninstallApp(ctx context.Context, appInstallationID string) error
	LinkAppInstallationToDatabase(ctx context.Context, appInstallationID string, databaseID string, purpose AppLinkDatabaseJSONBodyPurpose) error
	GetSystemSoftwareByName(ctx context.Context, name string) (*DeMittwaldV1AppSystemSoftware, bool, error)
	SelectSystemSoftwareVersion(ctx context.Context, systemSoftwareID, versionSelector string) (DeMittwaldV1AppSystemSoftwareVersionSet, error)
	GetSystemSoftwareAndVersion(ctx context.Context, systemSoftwareID, systemSoftwareVersionID string) (*DeMittwaldV1AppSystemSoftware, *DeMittwaldV1AppSystemSoftwareVersion, error)
	UpdateAppInstallation(ctx context.Context, appInstallationID string, updater ...AppInstallationUpdater) error
}

type appClient struct {
	client ClientWithResponsesInterface
}

func (c *appClient) GetAppVersion(ctx context.Context, appID string, versionID string) (*DeMittwaldV1AppAppVersion, error) {
	response, err := c.client.AppGetAppversionWithResponse(ctx, uuid.MustParse(appID), uuid.MustParse(versionID))
	if err != nil {
		return nil, err
	}

	if response.JSON200 != nil {
		return response.JSON200, nil
	}

	return nil, errUnexpectedStatus(response.StatusCode(), response.Body)
}

func (c *appClient) ListAppVersions(ctx context.Context, appID string) ([]DeMittwaldV1AppAppVersion, error) {
	response, err := c.client.AppListAppversionsWithResponse(ctx, uuid.MustParse(appID))
	if err != nil {
		return nil, err
	}

	if response.JSON200 != nil {
		return *response.JSON200, nil
	}

	return nil, errUnexpectedStatus(response.StatusCode(), response.Body)
}

func (c *appClient) ListApps(ctx context.Context) ([]DeMittwaldV1AppApp, error) {
	response, err := c.client.AppListAppsWithResponse(ctx, &AppListAppsParams{})
	if err != nil {
		return nil, err
	}

	if response.JSON200 != nil {
		return *response.JSON200, nil
	}

	return nil, errUnexpectedStatus(response.StatusCode(), response.Body)
}
