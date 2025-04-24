package containerstackresource

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/containerv2"
	"math"
	"strconv"
	"strings"
)

func (m *ContainerStackModel) FromAPIModel(ctx context.Context, apiModel *containerv2.StackResponse, disregardUnknown bool) (res diag.Diagnostics) {
	// Assign top-level attributes
	m.ID = types.StringValue(apiModel.Id)
	m.ProjectID = types.StringValue(apiModel.ProjectId)
	m.DefaultStack = types.BoolValue(apiModel.Description == "default")

	// Convert Services to Containers using pendingState
	containerMap := make(map[string]attr.Value)
	for _, service := range apiModel.Services {
		state := service.PendingState

		image := state.Image

		existing, hasExisting := m.Containers.Elements()[service.ServiceName]
		if hasExisting {
			// If the image is the same image as defined in the state, except
			// for the "library/" prefix, use the existing image
			existingImage, ok := imageFromContainerObject(existing)
			if ok {
				if strings.TrimPrefix(image, "library/") == existingImage {
					image = existingImage
				}
			}
		}

		// Disregard unmanaged containers in the default stack; these might be
		// managed by other means (e.g. Docker Compose, or another Terraform resource).
		if !hasExisting && m.DefaultStack.ValueBool() && disregardUnknown {
			tflog.Debug(ctx, "disregarding unmanaged container in default stack", map[string]any{"name": service.ServiceName})
			continue
		}

		container := ContainerModel{
			ID:          types.StringValue(service.Id),
			Image:       types.StringValue(image),
			Description: types.StringValue(service.Description),
			Command:     convertStringSliceToList(state.Command),
			Entrypoint:  convertStringSliceToList(state.Entrypoint),
			Environment: convertStringMapToMap(state.Envs),
			Ports:       convertPortStringsToSet(ctx, state.Ports, &res),
			Volumes:     convertVolumeStringsToSet(ctx, state.Volumes, &res),
		}

		containerVal, diags := types.ObjectValueFrom(ctx, containerModelType.AttrTypes, container)
		res.Append(diags...)
		if diags.HasError() {
			continue
		}

		containerMap[service.ServiceName] = containerVal
	}

	// Convert Volumes
	volumeMap := make(map[string]attr.Value)
	for _, volume := range apiModel.Volumes {
		_, hasExisting := m.Volumes.Elements()[volume.Name]

		if !hasExisting && m.DefaultStack.ValueBool() && disregardUnknown {
			tflog.Debug(ctx, "disregarding unmanaged volume in default stack", map[string]any{"name": volume.Name})
			continue
		}

		volumeModel := VolumeModel{}

		volumeVal, diags := types.ObjectValueFrom(ctx, volumeModelType.AttrTypes, volumeModel)
		res.Append(diags...)
		if diags.HasError() {
			continue
		}

		volumeMap[volume.Name] = volumeVal
	}

	// Assign transformed values
	containers, mapContainersRes := types.MapValue(types.ObjectType{AttrTypes: containerModelType.AttrTypes}, containerMap)
	res.Append(mapContainersRes...)

	volumes, mapVolumesRes := types.MapValue(types.ObjectType{AttrTypes: volumeModelType.AttrTypes}, volumeMap)
	res.Append(mapVolumesRes...)

	m.Containers = containers
	m.Volumes = volumes

	return res
}

func imageFromContainerObject(val attr.Value) (string, bool) {
	obj, ok := val.(types.Object)
	if !ok {
		return "", false
	}

	imgVal, ok := obj.Attributes()["image"]
	if !ok {
		return "", false
	}

	imgStr, ok := imgVal.(types.String)
	if !ok {
		return "", false
	}

	return imgStr.ValueString(), true
}

var containerModelType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"id":          types.StringType,
		"image":       types.StringType,
		"description": types.StringType,
		"command":     types.ListType{ElemType: types.StringType},
		"entrypoint":  types.ListType{ElemType: types.StringType},
		"environment": types.MapType{ElemType: types.StringType},
		"ports":       types.SetType{ElemType: containerPortModelType},
		"volumes":     types.SetType{ElemType: containerVolumeModelType},
	},
}

var containerPortModelType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"container_port": types.Int32Type,
		"public_port":    types.Int32Type,
		"protocol":       types.StringType,
	},
}

var containerVolumeModelType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"volume":       types.StringType,
		"project_path": types.StringType,
		"mount_path":   types.StringType,
	},
}

var volumeModelType = types.ObjectType{
	AttrTypes: map[string]attr.Type{},
}

func convertStringSliceToList(slice []string) types.List {
	values := make([]attr.Value, len(slice))
	for i, v := range slice {
		values[i] = types.StringValue(v)
	}
	list, _ := types.ListValue(types.StringType, values)
	return list
}

func convertStringMapToMap(m map[string]string) types.Map {
	values := make(map[string]attr.Value, len(m))
	for key, value := range m {
		values[key] = types.StringValue(value)
	}
	result, _ := types.MapValue(types.StringType, values)
	return result
}

// convertPortStringsToSet converts a slice of port strings ("80:8080/tcp") into
// a Set of ContainerPortModel.
func convertPortStringsToSet(ctx context.Context, ports []string, d *diag.Diagnostics) types.Set {
	var portModels []attr.Value
	for _, port := range ports {
		parts := strings.Split(port, "/")
		if len(parts) != 2 {
			d.AddWarning("Invalid port format", fmt.Sprintf("Skipping port: %s", port))
			continue
		}

		var publicPort, containerPort int64

		portMapping := strings.Split(parts[0], ":")
		if len(portMapping) == 1 {
			var err error
			containerPort, err = strconv.ParseInt(portMapping[0], 10, 32)
			if err != nil || containerPort <= 0 || containerPort > math.MaxInt32 {
				d.AddWarning("Invalid port values", fmt.Sprintf("Skipping port: %s", port))
				continue
			}
		} else if len(portMapping) == 2 {
			var err1, err2 error
			publicPort, err1 = strconv.ParseInt(portMapping[0], 10, 32)
			containerPort, err2 = strconv.ParseInt(portMapping[1], 10, 32)
			if err1 != nil || publicPort <= 0 || publicPort > math.MaxInt32 {
				d.AddWarning("Invalid public port", fmt.Sprintf("Skipping port: %s", port))
				continue
			}
			if err2 != nil || containerPort <= 0 || containerPort > math.MaxInt32 {
				d.AddWarning("Invalid container port", fmt.Sprintf("Skipping port: %s", port))
				continue
			}
		} else {
			d.AddWarning("Invalid port mapping", fmt.Sprintf("Skipping port: %s", port))
			continue
		}

		portModel := ContainerPortModel{
			PublicPort:    types.Int32Value(int32(containerPort)),
			ContainerPort: types.Int32Value(int32(containerPort)),
			Protocol:      types.StringValue(parts[1]),
		}

		if publicPort != 0 {
			portModel.PublicPort = types.Int32Value(int32(publicPort))
		}

		portVal, diags := types.ObjectValueFrom(ctx, containerPortModelType.AttrTypes, portModel)
		d.Append(diags...)
		if diags.HasError() {
			continue
		}

		portModels = append(portModels, portVal)
	}

	set, diags := types.SetValue(types.ObjectType{AttrTypes: containerPortModelType.AttrTypes}, portModels)
	d.Append(diags...)
	return set
}

// convertVolumeStringsToSet convert a slice of volume strings
// ("/project/path:/container/path") into a Set of ContainerVolumeModel.
func convertVolumeStringsToSet(ctx context.Context, volumes []string, d *diag.Diagnostics) types.Set {
	var volumeModels []attr.Value
	for _, volume := range volumes {
		parts := strings.Split(volume, ":")
		if len(parts) != 2 {
			d.AddWarning("Invalid volume format", fmt.Sprintf("Skipping volume: %s", volume))
			continue
		}

		volumeModel := ContainerVolumeModel{
			ProjectPath: types.StringNull(),
			MountPath:   types.StringValue(parts[1]),
			Volume:      types.StringNull(),
		}

		if strings.HasPrefix(parts[0], "/") {
			volumeModel.ProjectPath = types.StringValue(parts[0])
		} else {
			volumeModel.Volume = types.StringValue(parts[0])
		}

		volumeVal, diags := types.ObjectValueFrom(ctx, containerVolumeModelType.AttrTypes, volumeModel)
		d.Append(diags...)
		if diags.HasError() {
			continue
		}

		volumeModels = append(volumeModels, volumeVal)
	}

	set, diags := types.SetValue(types.ObjectType{AttrTypes: containerVolumeModelType.AttrTypes}, volumeModels)
	d.Append(diags...)
	return set
}
