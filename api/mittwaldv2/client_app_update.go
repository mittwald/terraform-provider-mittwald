package mittwaldv2

import (
	"context"
	"github.com/google/uuid"
)

type AppInstallationUpdater interface {
	Apply(b *AppPatchAppinstallationJSONRequestBody)
}

type AppInstallationUpdaterFunc func(b *AppPatchAppinstallationJSONRequestBody)
type AppInstallationUpdaterChain []AppInstallationUpdater

func (c AppInstallationUpdaterChain) Apply(b *AppPatchAppinstallationJSONRequestBody) {
	for _, u := range c {
		u.Apply(b)
	}
}

func (f AppInstallationUpdaterFunc) Apply(b *AppPatchAppinstallationJSONRequestBody) {
	f(b)
}

func UpdateAppInstallationDocumentRoot(documentRoot string) AppInstallationUpdater {
	return AppInstallationUpdaterFunc(func(b *AppPatchAppinstallationJSONRequestBody) {
		b.CustomDocumentRoot = &documentRoot
	})
}

func UpdateAppInstallationUpdatePolicy(updatePolicy DeMittwaldV1AppAppUpdatePolicy) AppInstallationUpdater {
	return AppInstallationUpdaterFunc(func(b *AppPatchAppinstallationJSONRequestBody) {
		b.UpdatePolicy = &updatePolicy
	})
}

func UpdateAppInstallationDescription(description string) AppInstallationUpdater {
	return AppInstallationUpdaterFunc(func(b *AppPatchAppinstallationJSONRequestBody) {
		b.Description = &description
	})
}

func UpdateAppInstallationSystemSoftware(systemSoftwareID, systemSoftwareVersionID string, updatePolicy DeMittwaldV1AppSystemSoftwareUpdatePolicy) AppInstallationUpdater {
	return AppInstallationUpdaterFunc(func(b *AppPatchAppinstallationJSONRequestBody) {
		if b.SystemSoftware == nil {
			systemSoftware := make(AppPatchInstallationSystemSoftware)
			b.SystemSoftware = &systemSoftware
		}

		(*b.SystemSoftware)[systemSoftwareID] = AppPatchInstallationSystemSoftwareItem{
			SystemSoftwareVersion: &systemSoftwareVersionID,
			UpdatePolicy:          &updatePolicy,
		}
	})
}

func RemoveAppInstallationSystemSoftware(systemSoftwareID string) AppInstallationUpdater {
	return AppInstallationUpdaterFunc(func(b *AppPatchAppinstallationJSONRequestBody) {
		if b.SystemSoftware == nil {
			systemSoftware := make(AppPatchInstallationSystemSoftware)
			b.SystemSoftware = &systemSoftware
		}

		(*b.SystemSoftware)[systemSoftwareID] = AppPatchInstallationSystemSoftwareItem{
			SystemSoftwareVersion: nil,
			UpdatePolicy:          nil,
		}
	})
}

func (c *appClient) UpdateAppInstallation(ctx context.Context, appInstallationID string, updater ...AppInstallationUpdater) error {
	body := AppPatchAppinstallationJSONRequestBody{}

	for _, u := range updater {
		u.Apply(&body)
	}

	response, err := c.client.AppPatchAppinstallationWithResponse(ctx, uuid.MustParse(appInstallationID), body)
	if err != nil {
		return err
	}

	if response.StatusCode() >= 200 && response.StatusCode() < 300 {
		return nil
	}

	return errUnexpectedStatus(response.StatusCode(), response.Body)
}
