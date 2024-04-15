package mittwaldv2

import (
	"context"
	"fmt"
)

type UserClient interface {
	GetCurrentUser(ctx context.Context) (*DeMittwaldV1SignupAccount, error)
}

type userClient struct {
	client ClientWithResponsesInterface
}

func (c *userClient) GetCurrentUser(ctx context.Context) (*DeMittwaldV1SignupAccount, error) {
	resp, err := c.client.UserGetOwnAccountWithResponse(ctx, UserGetOwnAccountJSONRequestBody{})
	if err != nil {
		return nil, fmt.Errorf("error getting cronjob: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, errUnexpectedStatus(resp.StatusCode(), resp.Body)
	}

	return resp.JSON200, nil
}
