package apiext

import (
	"context"
	"errors"

	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/domainclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/projectclientv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiutils"
)

// ErrNoDefaultIngress is returned when a project does not have a default ingress yet,
// or when the default ingress exists but its IP addresses have not been populated yet.
var ErrNoDefaultIngress = errors.New("project does not have a default ingress")

type ProjectClient interface {
	projectclientv2.Client

	GetProjectDefaultIPs(context.Context, string) ([]string, error)
	PollProjectDefaultIPs(context.Context, string) ([]string, error)
}

type projectClient struct {
	projectclientv2.Client
	clientSet mittwaldv2.Client
}

func NewProjectClient(c mittwaldv2.Client) ProjectClient {
	return &projectClient{
		Client:    c.Project(),
		clientSet: c,
	}
}

func (c *projectClient) GetProjectDefaultIPs(ctx context.Context, projectID string) ([]string, error) {
	ingressesRequest := domainclientv2.ListIngressesRequest{ProjectID: &projectID}
	ingresses, _, err := c.clientSet.Domain().ListIngresses(ctx, ingressesRequest)

	if err != nil {
		return nil, err
	}

	for _, ingress := range *ingresses {
		// IP addresses are populated asynchronously, so even if we find the
		// default ingress, it may not have IPs yet. In that case, we want to
		// return ErrNoDefaultIngress so that callers can safely retry.
		if ingress.IsDefault && len(ingress.Ips.V4) > 0 {
			return ingress.Ips.V4, nil
		}
	}

	return nil, ErrNoDefaultIngress
}

// PollProjectDefaultIPs polls for the project's default IP addresses until they
// become available, or until the context is done. Since the default ingress is
// created asynchronously after a project is created, this may take a while.
//
// If the context expires before the default ingress becomes available, this
// method returns ErrNoDefaultIngress; callers can use this to distinguish "not
// (yet) available" from an actual API error, and degrade gracefully.
func (c *projectClient) PollProjectDefaultIPs(ctx context.Context, projectID string) ([]string, error) {
	pollIPs := func(ctx context.Context, projectID string) ([]string, error) {
		ips, err := c.GetProjectDefaultIPs(ctx, projectID)
		if errors.Is(err, ErrNoDefaultIngress) {
			return nil, apiutils.ErrPollShouldRetry
		}
		return ips, err
	}

	ips, err := apiutils.Poll(ctx, apiutils.PollOpts{}, pollIPs, projectID)
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return nil, ErrNoDefaultIngress
	}

	return ips, err
}
