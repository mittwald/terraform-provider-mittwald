package mittwaldv2

import (
	"context"
	"github.com/google/uuid"
)

type AppClient struct {
	client ClientWithResponsesInterface
}

func (c *AppClient) GetAppVersion(ctx context.Context, appID string, versionID string) (*DeMittwaldV1AppAppVersion, error) {
	response, err := c.client.AppGetAppversionWithResponse(ctx, uuid.MustParse(appID), uuid.MustParse(versionID))
	if err != nil {
		return nil, err
	}

	if response.JSON200 != nil {
		return response.JSON200, nil
	}

	return nil, errUnexpectedStatus(response.StatusCode(), response.Body)
}

func (c *AppClient) ListAppVersions(ctx context.Context, appID string) ([]DeMittwaldV1AppAppVersion, error) {
	response, err := c.client.AppListAppversionsWithResponse(ctx, uuid.MustParse(appID))
	if err != nil {
		return nil, err
	}

	if response.JSON200 != nil {
		return *response.JSON200, nil
	}

	return nil, errUnexpectedStatus(response.StatusCode(), response.Body)
}

func (c *AppClient) ListApps(ctx context.Context) ([]DeMittwaldV1AppApp, error) {
	response, err := c.client.AppListAppsWithResponse(ctx, &AppListAppsParams{})
	if err != nil {
		return nil, err
	}

	if response.JSON200 != nil {
		return *response.JSON200, nil
	}

	return nil, errUnexpectedStatus(response.StatusCode(), response.Body)
}
