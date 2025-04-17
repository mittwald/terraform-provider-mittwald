package apiext

import (
	"context"
	"fmt"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/domainclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/ingressv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/ptrutil"
)

type DomainClient interface {
	domainclientv2.Client

	GetIngressByName(ctx context.Context, projectID, ingressName string) (*ingressv2.Ingress, error)
}
type domainClient struct {
	domainclientv2.Client
	clientSet mittwaldv2.Client
}

func NewDomainClient(c mittwaldv2.Client) DomainClient {
	return &domainClient{
		Client:    c.Domain(),
		clientSet: c,
	}
}

func (c *domainClient) GetIngressByName(ctx context.Context, projectID, ingressName string) (*ingressv2.Ingress, error) {
	listIngressRequest := domainclientv2.ListIngressesRequest{ProjectID: &projectID, Limit: ptrutil.To[int64](0)}
	ingresses, _, err := c.clientSet.Domain().ListIngresses(ctx, listIngressRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to list ingresses: %w", err)
	}

	for _, ingress := range *ingresses {
		if ingress.Hostname == ingressName {
			return &ingress, nil
		}
	}

	return nil, fmt.Errorf("project %s does not appear to have an ingress with name '%s'", projectID, ingressName)
}
