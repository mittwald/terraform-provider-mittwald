package containerstackresource

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/containerv2"
	"strconv"
	"strings"
)

func (m *ContainerStackModel) FromAPIModel(ctx context.Context, apiModel *containerv2.StackResponse) (res diag.Diagnostics) {
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

		container := ContainerModel{
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

// Define ContainerModelType for schema conversion
var containerModelType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"image":       types.StringType,
		"description": types.StringType,
		"command":     types.ListType{ElemType: types.StringType},
		"entrypoint":  types.ListType{ElemType: types.StringType},
		"environment": types.MapType{ElemType: types.StringType},
		"ports":       types.SetType{ElemType: containerPortModelType},
		"volumes":     types.SetType{ElemType: containerVolumeModelType},
	},
}

// Define ContainerPortModelType for schema conversion
var containerPortModelType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"container_port": types.Int32Type,
		"public_port":    types.Int32Type,
		"protocol":       types.StringType,
	},
}

// Define ContainerVolumeModelType for schema conversion
var containerVolumeModelType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"volume":       types.StringType,
		"project_path": types.StringType,
		"mount_path":   types.StringType,
	},
}

// Define ContainerVolumeModelType for schema conversion
var volumeModelType = types.ObjectType{
	AttrTypes: map[string]attr.Type{},
}

// Convert a slice of strings to a Terraform List of Strings
func convertStringSliceToList(slice []string) types.List {
	values := make([]attr.Value, len(slice))
	for i, v := range slice {
		values[i] = types.StringValue(v)
	}
	list, _ := types.ListValue(types.StringType, values)
	return list
}

// Convert a map of strings to a Terraform Map of Strings
func convertStringMapToMap(m map[string]string) types.Map {
	values := make(map[string]attr.Value, len(m))
	for key, value := range m {
		values[key] = types.StringValue(value)
	}
	result, _ := types.MapValue(types.StringType, values)
	return result
}

// Convert a slice of port strings ("80:8080/tcp") into a Set of ContainerPortModel
func convertPortStringsToSet(ctx context.Context, ports []string, d *diag.Diagnostics) types.Set {
	var portModels []attr.Value
	for _, port := range ports {
		parts := strings.Split(port, "/")
		if len(parts) != 2 {
			d.AddWarning("Invalid port format", fmt.Sprintf("Skipping port: %s", port))
			continue
		}

		var publicPort, containerPort int

		portMapping := strings.Split(parts[0], ":")
		if len(portMapping) == 1 {
			var err error
			containerPort, err = strconv.Atoi(portMapping[0])
			if err != nil {
				d.AddWarning("Invalid port values", fmt.Sprintf("Skipping port: %s", port))
				continue
			}
		} else if len(portMapping) == 2 {
			var err1, err2 error
			publicPort, err1 = strconv.Atoi(portMapping[0])
			containerPort, err2 = strconv.Atoi(portMapping[1])
			if err1 != nil || err2 != nil {
				d.AddWarning("Invalid port values", fmt.Sprintf("Skipping port: %s", port))
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

// Convert a slice of volume strings ("/project/path:/container/path") into a Set of ContainerVolumeModel
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
