package containerstackresource

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = &MemoryValidator{}

// memoryPattern matches Docker Compose memory formats according to the specification.
// Valid formats: {amount}{byte unit} where units are: b (bytes), k or kb (kilo bytes),
// m or mb (mega bytes), and g or gb (giga bytes). All suffixes must be lowercase.
// A suffix is required - plain numbers without a suffix are not valid.
// Reference: https://docs.docker.com/reference/compose-file/extension/#specifying-byte-values
var memoryPattern = regexp.MustCompile(`^[0-9]+(b|kb?|mb?|gb?)$`)

// MemoryValidator validates that the memory value follows Docker Compose format.
type MemoryValidator struct{}

func (m *MemoryValidator) Description(_ context.Context) string {
	return "Asserts that the memory limit follows Docker Compose format (e.g., \"512mb\", \"1gb\", \"50m\")."
}

func (m *MemoryValidator) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m *MemoryValidator) ValidateString(_ context.Context, request validator.StringRequest, response *validator.StringResponse) {
	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}

	memory := request.ConfigValue.ValueString()

	if !memoryPattern.MatchString(memory) {
		response.Diagnostics.AddAttributeError(
			request.Path,
			"Invalid Memory Format",
			"The memory limit must follow Docker Compose format (e.g., \"512mb\", \"1gb\", \"50m\"). Valid suffixes are b (bytes), k or kb (kilo bytes), m or mb (mega bytes), and g or gb (giga bytes).",
		)
	}
}
