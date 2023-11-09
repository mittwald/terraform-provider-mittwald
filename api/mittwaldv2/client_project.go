package mittwaldv2

import (
	"context"
	"github.com/google/uuid"
)

type ProjectClient struct {
	client ClientWithResponsesInterface
}

func (c *ProjectClient) ListProjects(ctx context.Context) ([]DeMittwaldV1ProjectProject, error) {
	response, err := c.client.ProjectListProjectsWithResponse(ctx, &ProjectListProjectsParams{})
	if err != nil {
		return nil, err
	}

	if response.JSON200 != nil {
		out := make([]DeMittwaldV1ProjectProject, len(*response.JSON200))

		for i, project := range *response.JSON200 {
			out[i].Id = uuid.MustParse(project.Id)
			out[i].ShortId = project.ShortId
			out[i].Description = project.Description
			out[i].CreatedAt = project.CreatedAt
			out[i].CustomerId = project.CustomerId
			out[i].Description = project.Description
			// out[i].Directories = project.Directories
			out[i].DisableReason = project.DisableReason
			out[i].Enabled = project.Enabled
			out[i].IsReady = project.IsReady
			out[i].ProjectHostingId = project.ProjectHostingId
			out[i].Readiness = project.Readiness

			if s := project.ServerId; s != nil {
				u := uuid.MustParse(*s)
				out[i].ServerId = &u
			}

			if project.ImageRefId != nil {
				u := uuid.MustParse(*project.ImageRefId)
				out[i].ImageRefId = &u
			}

		}

		return out, nil
	}

	return nil, errUnexpectedStatus(response.StatusCode(), response.Body)
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
