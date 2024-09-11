package mittwaldv2

import (
	"context"
	"errors"
	"github.com/google/uuid"
)

func (c *appClient) RequestAppInstallation(ctx context.Context, projectID string, body AppRequestAppinstallationJSONRequestBody) (string, error) {
	response, err := c.client.AppRequestAppinstallationWithResponse(ctx, uuid.MustParse(projectID), body)
	if err != nil {
		return "", err
	}

	if response.JSON201 != nil {
		return response.JSON201.Id.String(), nil
	}

	return "", errUnexpectedStatus(response.StatusCode(), response.Body)
}

func (c *appClient) GetAppInstallation(ctx context.Context, appInstallationID string) (*DeMittwaldV1AppAppInstallation, error) {
	return poll(ctx, func() (*DeMittwaldV1AppAppInstallation, error) {
		response, err := c.client.AppGetAppinstallationWithResponse(ctx, uuid.MustParse(appInstallationID))
		if err != nil {
			return nil, err
		}

		if response.JSON200 != nil {
			return response.JSON200, nil
		}

		return nil, errUnexpectedStatus(response.StatusCode(), response.Body)
	})
}

func (c *appClient) WaitUntilAppInstallationIsReady(ctx context.Context, appID string) error {
	runner := func() (bool, error) {
		response, err := c.client.AppGetAppinstallationWithResponse(ctx, uuid.MustParse(appID))
		if err != nil {
			return false, err
		}

		if response.JSON200 == nil {
			return false, errUnexpectedStatus(response.StatusCode(), response.Body)
		}

		if response.JSON200.AppVersion.Current == nil {
			return false, nil
		}

		if *response.JSON200.AppVersion.Current != response.JSON200.AppVersion.Desired {
			return false, nil
		}

		return true, nil
	}

	if ready, err := poll(ctx, runner); err != nil {
		return err
	} else if !ready {
		return errors.New("app installation is not ready")
	}

	return nil
}

func (c *appClient) UninstallApp(ctx context.Context, appInstallationID string) error {
	response, err := c.client.AppUninstallAppinstallationWithResponse(ctx, uuid.MustParse(appInstallationID))
	if err != nil {
		return err
	}

	if response.StatusCode() >= 200 && response.StatusCode() < 300 {
		return nil
	}

	return errUnexpectedStatus(response.StatusCode(), response.Body)
}

func (c *appClient) LinkAppInstallationToDatabase(
	ctx context.Context,
	appInstallationID string,
	databaseID string,
	userID string,
	purpose AppLinkDatabaseJSONBodyPurpose,
) error {
	userIDs := map[string]string{
		"admin": userID,
	}

	response, err := c.client.AppLinkDatabaseWithResponse(ctx, uuid.MustParse(appInstallationID), AppLinkDatabaseJSONRequestBody{
		DatabaseId:      uuid.MustParse(databaseID),
		Purpose:         purpose,
		DatabaseUserIds: &userIDs,
	})
	if err != nil {
		return err
	}

	if response.StatusCode() >= 200 && response.StatusCode() < 300 {
		return nil
	}

	return errUnexpectedStatus(response.StatusCode(), response.Body)
}

func (c *appClient) UnlinkAppInstallationFromDatabase(ctx context.Context, appInstallationID string, databaseID string) error {
	resp, err := c.client.AppUnlinkDatabaseWithResponse(ctx, uuid.MustParse(appInstallationID), uuid.MustParse(databaseID))
	if err != nil {
		return err
	}

	if resp.StatusCode() >= 200 && resp.StatusCode() < 300 {
		return nil
	}

	return errUnexpectedStatus(resp.StatusCode(), resp.Body)
}
