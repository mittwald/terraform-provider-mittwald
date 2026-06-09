package apiext

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/appclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/appv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiutils"
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

	_, err := c.retryWhilePhaseNotReady(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.PatchAppinstallation(ctx, req)
	})
	return err
}

// LinkDatabase wraps the generated client call with a retry on the transient
// "not in ready phase" error; see retryWhilePhaseNotReady.
func (c *appClient) LinkDatabase(ctx context.Context, req appclientv2.LinkDatabaseRequest, reqEditors ...func(req *http.Request) error) (*http.Response, error) {
	return c.retryWhilePhaseNotReady(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.Client.LinkDatabase(ctx, req, reqEditors...)
	})
}

// UnlinkDatabase wraps the generated client call with a retry on the transient
// "not in ready phase" error; see retryWhilePhaseNotReady.
func (c *appClient) UnlinkDatabase(ctx context.Context, req appclientv2.UnlinkDatabaseRequest, reqEditors ...func(req *http.Request) error) (*http.Response, error) {
	return c.retryWhilePhaseNotReady(ctx, func(ctx context.Context) (*http.Response, error) {
		return c.Client.UnlinkDatabase(ctx, req, reqEditors...)
	})
}

// isPhaseNotReadyError reports whether err indicates that the app installation
// is currently in a phase that does not accept mutating requests. After an
// installation first becomes ready, the backend briefly transitions through
// additional phases (e.g. while deploying the project environment) and rejects
// mutations during that window with an error such as:
//
//	VError: expected phase APP_PHASE_READY, current phase is APP_PHASE_DEPLOYING_PROJECT_ENVIRONMENT
//
// Those phases are not exposed through the REST "phase" enum and are transient,
// so the request should simply be retried until the installation settles.
func isPhaseNotReadyError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "expected phase APP_PHASE_READY")
}

// retryWhilePhaseNotReady runs f, retrying for as long as the installation
// rejects the request because it is not in the "ready" phase. The first attempt
// runs immediately so the common (already-ready) case is not delayed.
func (c *appClient) retryWhilePhaseNotReady(ctx context.Context, f func(context.Context) (*http.Response, error)) (*http.Response, error) {
	resp, err := f(ctx)
	if !isPhaseNotReadyError(err) {
		return resp, err
	}

	tflog.Debug(ctx, "app installation is not in a mutable phase yet; retrying", map[string]any{"error": err.Error()})

	o := apiutils.PollOpts{
		InitialDelay: 2 * time.Second,
		MaxDelay:     30 * time.Second,
	}

	return apiutils.Poll(ctx, o, func(ctx context.Context, _ struct{}) (*http.Response, error) {
		resp, err := f(ctx)
		if isPhaseNotReadyError(err) {
			return nil, apiutils.ErrPollShouldRetry
		}
		return resp, err
	}, struct{}{})
}
