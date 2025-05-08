package containerstackresource

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ validator.Object = &VolumeMountValidator{}

// VolumeMountValidator is a validator that asserts that either volume or
// project_path is set on a container definition, but not both.
type VolumeMountValidator struct{}

func (v *VolumeMountValidator) Description(_ context.Context) string {
	return "Asserts that either volume or project_path is set, but not both."
}

func (v *VolumeMountValidator) MarkdownDescription(_ context.Context) string {
	return "Asserts that either `volume` or `project_path` is set, but not both."
}

func (v *VolumeMountValidator) ValidateObject(_ context.Context, request validator.ObjectRequest, response *validator.ObjectResponse) {
	attrs := request.ConfigValue.Attributes()

	volume, volumeIsString := attrs["volume"].(types.String)
	projectPath, projectPathIsString := attrs["project_path"].(types.String)

	if !volumeIsString {
		response.Diagnostics.AddError("Invalid Type", "volume must be a string")
	}

	if !projectPathIsString {
		response.Diagnostics.AddError("Invalid Type", "project_path must be a string")
	}

	if response.Diagnostics.HasError() {
		return
	}

	if volume.IsNull() && projectPath.IsNull() {
		response.Diagnostics.AddError("Invalid Value", "Either volume or project_path must be set")
	}

	if !volume.IsNull() && !projectPath.IsNull() {
		response.Diagnostics.AddError("Invalid Value", "Only one of volume or project_path can be set")
	}
}
