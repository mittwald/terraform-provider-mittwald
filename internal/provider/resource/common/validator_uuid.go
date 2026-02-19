package common

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = &UUIDValidator{}

// uuidPattern matches valid UUID format (case-insensitive).
var uuidPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// shortIDPatterns match the various short ID formats (case-insensitive).
var shortIDPatterns = map[string]*regexp.Regexp{
	"server":    regexp.MustCompile(`(?i)^s-[a-z0-9]{6}$`),
	"project":   regexp.MustCompile(`(?i)^p-[a-z0-9]{6}$`),
	"app":       regexp.MustCompile(`(?i)^a-[a-z0-9]{6}$`),
	"container": regexp.MustCompile(`(?i)^c-[a-z0-9]{6}$`),
}

// UUIDValidator validates that the value is a valid UUID and not a short ID.
type UUIDValidator struct{}

func (u *UUIDValidator) Description(_ context.Context) string {
	return "Validates that the value is a full UUID (not a short ID like s-XXXXXX, p-XXXXXX, a-XXXXXX, or c-XXXXXX)."
}

func (u *UUIDValidator) MarkdownDescription(ctx context.Context) string {
	return u.Description(ctx)
}

func (u *UUIDValidator) ValidateString(_ context.Context, request validator.StringRequest, response *validator.StringResponse) {
	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}

	value := request.ConfigValue.ValueString()

	// Check if it matches a short ID pattern
	for idType, pattern := range shortIDPatterns {
		if pattern.MatchString(value) {
			response.Diagnostics.AddAttributeError(
				request.Path,
				"Short ID Not Supported",
				"The provided value appears to be a short ID ("+value+"), but this field requires a full UUID.\n\n"+
					"Short IDs (like s-XXXXXX, p-XXXXXX, a-XXXXXX, c-XXXXXX) are displayed in the mStudio UI for convenience, "+
					"but the Terraform provider requires the full UUID returned by the mStudio API.\n\n"+
					"To get the full UUID for a "+idType+":\n"+
					"  1. Use the mStudio API directly\n"+
					"  2. Use a data source (e.g., mittwald_project_by_shortid) to look up the UUID from a short ID\n"+
					"  3. Use the ID from another Terraform resource's output",
			)
			return
		}
	}

	// Verify it's a valid UUID
	if !uuidPattern.MatchString(value) {
		response.Diagnostics.AddAttributeError(
			request.Path,
			"Invalid UUID Format",
			"The provided value ("+value+") is not a valid UUID. Expected format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
		)
	}
}
