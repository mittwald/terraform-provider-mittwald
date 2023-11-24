package cronjobresource

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.Object = &cronjobDestinationValidator{}

type cronjobDestinationValidator struct {
}

func (c *cronjobDestinationValidator) Description(_ context.Context) string {
	return "validator for cronjob destination"
}

func (c *cronjobDestinationValidator) MarkdownDescription(_ context.Context) string {
	return "This validator checks if either `url` or `command` is set."
}

func (c *cronjobDestinationValidator) ValidateObject(ctx context.Context, request validator.ObjectRequest, response *validator.ObjectResponse) {
	attrs := request.ConfigValue.Attributes()

	hasURL := !attrs["url"].IsUnknown() && !attrs["url"].IsNull()
	hasCommand := !attrs["command"].IsUnknown() && !attrs["command"].IsNull()

	if (!hasURL && !hasCommand) || (hasURL && hasCommand) {
		response.Diagnostics.AddAttributeWarning(request.Path.AtName("destination"), "Only one destination is allowed", "Only one destination is allowed. Either `url` or `command` must be set.")
	}
}
