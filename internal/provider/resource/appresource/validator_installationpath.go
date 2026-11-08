package appresource

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// installationPathValidator ensures that a user-provided installation path
// starts with a "/". The API normalizes paths without a leading slash (e.g.
// "foo" becomes "/foo"), which would otherwise cause an "inconsistent result
// after apply" error.
type installationPathValidator struct{}

var _ validator.String = &installationPathValidator{}

func (v *installationPathValidator) Description(_ context.Context) string {
	return `Validates that the installation path starts with a "/".`
}

func (v *installationPathValidator) MarkdownDescription(_ context.Context) string {
	return "Validates that the installation path starts with a `/`."
}

func (v *installationPathValidator) ValidateString(_ context.Context, request validator.StringRequest, response *validator.StringResponse) {
	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}

	value := request.ConfigValue.ValueString()
	if !strings.HasPrefix(value, "/") {
		response.Diagnostics.AddAttributeError(
			request.Path,
			"Invalid installation path",
			fmt.Sprintf(`The installation path must start with a "/" (relative to the web root); use "/%s" instead of "%s".`, value, value),
		)
	}
}
