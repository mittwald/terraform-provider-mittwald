package apiext

import (
	"context"
	"fmt"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/containerclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/containerv2"
)

type ContainerClient interface {
	containerclientv2.Client

	GetDefaultStack(context.Context, string) (*containerv2.StackResponse, error)
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

// GetDefaultStack returns the legacy default stack of the given project.
//
// Projects used to come with a pre-defined default stack. New projects do not
// have one anymore, and existing default stacks may have been deleted; in that
// case, an ErrNoDefaultStack is returned.
func (c *containerClient) GetDefaultStack(ctx context.Context, projectID string) (*containerv2.StackResponse, error) {
	listStacksRequest := containerclientv2.ListStacksRequest{ProjectID: projectID}
	stacks, _, err := c.clientSet.Container().ListStacks(ctx, listStacksRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to list stacks: %w", err)
	}

	for _, stack := range *stacks {
		if IsDefaultStack(&stack) {
			return &stack, nil
		}
	}

	return nil, &ErrNoDefaultStack{ProjectID: projectID}
}

// IsDefaultStack returns true if the given stack is a project's (legacy)
// default stack. Default stacks share their ID with the project they belong to,
// and are named "default".
func IsDefaultStack(stack *containerv2.StackResponse) bool {
	return stack.Id == stack.ProjectId || stack.Description == "default"
}
