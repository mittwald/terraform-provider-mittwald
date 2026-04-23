package cronjobresource

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var _ validator.Object = &cronjobDestinationValidator{}
var _ resource.ConfigValidator = cronjobTargetDestinationValidator{}

type cronjobDestinationValidator struct {
}

func (c *cronjobDestinationValidator) Description(_ context.Context) string {
	return "validator for cronjob destination"
}

func (c *cronjobDestinationValidator) MarkdownDescription(_ context.Context) string {
	return "This validator checks if exactly one of `url`, `command`, or `container_command` is set."
}

func (c *cronjobDestinationValidator) ValidateObject(ctx context.Context, request validator.ObjectRequest, response *validator.ObjectResponse) {
	attrs := request.ConfigValue.Attributes()

	urlAttr, hasURLAttr := attrs["url"]
	commandAttr, hasCommandAttr := attrs["command"]
	containerCommandAttr, hasContainerCommandAttr := attrs["container_command"]

	hasURL := hasURLAttr && !urlAttr.IsUnknown() && !urlAttr.IsNull()
	hasCommand := hasCommandAttr && !commandAttr.IsUnknown() && !commandAttr.IsNull()
	hasContainerCommand := hasContainerCommandAttr && !containerCommandAttr.IsUnknown() && !containerCommandAttr.IsNull()

	count := 0
	if hasURL {
		count++
	}
	if hasCommand {
		count++
	}
	if hasContainerCommand {
		count++
	}

	if count != 1 {
		if count == 0 {
			response.Diagnostics.AddAttributeError(request.Path, "Missing cronjob destination", "One of `destination.url`, `destination.command`, or `destination.container_command` must be set.")
			return
		}

		response.Diagnostics.AddAttributeError(request.Path, "Multiple cronjob destinations configured", "Only one of `destination.url`, `destination.command`, or `destination.container_command` can be set.")
	}
}

type cronjobTargetDestinationValidator struct{}

func (v cronjobTargetDestinationValidator) Description(_ context.Context) string {
	return "validates valid app/container target and destination combinations"
}

func (v cronjobTargetDestinationValidator) MarkdownDescription(_ context.Context) string {
	return "Validates that `app_id` is paired with `destination.url`/`destination.command`, and `container` is paired with `destination.container_command`."
}

func (v cronjobTargetDestinationValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var appID types.String
	var container types.Object
	var destination types.Object
	var destinationModel ResourceDestinationModel

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("app_id"), &appID)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("container"), &container)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("destination"), &destination)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appSet := !appID.IsNull() && !appID.IsUnknown()
	containerSet := !container.IsNull() && !container.IsUnknown()

	if !destination.IsNull() && !destination.IsUnknown() {
		resp.Diagnostics.Append(destination.As(ctx, &destinationModel, basetypes.ObjectAsOptions{})...)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	hasURL := !destinationModel.URL.IsNull() && !destinationModel.URL.IsUnknown()
	hasCommand := !destinationModel.Command.IsNull() && !destinationModel.Command.IsUnknown()
	hasContainerCommand := !destinationModel.ContainerCommand.IsNull() && !destinationModel.ContainerCommand.IsUnknown()

	if appSet && containerSet {
		resp.Diagnostics.AddAttributeError(path.Root("app_id"), "Invalid target configuration", "Only one of `app_id` or `container` may be configured.")
		resp.Diagnostics.AddAttributeError(path.Root("container"), "Invalid target configuration", "Only one of `app_id` or `container` may be configured.")
	}

	if !appSet && !containerSet {
		resp.Diagnostics.AddAttributeError(path.Root("app_id"), "Missing target configuration", "Either `app_id` or `container` must be configured.")
		resp.Diagnostics.AddAttributeError(path.Root("container"), "Missing target configuration", "Either `app_id` or `container` must be configured.")
	}

	if containerSet && !hasContainerCommand {
		resp.Diagnostics.AddAttributeError(path.Root("destination").AtName("container_command"), "Missing container command destination", "`destination.container_command` must be configured when `container` is set.")
	}

	if !containerSet && hasContainerCommand {
		resp.Diagnostics.AddAttributeError(path.Root("container"), "Missing container target", "`container` must be configured when `destination.container_command` is set.")
	}

	if appSet && !(hasURL || hasCommand) {
		resp.Diagnostics.AddAttributeError(path.Root("destination"), "Missing app destination", "When `app_id` is set, `destination.url` or `destination.command` must be configured.")
	}

	if !appSet && (hasURL || hasCommand) {
		resp.Diagnostics.AddAttributeError(path.Root("app_id"), "Missing app target", "`app_id` must be configured when `destination.url` or `destination.command` is set.")
	}
}
