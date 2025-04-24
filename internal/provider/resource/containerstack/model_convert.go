package containerstackresource

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func (m *ContainerStackModel) ContainerModels(ctx context.Context, d *diag.Diagnostics) map[string]ContainerModel {
	if m.Containers.IsNull() {
		return nil
	}

	containerModels := make(map[string]ContainerModel)
	d.Append(m.Containers.ElementsAs(ctx, &containerModels, false)...)

	return containerModels
}

func (m *ContainerStackModel) VolumeModels(ctx context.Context, d *diag.Diagnostics) map[string]VolumeModel {
	if m.Volumes.IsNull() {
		return nil
	}

	volumeModels := make(map[string]VolumeModel)
	d.Append(m.Volumes.ElementsAs(ctx, &volumeModels, false)...)

	return volumeModels
}
