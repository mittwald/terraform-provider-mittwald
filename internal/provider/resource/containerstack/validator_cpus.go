package containerstackresource

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.Float64 = &CpusValidator{}

// CpusValidator validates that the CPU value is greater than 0.
type CpusValidator struct{}

func (c *CpusValidator) Description(_ context.Context) string {
	return "Asserts that the CPU limit is a positive number."
}

func (c *CpusValidator) MarkdownDescription(ctx context.Context) string {
	return c.Description(ctx)
}

func (c *CpusValidator) ValidateFloat64(_ context.Context, request validator.Float64Request, response *validator.Float64Response) {
	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}

	cpus := request.ConfigValue.ValueFloat64()

	if cpus <= 0 {
		response.Diagnostics.AddAttributeError(
			request.Path,
			"Invalid CPU Limit",
			"The CPU limit must be greater than 0.",
		)
	}
}
