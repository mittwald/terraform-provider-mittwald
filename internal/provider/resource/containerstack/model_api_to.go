package containerstackresource

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/containerclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/containerv2"
)

func (m *ContainerStackModel) ToDeclareRequest(ctx context.Context, d *diag.Diagnostics) *containerclientv2.DeclareStackRequest {
	declareRequest := &containerclientv2.DeclareStackRequest{
		StackID: m.ID.ValueString(),
		Body: containerclientv2.DeclareStackRequestBody{
			Services: make(map[string]containerv2.ServiceDeclareRequest),
			Volumes:  make(map[string]containerv2.VolumeDeclareRequest),
		},
	}

	// Process Containers
	var containerModels map[string]ContainerModel
	detailDiags := m.Containers.ElementsAs(ctx, &containerModels, false)
	if detailDiags.HasError() {
		d.Append(detailDiags...)
		return nil
	}

	for name, container := range containerModels {
		service := containerv2.ServiceDeclareRequest{
			Image:       container.Image.ValueString(),
			Command:     extractStringList(container.Command),
			Entrypoint:  extractStringList(container.Entrypoint),
			Envs:        extractStringMap(container.Environment),
			Ports:       extractPortMappings(ctx, container.Ports, d),
			Volumes:     extractVolumeMappings(ctx, container.Volumes, d),
			Description: container.Description.ValueString(),
		}

		declareRequest.Body.Services[name] = service
	}

	// Process Volumes
	if !m.Volumes.IsUnknown() && !m.Volumes.IsNull() {
		var volumeModels map[string]VolumeModel
		detailDiags = m.Volumes.ElementsAs(ctx, &volumeModels, false)
		if detailDiags.HasError() {
			d.Append(detailDiags...)
			return nil
		}

		for name := range volumeModels {
			declareRequest.Body.Volumes[name] = containerv2.VolumeDeclareRequest{
				Name: name,
			}
		}
	}

	return declareRequest
}

// extractStringList is a helper to extract a Terraform list of strings into a
// Go slice.
func extractStringList(list types.List) []string {
	var result []string
	if list.IsNull() || list.IsUnknown() {
		return result
	}

	for _, v := range list.Elements() {
		if strVal, ok := v.(types.String); ok {
			result = append(result, strVal.ValueString())
		}
	}
	return result
}

// extractStringMap is a helper to extract a Terraform map of strings into a Go map.
func extractStringMap(m types.Map) map[string]string {
	result := make(map[string]string)
	if m.IsNull() || m.IsUnknown() {
		return result
	}

	for key, v := range m.Elements() {
		if strVal, ok := v.(types.String); ok {
			result[key] = strVal.ValueString()
		}
	}
	return result
}

// extractPortMappings is a helper to extract a Terraform set of ContainerPortModel
// into a slice of port strings.
func extractPortMappings(ctx context.Context, ports types.Set, d *diag.Diagnostics) []string {
	var result []string
	if ports.IsNull() || ports.IsUnknown() {
		return result
	}

	var portModels []ContainerPortModel
	detailDiags := ports.ElementsAs(ctx, &portModels, false)
	if detailDiags.HasError() {
		d.Append(detailDiags...)
		return result
	}

	for _, portModel := range portModels {
		hostPort := portModel.PublicPort.ValueInt32()
		containerPort := portModel.ContainerPort.ValueInt32()
		protocol := portModel.Protocol.ValueString()

		if hostPort == 0 {
			hostPort = containerPort // Default to same as container port if not provided
		}

		result = append(result, fmt.Sprintf("%d:%d/%s", hostPort, containerPort, protocol))
	}
	return result
}

// extractVolumeMappings is a helper to extract a Terraform set of
// ContainerVolumeModel into a slice of volume strings.
func extractVolumeMappings(ctx context.Context, volumes types.Set, d *diag.Diagnostics) []string {
	var result []string
	if volumes.IsNull() || volumes.IsUnknown() {
		return result
	}

	var volumeModels []ContainerVolumeModel
	detailDiags := volumes.ElementsAs(ctx, &volumeModels, false)
	if detailDiags.HasError() {
		d.Append(detailDiags...)
		return result
	}

	for _, volumeModel := range volumeModels {
		if !volumeModel.Volume.IsNull() {
			result = append(result, fmt.Sprintf("%s:%s", volumeModel.Volume.ValueString(), volumeModel.MountPath.ValueString()))
		} else {
			result = append(result, fmt.Sprintf("%s:%s", volumeModel.ProjectPath.ValueString(), volumeModel.MountPath.ValueString()))
		}
	}
	return result
}
