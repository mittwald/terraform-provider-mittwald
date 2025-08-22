package apiext

import (
	"context"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/containerclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/containerv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiutils"
	"net/http"
	"time"
)

// WaitUntilStackIsReady waits until the specified stack is ready, meaning all
// specified containers are running. If `containerNames` is nil, it waits for all
// containers in the stack to be running.
func (c *containerClient) WaitUntilStackIsReady(ctx context.Context, stackID string, containerNames []string) error {
	containerNameMap := make(map[string]struct{}, len(containerNames))
	for _, name := range containerNames {
		containerNameMap[name] = struct{}{}
	}

	request := containerclientv2.GetStackRequest{StackID: stackID}

	runner := func(ctx context.Context, req containerclientv2.GetStackRequest, reqEditors ...func(req *http.Request) error) (*containerv2.StackResponse, *http.Response, error) {
		stack, resp, err := c.GetStack(ctx, req, reqEditors...)
		if err != nil {
			return nil, nil, err
		}

		for _, service := range stack.Services {
			if containerNames != nil {
				if _, ok := containerNameMap[service.ServiceName]; !ok {
					continue
				}
			}

			if service.Status != containerv2.ServiceStatusRunning {
				return nil, nil, apiutils.ErrPollShouldRetry
			}
		}

		return stack, resp, nil
	}

	o := apiutils.PollOpts{
		InitialDelay: 1 * time.Second,
		MaxDelay:     60 * time.Second,
	}

	_, err := apiutils.Poll(ctx, o, runner, request)
	return err
}
