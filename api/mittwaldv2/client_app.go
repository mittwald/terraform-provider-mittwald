package mittwaldv2

import (
	"context"
	"errors"
	"github.com/google/uuid"
)

type AppClient struct {
	client ClientWithResponsesInterface
}

func (c *AppClient) RequestAppInstallation(ctx context.Context, projectID string, body AppRequestAppinstallationJSONRequestBody) (string, error) {
	response, err := c.client.AppRequestAppinstallationWithResponse(ctx, uuid.MustParse(projectID), body)
	if err != nil {
		return "", err
	}

	if response.JSON201 != nil {
		return response.JSON201.Id.String(), nil
	}

	return "", errUnexpectedStatus(response.StatusCode(), response.Body)
}

func (c *AppClient) GetAppInstallation(ctx context.Context, appInstallationID string) (*DeMittwaldV1AppAppInstallation, error) {
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

func (c *AppClient) WaitUntilAppInstallationIsReady(ctx context.Context, appID string) error {
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

	ready := make(chan bool)
	errs := make(chan error)

	defer close(ready)
	defer close(errs)

	go func() {
		for {
			r, err := runner()
			if err != nil {
				if notFound := (ErrNotFound{}); errors.As(err, &notFound) {
					continue
				}
				errs <- err
				return
			}

			if r {
				ready <- r
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errs:
		return err
	case <-ready:
		return nil
	}
}

func (c *AppClient) UninstallApp(ctx context.Context, appInstallationID string) error {
	response, err := c.client.AppUninstallAppinstallationWithResponse(ctx, uuid.MustParse(appInstallationID))
	if err != nil {
		return err
	}

	if response.StatusCode() >= 200 && response.StatusCode() < 300 {
		return nil
	}

	return errUnexpectedStatus(response.StatusCode(), response.Body)
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
