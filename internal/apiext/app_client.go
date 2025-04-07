package apiext

import (
	"context"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/appclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/appv2"
)

type AppClient interface {
	appclientv2.Client
	GetSystemsoftwareByName(ctx context.Context, name string) (*appv2.SystemSoftware, bool, error)
	GetSystemsoftwareAndVersion(ctx context.Context, systemSoftwareID, systemSoftwareVersionID string) (*appv2.SystemSoftware, *appv2.SystemSoftwareVersion, error)
	SelectSystemsoftwareVersion(ctx context.Context, systemSoftwareID, versionSelector string) (SystemSoftwareVersionSet, error)
	UpdateAppinstallation(ctx context.Context, appInstallationID string, updater ...AppInstallationUpdater) error
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

func (c *appClient) UpdateAppinstallation(ctx context.Context, appInstallationID string, updater ...AppInstallationUpdater) error {
	if len(updater) == 0 {
		return nil
	}

	req := appclientv2.PatchAppinstallationRequest{
		AppInstallationID: appInstallationID,
	}

	for _, u := range updater {
		u.Apply(&req.Body)
	}

	_, err := c.Client.PatchAppinstallation(ctx, req)
	return err
}
