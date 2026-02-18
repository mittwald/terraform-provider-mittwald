package containerstackresource

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/containerv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/valueutil"
)

func (m *ContainerStackModel) FromAPIModel(ctx context.Context, apiModel *containerv2.StackResponse, plan *ContainerStackModel, disregardUnknown bool) (res diag.Diagnostics) {
	// Assign top-level attributes
	m.ID = types.StringValue(apiModel.Id)
	m.ProjectID = types.StringValue(apiModel.ProjectId)
	m.DefaultStack = types.BoolValue(apiModel.Description == "default")

	// Convert Services to Containers using pendingState
	containerMap := make(map[string]attr.Value)
	for _, service := range apiModel.Services {
		state := service.PendingState

		// Normalize image name by removing the "library/" prefix. For the Plan,
		// the same thing is achieved by the StripLibraryPrefixFromImage modifier.
		image := strings.TrimPrefix(state.Image, "library/")

		_, hasExisting := plan.Containers.Elements()[service.ServiceName]

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
			Command:     valueutil.ConvertStringSliceToList(state.Command),
			Entrypoint:  valueutil.ConvertStringSliceToList(state.Entrypoint),
			Environment: convertStringMapToMap(state.Envs),
			Ports:       convertPortStringsToSet(ctx, state.Ports, &res),
			Volumes:     convertVolumeStringsToSet(ctx, state.Volumes, &res),
			Limits:      convertLimitsToObject(ctx, service.Deploy, &res),
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
		_, hasExisting := plan.Volumes.Elements()[volume.Name]

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
	containers, mapContainersRes := types.MapValue(containerModelType, containerMap)
	res.Append(mapContainersRes...)

	volumes, mapVolumesRes := types.MapValue(volumeModelType, volumeMap)
	res.Append(mapVolumesRes...)

	m.Containers = containers
	m.Volumes = volumes

	return res
}

var containerModelType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"id":                    types.StringType,
		"image":                 types.StringType,
		"description":           types.StringType,
		"command":               types.ListType{ElemType: types.StringType},
		"entrypoint":            types.ListType{ElemType: types.StringType},
		"environment":           types.MapType{ElemType: types.StringType},
		"ports":                 types.SetType{ElemType: containerPortModelType},
		"volumes":               types.SetType{ElemType: containerVolumeModelType},
		"limits":                containerLimitsModelType,
		"no_recreate_on_change": types.BoolType,
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

var containerLimitsModelType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"cpus":   types.Float64Type,
		"memory": types.StringType,
	},
}

var volumeModelType = types.ObjectType{
	AttrTypes: map[string]attr.Type{},
}

func convertStringMapToMap(m map[string]string) types.Map {
	values := make(map[string]attr.Value, len(m))
	for key, value := range m {
		values[key] = types.StringValue(value)
	}
	result, _ := types.MapValue(types.StringType, values)
	return result
}

// ParsePortString parses a port string in the format "public_port:container_port/protocol"
// or "container_port/protocol" into a ContainerPortModel.
func ParsePortString(portStr string) (ContainerPortModel, error) {
	parts := strings.Split(portStr, "/")
	if len(parts) != 2 {
		return ContainerPortModel{}, fmt.Errorf("invalid port format: expected 'port/protocol', got '%s'", portStr)
	}

	var publicPort, containerPort int64

	portMapping := strings.Split(parts[0], ":")
	if len(portMapping) == 1 {
		var err error
		containerPort, err = strconv.ParseInt(portMapping[0], 10, 32)
		if err != nil || containerPort <= 0 || containerPort > math.MaxInt32 {
			return ContainerPortModel{}, fmt.Errorf("invalid port value: %s", portStr)
		}
	} else if len(portMapping) == 2 {
		var err1, err2 error
		publicPort, err1 = strconv.ParseInt(portMapping[0], 10, 32)
		containerPort, err2 = strconv.ParseInt(portMapping[1], 10, 32)
		if err1 != nil || publicPort <= 0 || publicPort > math.MaxInt32 {
			return ContainerPortModel{}, fmt.Errorf("invalid public port: %s", portStr)
		}
		if err2 != nil || containerPort <= 0 || containerPort > math.MaxInt32 {
			return ContainerPortModel{}, fmt.Errorf("invalid container port: %s", portStr)
		}
	} else {
		return ContainerPortModel{}, fmt.Errorf("invalid port mapping: expected 'port' or 'public:container', got '%s'", portStr)
	}

	portModel := ContainerPortModel{
		PublicPort:    types.Int32Value(int32(containerPort)),
		ContainerPort: types.Int32Value(int32(containerPort)),
		Protocol:      types.StringValue(parts[1]),
	}

	if publicPort != 0 {
		portModel.PublicPort = types.Int32Value(int32(publicPort))
	}

	return portModel, nil
}

// convertPortStringsToSet converts a slice of port strings ("80:8080/tcp") into
// a Set of ContainerPortModel.
func convertPortStringsToSet(ctx context.Context, ports []string, d *diag.Diagnostics) types.Set {
	var portModels []attr.Value
	for _, port := range ports {
		portModel, err := ParsePortString(port)
		if err != nil {
			d.AddWarning("Invalid port format", fmt.Sprintf("Skipping port %s: %s", port, err.Error()))
			continue
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

// convertLimitsToObject converts the Deploy.Resources.Limits from the API into a Terraform object.
func convertLimitsToObject(ctx context.Context, deploy *containerv2.Deploy, d *diag.Diagnostics) types.Object {
	if deploy == nil || deploy.Resources == nil || deploy.Resources.Limits == nil {
		return types.ObjectNull(containerLimitsModelType.AttrTypes)
	}

	limits := deploy.Resources.Limits
	limitsModel := ContainerLimitsModel{
		Cpus:   types.Float64Null(),
		Memory: types.StringNull(),
	}

	if limits.Cpus != nil {
		// Parse the CPU string to a float64
		var cpus float64
		_, err := fmt.Sscanf(*limits.Cpus, "%f", &cpus)
		if err != nil {
			d.AddWarning("Invalid CPU format", fmt.Sprintf("Failed to parse CPU value '%s': %s", *limits.Cpus, err.Error()))
		} else {
			limitsModel.Cpus = types.Float64Value(cpus)
		}
	}

	if limits.Memory != nil {
		limitsModel.Memory = types.StringValue(*limits.Memory)
	}

	obj, diags := types.ObjectValueFrom(ctx, containerLimitsModelType.AttrTypes, limitsModel)
	d.Append(diags...)
	return obj
}
