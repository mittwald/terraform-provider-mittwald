package apiext

import (
	"context"
	"errors"
	"fmt"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/containerclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/containerv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiutils"
	"time"
)

type ContainerClient interface {
	containerclientv2.Client

	GetDefaultStack(context.Context, string) (*containerv2.StackResponse, error)
	PollDefaultStack(context.Context, string) (*containerv2.StackResponse, error)
	GetRegistryByName(ctx context.Context, projectID string, registryURI string) (*containerv2.Registry, error)
	WaitUntilStackIsReady(ctx context.Context, stackID string, containerNames []string) error
}
type containerClient struct {
	containerclientv2.Client
	clientSet mittwaldv2.Client
}

func NewContainerClient(c mittwaldv2.Client) ContainerClient {
	return &containerClient{
		Client:    c.Container(),
		clientSet: c,
	}
}

func (c *containerClient) GetRegistryByName(ctx context.Context, projectID string, registryURI string) (*containerv2.Registry, error) {
	listRegistryRequest := containerclientv2.ListRegistriesRequest{ProjectID: projectID}
	registries, _, err := c.clientSet.Container().ListRegistries(ctx, listRegistryRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to list registries: %w", err)
	}

	for _, registry := range *registries {
		if registry.Uri == registryURI {
			return &registry, nil
		}
	}

	return nil, fmt.Errorf("project %s does not have a registry with URI %s", projectID, registryURI)
}

func (c *containerClient) GetDefaultStack(ctx context.Context, projectID string) (*containerv2.StackResponse, error) {
	listStacksRequest := containerclientv2.ListStacksRequest{ProjectID: projectID}
	stacks, _, err := c.clientSet.Container().ListStacks(ctx, listStacksRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to list stacks: %w", err)
	}

	for _, stack := range *stacks {
		if stack.Description == "default" {
			return &stack, nil
		}
	}

	return nil, &ErrNoDefaultStack{ProjectID: projectID}
}

// PollDefaultStack polls until the default stack for the given project ID is found, or an error occurs.
// This is useful in scenarios where the default stack might not be immediately available after project creation.
func (c *containerClient) PollDefaultStack(ctx context.Context, projectID string) (*containerv2.StackResponse, error) {
	opts := apiutils.PollOpts{
		InitialDelay: 0,
		MaxDelay:     15 * time.Second,
	}

	runner := func(ctx context.Context, projectID string) (*containerv2.StackResponse, error) {
		stack, err := c.GetDefaultStack(ctx, projectID)
		if errors.Is(err, &ErrNoDefaultStack{}) {
			return nil, apiutils.ErrPollShouldRetry
		}

		return stack, err
	}

	return apiutils.Poll(ctx, opts, runner, projectID)
}
