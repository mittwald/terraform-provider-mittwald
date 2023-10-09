package apps

import (
	"context"
	"fmt"
	"github.com/mittwald/terraform-provider-mittwald/internal/mittwaldv2"
	appsv2 "github.com/mittwald/terraform-provider-mittwald/internal/mittwaldv2/models/apps"
	"time"
)

type Client struct {
	client *mittwaldv2.Client
}

func NewClient(c *mittwaldv2.Client) *Client {
	return &Client{client: c}
}

func (c *Client) CreateAppInstallation(ctx context.Context, projectID string, input *appsv2.CreateAppInstallationRequest) (*appsv2.AppInstallation, error) {
	var appInstallationResponse appsv2.CreateAppInstallationResponse
	var appInstallation appsv2.AppInstallation

	appInstallationsURL := fmt.Sprintf("/projects/%s/appinstallations", projectID)

	if err := c.client.Post(ctx, appInstallationsURL, input, &appInstallationResponse); err != nil {
		return nil, err
	}

	appInstallationURL := fmt.Sprintf("/appinstallations/%s", appInstallationResponse.ID)

	if err := c.client.Poll(ctx, appInstallationURL, &appInstallation); err != nil {
		return nil, err
	}

	return &appInstallation, nil
}

func (c *Client) WaitUntilAppInstallationIsReady(ctx context.Context, appInstallationID string) error {
	appInstallation := appsv2.AppInstallation{}
	appInstallationURL := fmt.Sprintf("/appinstallations/%s", appInstallationID)
	appReady := make(chan error)
	appTicker := time.NewTicker(500 * time.Millisecond)

	defer close(appReady)
	defer appTicker.Stop()

	go func() {
		for range appTicker.C {
			if err := c.client.Get(ctx, appInstallationURL, &appInstallation); err != nil {
				appReady <- err
				return
			}

			if appInstallation.AppVersion.Desired == appInstallation.AppVersion.Current {
				appReady <- nil
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-appReady:
		return err
	}
}

func (c *Client) GetApp(ctx context.Context, id string) (*appsv2.App, error) {
	var app appsv2.App

	if err := c.client.Get(ctx, "/apps/"+id, &app); err != nil {
		return nil, err
	}

	return &app, nil
}

func (c *Client) GetAppVersion(ctx context.Context, appID, versionID string) (*appsv2.AppVersion, error) {
	appVersion := appsv2.AppVersion{}
	appVersionURL := fmt.Sprintf("/apps/%s/versions/%s", appID, versionID)

	if err := c.client.Get(ctx, appVersionURL, &appVersion); err != nil {
		return nil, err
	}

	return &appVersion, nil
}

func (c *Client) GetAppVersions(ctx context.Context, appID string) ([]appsv2.AppVersion, error) {
	appVersions := make([]appsv2.AppVersion, 0)
	appVersionsURL := fmt.Sprintf("/apps/%s/versions", appID)

	if err := c.client.Get(ctx, appVersionsURL, &appVersions); err != nil {
		return nil, err
	}

	return appVersions, nil
}

func (c *Client) GetAppInstallation(ctx context.Context, id string) (*appsv2.AppInstallation, error) {
	var appInstallation appsv2.AppInstallation

	if err := c.client.Poll(ctx, "/appinstallations/"+id, &appInstallation); err != nil {
		return nil, err
	}

	return &appInstallation, nil
}

func (c *Client) DeleteAppInstallation(ctx context.Context, id string) error {
	return c.client.Delete(ctx, "/appinstallations/"+id)
}
