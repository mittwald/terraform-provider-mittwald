package sshuserresource

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type rfc3339Validator struct{}

func (v *rfc3339Validator) Description(_ context.Context) string {
	return "Validates that the value is a valid RFC3339 timestamp."
}

func (v *rfc3339Validator) MarkdownDescription(_ context.Context) string {
	return "Validates that the value is a valid RFC3339 timestamp."
}

func (v *rfc3339Validator) ValidateString(_ context.Context, request validator.StringRequest, response *validator.StringResponse) {
	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}

	_, err := time.Parse(time.RFC3339, request.ConfigValue.ValueString())
	if err != nil {
		response.Diagnostics.AddAttributeError(
			request.Path,
			"Invalid RFC3339 timestamp",
			"The value must be a valid RFC3339 timestamp (e.g., 2024-12-31T23:59:59Z). Error: "+err.Error(),
		)
	}
}
