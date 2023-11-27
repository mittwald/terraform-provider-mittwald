package mittwaldv2

import (
	"context"
	"fmt"
	"github.com/google/uuid"
)

type CronjobClient interface {
	GetCronjob(ctx context.Context, cronjobID string) (*DeMittwaldV1CronjobCronjob, error)
	CreateCronjob(ctx context.Context, projectID string, body CronjobCreateCronjobJSONRequestBody) (string, error)
	UpdateCronjob(ctx context.Context, cronjobID string, body CronjobUpdateCronjobJSONRequestBody) error
	DeleteCronjob(ctx context.Context, cronjobID string) error
}

type cronjobClient struct {
	client ClientWithResponsesInterface
}

func (c *cronjobClient) GetCronjob(ctx context.Context, cronjobID string) (*DeMittwaldV1CronjobCronjob, error) {
	resp, err := c.client.CronjobGetCronjobWithResponse(ctx, uuid.MustParse(cronjobID))
	if err != nil {
		return nil, fmt.Errorf("error getting cronjob: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, errUnexpectedStatus(resp.StatusCode(), resp.Body)
	}

	return resp.JSON200, nil
}

func (c *cronjobClient) CreateCronjob(ctx context.Context, projectID string, body CronjobCreateCronjobJSONRequestBody) (string, error) {
	resp, err := c.client.CronjobCreateCronjobWithResponse(ctx, uuid.MustParse(projectID), body)
	if err != nil {
		return "", fmt.Errorf("error creating cronjob: %w", err)
	}

	if resp.JSON201 == nil {
		return "", errUnexpectedStatus(resp.StatusCode(), resp.Body)
	}

	return resp.JSON201.Id.String(), nil
}

func (c *cronjobClient) DeleteCronjob(ctx context.Context, cronjobID string) error {
	resp, err := c.client.CronjobDeleteCronjobWithResponse(ctx, uuid.MustParse(cronjobID))
	if err != nil {
		return fmt.Errorf("error deleting cronjob: %w", err)
	}

	if resp.StatusCode() != 204 {
		return errUnexpectedStatus(resp.StatusCode(), resp.Body)
	}

	return nil
}

func (c *cronjobClient) UpdateCronjob(ctx context.Context, cronjobID string, body CronjobUpdateCronjobJSONRequestBody) error {
	resp, err := c.client.CronjobUpdateCronjobWithResponse(ctx, uuid.MustParse(cronjobID), body)
	if err != nil {
		return fmt.Errorf("error updating cronjob: %w", err)
	}

	if resp.JSON200 == nil {
		return errUnexpectedStatus(resp.StatusCode(), resp.Body)
	}

	return nil
}
