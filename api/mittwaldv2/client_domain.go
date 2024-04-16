package mittwaldv2

import (
	"context"
	"github.com/google/uuid"
)

type DomainClient interface {
	GetIngress(ctx context.Context, ingressID string) (*DeMittwaldV1IngressIngress, error)
	CreateIngress(ctx context.Context, projectID string, body IngressCreateIngressJSONRequestBody) (string, error)
	UpdateIngressPaths(ctx context.Context, ingressID string, body IngressUpdateIngressPathsJSONRequestBody) error
	DeleteIngress(ctx context.Context, ingressID string) error
}

type domainClient struct {
	client ClientWithResponsesInterface
}

func (c *domainClient) GetIngress(ctx context.Context, ingressID string) (*DeMittwaldV1IngressIngress, error) {
	resp, err := c.client.IngressGetIngressWithResponse(ctx, uuid.MustParse(ingressID))
	if err != nil {
		return nil, err
	}

	if resp.JSON200 == nil {
		return nil, errUnexpectedStatus(resp.StatusCode(), resp.Body)
	}

	return resp.JSON200, nil
}

func (c *domainClient) CreateIngress(ctx context.Context, projectID string, body IngressCreateIngressJSONRequestBody) (string, error) {
	resp, err := c.client.IngressCreateIngressWithResponse(ctx, body)
	if err != nil {
		return "", err
	}

	body.ProjectId = uuid.MustParse(projectID)

	if resp.JSON201 == nil {
		return "", errUnexpectedStatus(resp.StatusCode(), resp.Body)
	}

	return resp.JSON201.Id.String(), nil
}

func (c *domainClient) UpdateIngressPaths(ctx context.Context, ingressID string, body IngressUpdateIngressPathsJSONRequestBody) error {
	resp, err := c.client.IngressUpdateIngressPathsWithResponse(ctx, uuid.MustParse(ingressID), body)
	if err != nil {
		return err
	}

	if resp.StatusCode() != 204 {
		return errUnexpectedStatus(resp.StatusCode(), resp.Body)
	}

	return nil
}

func (c *domainClient) DeleteIngress(ctx context.Context, ingressID string) error {
	resp, err := c.client.IngressDeleteIngressWithResponse(ctx, uuid.MustParse(ingressID))
	if err != nil {
		return err
	}

	if resp.StatusCode() != 204 {
		return errUnexpectedStatus(resp.StatusCode(), resp.Body)
	}

	return nil
}
