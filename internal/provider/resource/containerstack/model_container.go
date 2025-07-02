package containerstackresource

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/containerv2"
)

func (m *ContainerModel) Equals(other *ContainerModel) bool {
	if !m.ID.Equal(other.ID) {
		return false
	}

	if !m.Image.Equal(other.Image) {
		return false
	}

	if !m.Description.Equal(other.Description) {
		return false
	}

	if !m.Command.Equal(other.Command) {
		return false
	}

	if !m.Entrypoint.Equal(other.Entrypoint) {
		return false
	}

	if !m.Environment.Equal(other.Environment) {
		return false
	}

	if !m.Ports.Equal(other.Ports) {
		return false
	}

	if !m.Volumes.Equal(other.Volumes) {
		return false
	}

	return true
}

func (m *ContainerModel) ToDeclareRequest(ctx context.Context, d *diag.Diagnostics) containerv2.ServiceDeclareRequest {
	return containerv2.ServiceDeclareRequest{
		Image:       m.Image.ValueString(),
		Command:     extractStringList(m.Command),
		Entrypoint:  extractStringList(m.Entrypoint),
		Environment: extractStringMap(m.Environment),
		Ports:       extractPortMappings(ctx, m.Ports, d),
		Volumes:     extractVolumeMappings(ctx, m.Volumes, d),
		Description: m.Description.ValueStringPointer(),
	}
}

func (m *ContainerModel) ToUpdateRequestFromExisting(ctx context.Context, other *ContainerModel, d *diag.Diagnostics) containerv2.ServiceRequest {
	req := containerv2.ServiceRequest{}

	if !m.Image.Equal(other.Image) {
		req.Image = m.Image.ValueStringPointer()
	}

	if !m.Command.Equal(other.Command) {
		req.Command = extractStringList(m.Command)
	}

	if !m.Entrypoint.Equal(other.Entrypoint) {
		req.Entrypoint = extractStringList(m.Entrypoint)
	}

	if !m.Environment.Equal(other.Environment) {
		req.Envs = extractStringMap(m.Environment)
	}

	if !m.Ports.Equal(other.Ports) {
		req.Ports = extractPortMappings(ctx, m.Ports, d)
	}

	if !m.Volumes.Equal(other.Volumes) {
		req.Volumes = extractVolumeMappings(ctx, m.Volumes, d)
	}

	if !m.Description.Equal(other.Description) {
		req.Description = m.Description.ValueStringPointer()
	}

	return req
}

func (m *ContainerModel) ToUpdateRequestFromEmpty(ctx context.Context, d *diag.Diagnostics) containerv2.ServiceRequest {
	return containerv2.ServiceRequest{
		Image:       m.Image.ValueStringPointer(),
		Command:     extractStringList(m.Command),
		Entrypoint:  extractStringList(m.Entrypoint),
		Envs:        extractStringMap(m.Environment),
		Ports:       extractPortMappings(ctx, m.Ports, d),
		Volumes:     extractVolumeMappings(ctx, m.Volumes, d),
		Description: m.Description.ValueStringPointer(),
	}
}
