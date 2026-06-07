package apiext

import (
	"context"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/appclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/appv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiutils"
	"net/http"
	"time"
)

func strOrNil(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

func (c *appClient) WaitUntilAppInstallationIsReady(ctx context.Context, appID string) error {
	request := appclientv2.GetAppinstallationRequest{AppInstallationID: appID}

	runner := func(ctx context.Context, req appclientv2.GetAppinstallationRequest, reqEditors ...func(req *http.Request) error) (*appv2.AppInstallation, *http.Response, error) {
		inst, resp, err := c.GetAppinstallation(ctx, req, reqEditors...)
		if err != nil {
			return nil, nil, err
		}

		notReady := func(reason string, extra map[string]any) (*appv2.AppInstallation, *http.Response, error) {
			fields := map[string]any{
				"app_installation_id": inst.Id,
				"phase":               string(inst.Phase),
				"reason":              reason,
			}
			for k, v := range extra {
				fields[k] = v
			}
			tflog.Debug(ctx, "app installation not ready yet", fields)
			return nil, nil, apiutils.ErrPollShouldRetry
		}

		// The API only accepts mutating requests (such as PATCH or linking a
		// database) once the installation has finished provisioning and reached
		// the "ready" phase. This also covers the "reconfiguring"/"upgrading"
		// phases that an app or dependency update transitions through.
		if inst.Phase != appv2.PhaseReady {
			return notReady("phase is not ready", nil)
		}

		// Wait until the running app version has caught up with the desired one.
		if inst.AppVersion.Current == nil || *inst.AppVersion.Current != inst.AppVersion.Desired {
			return notReady("app version not converged", map[string]any{
				"app_version_current": strOrNil(inst.AppVersion.Current),
				"app_version_desired": inst.AppVersion.Desired,
			})
		}

		// Wait until every *installed* system software dependency (e.g. PHP) has
		// caught up with its desired version. A configuration change such as a
		// PHP version bump takes the app offline for a while; the desired version
		// is recorded synchronously by the PATCH while the installed version lags
		// behind, so this reliably detects an update that is still being applied.
		//
		// We deliberately ignore entries whose current version is nil: those are
		// merely *suggested* dependencies that are not installed yet and only get
		// installed once they are explicitly configured via PATCH. Waiting for
		// them here would deadlock the create flow, which has to reach a ready
		// state before it is allowed to send that PATCH in the first place.
		for _, software := range inst.SystemSoftware {
			version := software.SystemSoftwareVersion
			if version.Current != nil && *version.Current != version.Desired {
				return notReady("system software version not converged", map[string]any{
					"system_software":         software.Name,
					"system_software_id":      software.SystemSoftwareId,
					"system_software_current": strOrNil(version.Current),
					"system_software_desired": version.Desired,
				})
			}
		}

		tflog.Debug(ctx, "app installation is ready", map[string]any{"app_installation_id": inst.Id})

		return inst, resp, nil
	}

	o := apiutils.PollOpts{
		InitialDelay: 1 * time.Second,
		MaxDelay:     60 * time.Second,
	}

	_, err := apiutils.PollRequest(ctx, o, runner, request)
	return err
}
