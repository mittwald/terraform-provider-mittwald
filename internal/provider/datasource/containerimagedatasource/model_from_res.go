package containerimagedatasource

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/containerv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/valueutil"
)

var envAttrTypes = map[string]attr.Type{
	"key":          types.StringType,
	"value":        types.StringType,
	"description":  types.StringType,
	"is_sensitive": types.BoolType,
}

var exposedPortAttrTypes = map[string]attr.Type{
	"port":        types.StringType,
	"description": types.StringType,
}

var volumeAttrTypes = map[string]attr.Type{
	"volume":      types.StringType,
	"description": types.StringType,
}

func (c *ContainerImageDataSourceModel) FromAPIModel(m *containerv2.ContainerImageConfig) (res diag.Diagnostics) {
	c.Command = valueutil.ConvertStringSliceToList(m.Command)
	c.Entrypoint = valueutil.ConvertStringSliceToList(m.Entrypoint)
	c.Digest = types.StringValue(m.Digest)
	c.User = types.StringValue(m.User)
	c.UserID = types.Int64Value(m.UserId)
	c.IsUserRoot = types.BoolValue(m.IsUserRoot)

	envValues := make([]attr.Value, 0, len(m.Env))
	for _, env := range m.Env {
		obj, d := types.ObjectValue(envAttrTypes, map[string]attr.Value{
			"key":          types.StringValue(env.Key),
			"value":        valueutil.StringPtrOrNull(env.Value),
			"description":  valueutil.StringPtrOrNull(env.Description),
			"is_sensitive": valueutil.BoolPtrOrNull(env.IsSensitive),
		})
		res.Append(d...)
		envValues = append(envValues, obj)
	}
	c.Env, res = types.ListValue(types.ObjectType{AttrTypes: envAttrTypes}, envValues)

	portValues := make([]attr.Value, 0, len(m.ExposedPorts))
	for _, port := range m.ExposedPorts {
		obj, d := types.ObjectValue(exposedPortAttrTypes, map[string]attr.Value{
			"port":        types.StringValue(port.Port),
			"description": valueutil.StringPtrOrNull(port.Description),
		})
		res.Append(d...)
		portValues = append(portValues, obj)
	}
	c.ExposedPorts, res = types.ListValue(types.ObjectType{AttrTypes: exposedPortAttrTypes}, portValues)

	volumeValues := make([]attr.Value, 0, len(m.Volumes))
	for _, volume := range m.Volumes {
		obj, d := types.ObjectValue(volumeAttrTypes, map[string]attr.Value{
			"volume":      types.StringValue(volume.Volume),
			"description": valueutil.StringPtrOrNull(volume.Description),
		})
		res.Append(d...)
		volumeValues = append(volumeValues, obj)
	}
	c.Volumes, res = types.ListValue(types.ObjectType{AttrTypes: volumeAttrTypes}, volumeValues)

	return
}
