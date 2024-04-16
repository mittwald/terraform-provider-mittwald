package mittwaldv2

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type UserClient interface {
	GetCurrentUser(ctx context.Context) (*DeMittwaldV1UserUser, error)
	GetUser(ctx context.Context, userID string) (*DeMittwaldV1UserUser, error)
}

type userClient struct {
	client ClientWithResponsesInterface
}

func (c *userClient) GetCurrentUser(ctx context.Context) (*DeMittwaldV1UserUser, error) {
	return c.GetUser(ctx, "self")
}

func (c *userClient) GetUser(ctx context.Context, userID string) (*DeMittwaldV1UserUser, error) {
	// NOTE:
	// It is necessary to work directly with the innards of the client here,
	// because the generated client is incorrect at this point. This is related
	// to a number of issues:
	//
	// - https://github.com/deepmap/oapi-codegen/issues/1429
	// - https://github.com/deepmap/oapi-codegen/issues/1433
	// - https://github.com/deepmap/oapi-codegen/issues/1029

	clientWithResponses, ok := c.client.(*ClientWithResponses)
	if !ok {
		return nil, fmt.Errorf("unexpected client type: %T", c.client)
	}

	client, ok := clientWithResponses.ClientInterface.(*Client)
	if !ok {
		return nil, fmt.Errorf("unexpected client type: %T", clientWithResponses.ClientInterface)
	}

	serverURL, err := url.Parse(client.Server)
	if err != nil {
		return nil, fmt.Errorf("error parsing server URL: %w", err)
	}
	operationURL, err := serverURL.Parse(fmt.Sprintf("/v2/users/%s", userID))
	if err != nil {
		return nil, fmt.Errorf("error parsing operation URL: %w", err)
	}

	req, _ := http.NewRequest("GET", operationURL.String(), nil)
	req = req.WithContext(ctx)
	if err := client.applyEditors(ctx, req, nil); err != nil {
		return nil, err
	}
	httpResp, err := client.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error getting user: %w", err)
	}

	resp, err := ParseUserGetUserResponse(httpResp)
	if err != nil {
		return nil, fmt.Errorf("error getting user: %w", err)
	}

	if resp.JSON200 == nil {
		return nil, errUnexpectedStatus(resp.StatusCode(), resp.Body)
	}

	return resp.JSON200, nil
}
