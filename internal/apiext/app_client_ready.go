package apiext

import (
	"context"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/appclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/appv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiutils"
	"net/http"
	"time"
)

func (c *appClient) WaitUntilAppInstallationIsReady(ctx context.Context, appID string) error {
	request := appclientv2.GetAppinstallationRequest{AppInstallationID: appID}

	runner := func(ctx context.Context, req appclientv2.GetAppinstallationRequest, reqEditors ...func(req *http.Request) error) (*appv2.AppInstallation, *http.Response, error) {
		inst, resp, err := c.GetAppinstallation(ctx, req, reqEditors...)
		if err != nil {
			return nil, nil, err
		}

		if inst.AppVersion.Current == nil {
			return nil, nil, apiutils.ErrPollShouldRetry
		}

		if *inst.AppVersion.Current != inst.AppVersion.Desired {
			return nil, nil, apiutils.ErrPollShouldRetry
		}

		return inst, resp, nil
	}

	o := apiutils.PollOpts{
		InitialDelay: 1 * time.Second,
		MaxDelay:     60 * time.Second,
	}

	_, err := apiutils.Poll(ctx, o, runner, request)
	return err
}
