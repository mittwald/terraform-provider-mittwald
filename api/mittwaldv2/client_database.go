package mittwaldv2

import (
	"context"
	"github.com/google/uuid"
)

type DatabaseClient interface {
	CreateMySQLDatabase(ctx context.Context, projectID string, body DatabaseCreateMysqlDatabaseJSONRequestBody) (string, string, error)
	SetMySQLDatabaseDescription(ctx context.Context, databaseID string, description string) error
	DeleteMySQLDatabase(ctx context.Context, databaseID string) error
	PollMySQLDatabase(ctx context.Context, databaseID string) (*DeMittwaldV1DatabaseMySqlDatabase, error)
	SetMySQLUserPassword(ctx context.Context, userID string, password string) error
	PollMySQLUsersForDatabase(ctx context.Context, databaseID string) ([]DeMittwaldV1DatabaseMySqlUser, error)
	PollMySQLUser(ctx context.Context, userID string) (*DeMittwaldV1DatabaseMySqlUser, error)
}

type databaseClient struct {
	client ClientWithResponsesInterface
}

func (c *databaseClient) CreateMySQLDatabase(ctx context.Context, projectID string, body DatabaseCreateMysqlDatabaseJSONRequestBody) (string, string, error) {
	response, err := c.client.DatabaseCreateMysqlDatabaseWithResponse(ctx, uuid.MustParse(projectID), body)
	if err != nil {
		return "", "", err
	}

	if response.JSON201 != nil {
		return response.JSON201.Id.String(), response.JSON201.UserId.String(), nil
	}

	return "", "", errUnexpectedStatus(response.StatusCode(), response.Body)
}

func (c *databaseClient) SetMySQLDatabaseDescription(ctx context.Context, databaseID string, description string) error {
	response, err := c.client.DatabaseUpdateMysqlDatabaseDescriptionWithResponse(ctx, uuid.MustParse(databaseID), DatabaseUpdateMysqlDatabaseDescriptionJSONRequestBody{
		Description: description,
	})
	if err != nil {
		return err
	}

	if response.StatusCode() >= 200 && response.StatusCode() < 300 {
		return nil
	}

	return errUnexpectedStatus(response.StatusCode(), response.Body)
}

func (c *databaseClient) DeleteMySQLDatabase(ctx context.Context, databaseID string) error {
	response, err := c.client.DatabaseDeleteMysqlDatabaseWithResponse(ctx, uuid.MustParse(databaseID))
	if err != nil {
		return err
	}

	if response.StatusCode() >= 200 && response.StatusCode() < 300 {
		return nil
	}

	return errUnexpectedStatus(response.StatusCode(), response.Body)
}

func (c *databaseClient) PollMySQLDatabase(ctx context.Context, databaseID string) (*DeMittwaldV1DatabaseMySqlDatabase, error) {
	return poll(ctx, func() (*DeMittwaldV1DatabaseMySqlDatabase, error) {
		response, err := c.client.DatabaseGetMysqlDatabaseWithResponse(ctx, uuid.MustParse(databaseID))
		if err != nil {
			return nil, err
		}

		if response.JSON200 != nil {
			return response.JSON200, nil
		}

		return nil, errUnexpectedStatus(response.StatusCode(), response.Body)
	})
}

func (c *databaseClient) SetMySQLUserPassword(ctx context.Context, userID string, password string) error {
	response, err := c.client.DatabaseUpdateMysqlUserPasswordWithResponse(ctx, uuid.MustParse(userID), DatabaseUpdateMysqlUserPasswordJSONRequestBody{
		Password: password,
	})
	if err != nil {
		return err
	}

	if response.StatusCode() >= 200 && response.StatusCode() < 300 {
		return nil
	}

	return errUnexpectedStatus(response.StatusCode(), response.Body)
}

func (c *databaseClient) PollMySQLUsersForDatabase(ctx context.Context, databaseID string) ([]DeMittwaldV1DatabaseMySqlUser, error) {
	return poll(ctx, func() ([]DeMittwaldV1DatabaseMySqlUser, error) {
		response, err := c.client.DatabaseListMysqlUsersWithResponse(ctx, uuid.MustParse(databaseID))
		if err != nil {
			return nil, err
		}

		if response.JSON200 != nil {
			return *response.JSON200, nil
		}

		return nil, errUnexpectedStatus(response.StatusCode(), response.Body)
	})
}

func (c *databaseClient) PollMySQLUser(ctx context.Context, userID string) (*DeMittwaldV1DatabaseMySqlUser, error) {
	return poll(ctx, func() (*DeMittwaldV1DatabaseMySqlUser, error) {
		response, err := c.client.DatabaseGetMysqlUserWithResponse(ctx, uuid.MustParse(userID))
		if err != nil {
			return nil, err
		}

		if response.JSON200 != nil {
			return response.JSON200, nil
		}

		return nil, errUnexpectedStatus(response.StatusCode(), response.Body)
	})
}
