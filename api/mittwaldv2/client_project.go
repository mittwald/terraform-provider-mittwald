package mittwaldv2

import (
	"context"
	"github.com/google/uuid"
)

type ProjectClient struct {
	client ClientWithResponsesInterface
}

func (c *ProjectClient) CreateProjectOnServer(ctx context.Context, serverID string, body ProjectCreateProjectJSONRequestBody) (string, error) {
	response, err := c.client.ProjectCreateProjectWithResponse(
		ctx,
		uuid.MustParse(serverID),
		body,
	)

	if err != nil {
		return "", err
	}

	if response.JSON201 != nil {
		return response.JSON201.Id.String(), nil
	}

	return "", errUnexpectedStatus(response.StatusCode(), response.Body)
}

func (c *ProjectClient) DeleteProject(ctx context.Context, projectID string) error {
	response, err := c.client.ProjectDeleteProjectWithResponse(ctx, projectID)
	if err != nil {
		return err
	}

	if response.JSON200 != nil {
		return nil
	}

	return errUnexpectedStatus(response.StatusCode(), response.Body)
}

func (c *ProjectClient) PollProject(ctx context.Context, projectID string) (*DeMittwaldV1ProjectProject, error) {
	return poll(ctx, func() (*DeMittwaldV1ProjectProject, error) {
		response, err := c.client.ProjectGetProjectWithResponse(ctx, uuid.MustParse(projectID))
		if err != nil {
			return nil, err
		}

		if response.JSON200 != nil {
			return response.JSON200, nil
		}

		return nil, errUnexpectedStatus(response.StatusCode(), response.Body)
	})
}
