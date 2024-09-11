package mittwaldv2

import (
	"context"
	"fmt"
	"github.com/google/uuid"
)

type ProjectClient interface {
	ListProjects(ctx context.Context) ([]DeMittwaldV1ProjectProject, error)
	GetProject(ctx context.Context, projectID string) (*DeMittwaldV1ProjectProject, error)
	CreateProjectOnServer(ctx context.Context, serverID string, body ProjectCreateProjectJSONRequestBody) (string, error)
	DeleteProject(ctx context.Context, projectID string) error
	PollProject(ctx context.Context, projectID string) (*DeMittwaldV1ProjectProject, error)
	GetProjectDefaultIPs(ctx context.Context, projectID string) ([]string, error)
	UpdateProjectDescription(ctx context.Context, projectID, description string) error
}

type projectClient struct {
	client ClientWithResponsesInterface
}

func (c *projectClient) ListProjects(ctx context.Context) ([]DeMittwaldV1ProjectProject, error) {
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

func (c *projectClient) GetProject(ctx context.Context, projectID string) (*DeMittwaldV1ProjectProject, error) {
	response, err := c.client.ProjectGetProjectWithResponse(ctx, uuid.MustParse(projectID))
	if err != nil {
		return nil, err
	}

	if response.JSON200 != nil {
		return response.JSON200, nil
	}

	return nil, errUnexpectedStatus(response.StatusCode(), response.Body)
}

func (c *projectClient) CreateProjectOnServer(ctx context.Context, serverID string, body ProjectCreateProjectJSONRequestBody) (string, error) {
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

func (c *projectClient) UpdateProjectDescription(ctx context.Context, projectID, description string) error {
	response, err := c.client.ProjectUpdateProjectDescriptionWithResponse(ctx, uuid.MustParse(projectID), ProjectUpdateProjectDescriptionJSONRequestBody{
		Description: description,
	})

	if err != nil {
		return fmt.Errorf("error updating project description: %w", err)
	}

	if response.StatusCode() >= 400 {
		return errUnexpectedStatus(response.StatusCode(), response.Body)
	}

	return nil
}

func (c *projectClient) DeleteProject(ctx context.Context, projectID string) error {
	response, err := c.client.ProjectDeleteProjectWithResponse(ctx, projectID)
	if err != nil {
		return err
	}

	if response.StatusCode() >= 400 {
		return errUnexpectedStatus(response.StatusCode(), response.Body)
	}

	return nil
}

func (c *projectClient) PollProject(ctx context.Context, projectID string) (*DeMittwaldV1ProjectProject, error) {
	return poll(ctx, pollOpts{}, func() (*DeMittwaldV1ProjectProject, error) {
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

func (c *projectClient) GetProjectDefaultIPs(ctx context.Context, projectID string) ([]string, error) {
	projectUUID := uuid.MustParse(projectID)
	response, err := c.client.IngressListIngressesWithResponse(ctx, &IngressListIngressesParams{ProjectId: &projectUUID})
	if err != nil {
		return nil, err
	}

	if response.JSON200 == nil {
		return nil, errUnexpectedStatus(response.StatusCode(), response.Body)
	}

	for _, ingress := range *response.JSON200 {
		if ingress.IsDefault {
			return ingress.Ips.V4, nil
		}
	}

	return nil, fmt.Errorf("project %s does not appear to have a default ingress", projectID)
}
