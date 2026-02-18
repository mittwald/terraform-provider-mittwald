package containerstackresource

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/containerclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/containerv2"
	"strconv"
)

func (m *ContainerStackModel) ToDeletePatchRequest(ctx context.Context, d *diag.Diagnostics) *containerclientv2.UpdateStackRequest {
	updateRequest := &containerclientv2.UpdateStackRequest{
		StackID: m.ID.ValueString(),
		Body: containerclientv2.UpdateStackRequestBody{
			Services: make(map[string]containerv2.ServiceRequest),
			Volumes:  make(map[string]containerv2.VolumeRequest),
		},
	}

	for name := range m.Containers.Elements() {
		// empty object means "delete this container"
		updateRequest.Body.Services[name] = containerv2.ServiceRequest{}
	}

	for name := range m.Volumes.Elements() {
		// empty object means "delete this volume"
		updateRequest.Body.Volumes[name] = containerv2.VolumeRequest{}
	}

	return updateRequest
}

func (m *ContainerStackModel) ToUpdateRequest(ctx context.Context, current *ContainerStackModel, d *diag.Diagnostics) *containerclientv2.UpdateStackRequest {
	updateRequest := &containerclientv2.UpdateStackRequest{
		StackID: m.ID.ValueString(),
		Body: containerclientv2.UpdateStackRequestBody{
			Services: make(map[string]containerv2.ServiceRequest),
			Volumes:  make(map[string]containerv2.VolumeRequest),
		},
	}

	plannedContainers, currentContainers := m.ContainerModels(ctx, d), current.ContainerModels(ctx, d)

	if d.HasError() {
		return nil
	}

	// Find containers and volumes that are present in the current state, but *not* in the
	// plan. Delete these containers and volumes by setting them to an empty object.
	for name := range currentContainers {
		if _, ok := plannedContainers[name]; !ok {
			// empty object means "delete this container"
			updateRequest.Body.Services[name] = containerv2.ServiceRequest{}
		}
	}

	for name, plannedContainer := range plannedContainers {
		currentContainer, hasCurrentContainer := currentContainers[name]

		containerIsUnmodified := hasCurrentContainer && plannedContainer.Equals(&currentContainer)
		containerIsNew := !hasCurrentContainer

		if containerIsUnmodified {
			// no action needed, do nothing
			continue
		}

		if containerIsNew {
			updateRequest.Body.Services[name] = plannedContainer.ToUpdateRequestFromEmpty(ctx, d)
			continue
		}

		updateRequest.Body.Services[name] = plannedContainer.ToUpdateRequestFromExisting(ctx, &currentContainer, d)
	}

	if !m.Volumes.IsUnknown() {
		plannedVolumes, currentVolumes := m.VolumeModels(ctx, d), current.VolumeModels(ctx, d)
		if d.HasError() {
			return nil
		}

		for name := range currentVolumes {
			if _, ok := plannedVolumes[name]; !ok {
				// empty object means "delete this volume"
				updateRequest.Body.Volumes[name] = containerv2.VolumeRequest{}
			}
		}

		for name := range plannedVolumes {
			_, hasCurrentVolume := currentVolumes[name]

			volumeIsNew := !hasCurrentVolume

			if volumeIsNew {
				updateRequest.Body.Volumes[name] = containerv2.VolumeRequest{
					Name: &name,
				}
			}
		}
	}

	return updateRequest
}

func (m *ContainerStackModel) ToDeclareRequest(ctx context.Context, d *diag.Diagnostics) *containerclientv2.DeclareStackRequest {
	declareRequest := &containerclientv2.DeclareStackRequest{
		StackID: m.ID.ValueString(),
		Body: containerclientv2.DeclareStackRequestBody{
			Services: make(map[string]containerv2.ServiceDeclareRequest),
			Volumes:  make(map[string]containerv2.VolumeDeclareRequest),
		},
	}

	containerModels := m.ContainerModels(ctx, d)
	if d.HasError() {
		return nil
	}

	for name, container := range containerModels {
		declareRequest.Body.Services[name] = container.ToDeclareRequest(ctx, d)
	}

	// Process Volumes
	if !m.Volumes.IsUnknown() && !m.Volumes.IsNull() {
		volumeModels := m.VolumeModels(ctx, d)
		if d.HasError() {
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

func (m *ContainerStackModel) ContainerNames() []string {
	var names []string
	if m.Containers.IsNull() || m.Containers.IsUnknown() {
		return names
	}

	for _, v := range m.Containers.Elements() {
		if strVal, ok := v.(types.String); ok {
			names = append(names, strVal.ValueString())
		}
	}
	return names
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

// extractDeploy converts the Terraform limits object to the API Deploy structure.
func extractDeploy(ctx context.Context, limits types.Object, d *diag.Diagnostics) *containerv2.Deploy {
	if limits.IsNull() || limits.IsUnknown() {
		return nil
	}

	var limitsModel ContainerLimitsModel
	diags := limits.As(ctx, &limitsModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		d.Append(diags...)
		return nil
	}

	// Only create the Deploy object if at least one limit is set
	if limitsModel.Cpus.IsNull() && limitsModel.Memory.IsNull() {
		return nil
	}

	deploy := &containerv2.Deploy{
		Resources: &containerv2.Resources{
			Limits: &containerv2.ResourceSpec{},
		},
	}

	if !limitsModel.Cpus.IsNull() {
		cpusStr := strconv.FormatFloat(limitsModel.Cpus.ValueFloat64(), 'f', -1, 64)
		deploy.Resources.Limits.Cpus = &cpusStr
	}

	if !limitsModel.Memory.IsNull() {
		memory := limitsModel.Memory.ValueString()
		deploy.Resources.Limits.Memory = &memory
	}

	return deploy
}
