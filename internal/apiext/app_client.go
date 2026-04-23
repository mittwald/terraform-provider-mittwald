package apiext

import (
	"context"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/appclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/appv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiutils"
	"time"
)

type AppClient interface {
	appclientv2.Client
	GetSystemsoftwareByName(ctx context.Context, name string) (*appv2.SystemSoftware, bool, error)
	GetSystemsoftwareAndVersion(ctx context.Context, systemSoftwareID, systemSoftwareVersionID string) (*appv2.SystemSoftware, *appv2.SystemSoftwareVersion, error)
	SelectSystemsoftwareVersion(ctx context.Context, systemSoftwareID, versionSelector string) (SystemSoftwareVersionSet, error)
	UpdateAppinstallation(ctx context.Context, appInstallationID string, updater ...AppInstallationUpdater) error
	UpdateAppinstallationWithRetry(ctx context.Context, appInstallationID string, updater ...AppInstallationUpdater) error
	WaitUntilAppInstallationIsReady(ctx context.Context, appID string) error
	GetAppByName(ctx context.Context, name string) (*appv2.App, bool, error)
	SelectAppVersion(ctx context.Context, appID, versionSelector string) (AppVersionSet, error)
}

type appClient struct {
	appclientv2.Client
	clientSet mittwaldv2.Client
}

func NewAppClient(c mittwaldv2.Client) AppClient {
	return &appClient{
		Client:    c.App(),
		clientSet: c,
	}
}

type AppInstallationUpdater interface {
	Apply(b *appclientv2.PatchAppinstallationRequestBody)
}

type AppInstallationUpdaterFunc func(b *appclientv2.PatchAppinstallationRequestBody)
type AppInstallationUpdaterChain []AppInstallationUpdater

func (c AppInstallationUpdaterChain) Apply(b *appclientv2.PatchAppinstallationRequestBody) {
	for _, u := range c {
		u.Apply(b)
	}
}

func (f AppInstallationUpdaterFunc) Apply(b *appclientv2.PatchAppinstallationRequestBody) {
	f(b)
}

func UpdateAppInstallationDocumentRoot(documentRoot string) AppInstallationUpdater {
	return AppInstallationUpdaterFunc(func(b *appclientv2.PatchAppinstallationRequestBody) {
		b.CustomDocumentRoot = &documentRoot
	})
}

func UpdateAppInstallationUpdatePolicy(updatePolicy appv2.AppUpdatePolicy) AppInstallationUpdater {
	return AppInstallationUpdaterFunc(func(b *appclientv2.PatchAppinstallationRequestBody) {
		b.UpdatePolicy = &updatePolicy
	})
}

func UpdateAppInstallationDescription(description string) AppInstallationUpdater {
	return AppInstallationUpdaterFunc(func(b *appclientv2.PatchAppinstallationRequestBody) {
		b.Description = &description
	})
}

func UpdateAppInstallationSystemSoftware(systemSoftwareID, systemSoftwareVersionID string, updatePolicy appv2.SystemSoftwareUpdatePolicy) AppInstallationUpdater {
	return AppInstallationUpdaterFunc(func(b *appclientv2.PatchAppinstallationRequestBody) {
		if b.SystemSoftware == nil {
			systemSoftware := make(map[string]appclientv2.PatchAppinstallationRequestBodySystemSoftwareItem)
			b.SystemSoftware = systemSoftware
		}

		b.SystemSoftware[systemSoftwareID] = appclientv2.PatchAppinstallationRequestBodySystemSoftwareItem{
			SystemSoftwareVersion: &systemSoftwareVersionID,
			UpdatePolicy:          &updatePolicy,
		}
	})
}

func RemoveAppInstallationSystemSoftware(systemSoftwareID string) AppInstallationUpdater {
	return AppInstallationUpdaterFunc(func(b *appclientv2.PatchAppinstallationRequestBody) {
		if b.SystemSoftware == nil {
			systemSoftware := make(map[string]appclientv2.PatchAppinstallationRequestBodySystemSoftwareItem)
			b.SystemSoftware = systemSoftware
		}

		b.SystemSoftware[systemSoftwareID] = appclientv2.PatchAppinstallationRequestBodySystemSoftwareItem{
			SystemSoftwareVersion: nil,
			UpdatePolicy:          nil,
		}
	})
}

func (c *appClient) buildPatchRequest(appInstallationID string, updater ...AppInstallationUpdater) appclientv2.PatchAppinstallationRequest {
	req := appclientv2.PatchAppinstallationRequest{
		AppInstallationID: appInstallationID,
	}

	for _, u := range updater {
		u.Apply(&req.Body)
	}

	return req
}

func (c *appClient) UpdateAppinstallation(ctx context.Context, appInstallationID string, updater ...AppInstallationUpdater) error {
	if len(updater) == 0 {
		return nil
	}

	_, err := c.PatchAppinstallation(ctx, c.buildPatchRequest(appInstallationID, updater...))
	return err
}

// UpdateAppinstallationWithRetry wraps PatchAppinstallation with polling
// to handle transient permission errors that occur when the API has not yet
// propagated permissions for a newly created app installation.
func (c *appClient) UpdateAppinstallationWithRetry(ctx context.Context, appInstallationID string, updater ...AppInstallationUpdater) error {
	if len(updater) == 0 {
		return nil
	}

	req := c.buildPatchRequest(appInstallationID, updater...)

	_, err := apiutils.Poll(ctx, apiutils.PollOpts{
		InitialDelay: 500 * time.Millisecond,
		MaxDelay:     5 * time.Second,
	}, func(ctx context.Context, r appclientv2.PatchAppinstallationRequest) (struct{}, error) {
		_, err := c.PatchAppinstallation(ctx, r)
		return struct{}{}, err
	}, req)

	return err
}
