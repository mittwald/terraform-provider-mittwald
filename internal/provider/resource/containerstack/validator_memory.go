package containerstackresource

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = &MemoryValidator{}

// memoryPattern matches Docker-style memory formats like "50M", "1G", "512M", "2G", etc.
// Accepts both uppercase and lowercase suffixes (K/k, M/m, G/g, T/t).
var memoryPattern = regexp.MustCompile(`^[1-9][0-9]*[KMGTkmgt]?$`)

// MemoryValidator validates that the memory value follows Docker format.
type MemoryValidator struct{}

func (m *MemoryValidator) Description(_ context.Context) string {
	return "Asserts that the memory limit follows Docker format (e.g., \"50M\", \"1G\", \"512M\")."
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
			"The memory limit must follow Docker format (e.g., \"50M\", \"1G\", \"512M\"). Valid suffixes are K, M, G, T (uppercase or lowercase).",
		)
	}
}
