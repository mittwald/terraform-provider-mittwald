package apiext

import (
	"context"
	"fmt"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/domainclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/projectclientv2"
)

type ProjectClient interface {
	projectclientv2.Client

	GetProjectDefaultIPs(context.Context, string) ([]string, error)
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
		if ingress.IsDefault {
			return ingress.Ips.V4, nil
		}
	}

	return nil, fmt.Errorf("project %s does not appear to have a default ingress", projectID)
}
