package projectresource

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	// DiskspaceGBMinimum is the smallest disk size the API accepts for a
	// stand-alone project.
	DiskspaceGBMinimum = 20

	// DiskspaceGBIncrement is the step size the API requires disk sizes for
	// stand-alone projects to be a multiple of.
	DiskspaceGBIncrement = 20
)

var _ resource.ConfigValidator = projectPlacementValidator{}
var _ validator.Int64 = diskspaceValidator{}

// projectPlacementValidator enforces that a project is either placed on an
// existing server or ordered as a stand-alone project, but never both.
type projectPlacementValidator struct{}

func (v projectPlacementValidator) Description(_ context.Context) string {
	return "validates that a project is either placed on a server or ordered stand-alone"
}

func (v projectPlacementValidator) MarkdownDescription(_ context.Context) string {
	return "Validates that either `server_id` or `article_id` (together with `customer_id`) is configured, but not both."
}

func (v projectPlacementValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var serverID, customerID, articleID types.String
	var diskspaceGB types.Int64
	var useFreeTrial types.Bool

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("server_id"), &serverID)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("customer_id"), &customerID)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("article_id"), &articleID)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("diskspace_gb"), &diskspaceGB)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("use_free_trial"), &useFreeTrial)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Presence checks: an unknown value still means the attribute is
	// configured (its value is merely not known yet), so only a null value
	// counts as "not set".
	serverSet := !serverID.IsNull()
	customerSet := !customerID.IsNull()
	articleSet := !articleID.IsNull()

	switch {
	case serverSet && articleSet:
		const summary = "Conflicting project placement"
		const detail = "Only one of `server_id` or `article_id` may be configured: a project is either placed on an existing server, or ordered as a stand-alone project."

		resp.Diagnostics.AddAttributeError(path.Root("server_id"), summary, detail)
		resp.Diagnostics.AddAttributeError(path.Root("article_id"), summary, detail)
	case !serverSet && !articleSet:
		const summary = "Missing project placement"
		const detail = "Either `server_id` must be configured to place the project on an existing server, or `article_id` must be configured to order a stand-alone project."

		resp.Diagnostics.AddAttributeError(path.Root("server_id"), summary, detail)
		resp.Diagnostics.AddAttributeError(path.Root("article_id"), summary, detail)
	}

	if articleSet && !customerSet {
		resp.Diagnostics.AddAttributeError(
			path.Root("customer_id"),
			"Missing customer",
			"`customer_id` must be configured when `article_id` is set, to determine which organization is billed for the stand-alone project.",
		)
	}

	if articleSet && diskspaceGB.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("diskspace_gb"),
			"Missing disk space",
			fmt.Sprintf("`diskspace_gb` must be configured when `article_id` is set, to determine the disk space the stand-alone project is ordered with. It must be at least %d and a multiple of %d.", DiskspaceGBMinimum, DiskspaceGBIncrement),
		)
	}

	if serverSet && customerSet {
		resp.Diagnostics.AddAttributeError(
			path.Root("customer_id"),
			"Conflicting customer",
			"`customer_id` must not be configured together with `server_id`: a project placed on a server is billed via that server's contract, and inherits its customer.",
		)
	}

	// The remaining attributes only make sense for a stand-alone project, which
	// is what `article_id` selects.
	if articleSet {
		return
	}

	if !diskspaceGB.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("diskspace_gb"),
			"Disk space not configurable",
			"`diskspace_gb` can only be set for stand-alone projects (those configured with an `article_id`). A project placed on a server uses that server's disk space.",
		)
	}

	if !useFreeTrial.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("use_free_trial"),
			"Free trial not available",
			"`use_free_trial` can only be set for stand-alone projects (those configured with an `article_id`), because only those are ordered separately.",
		)
	}
}

// diskspaceValidator enforces the disk sizes the API accepts for stand-alone
// projects.
type diskspaceValidator struct{}

func (v diskspaceValidator) Description(_ context.Context) string {
	return fmt.Sprintf("value must be at least %d and a multiple of %d", DiskspaceGBMinimum, DiskspaceGBIncrement)
}

func (v diskspaceValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v diskspaceValidator) ValidateInt64(_ context.Context, req validator.Int64Request, resp *validator.Int64Response) {
	// An unknown value cannot be checked yet; it is validated by the API once
	// it is resolved.
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueInt64()

	if value < DiskspaceGBMinimum {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid disk space",
			fmt.Sprintf("`diskspace_gb` must be at least %d, but was %d.", DiskspaceGBMinimum, value),
		)
		return
	}

	if value%DiskspaceGBIncrement != 0 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid disk space",
			fmt.Sprintf("`diskspace_gb` must be a multiple of %d, but was %d.", DiskspaceGBIncrement, value),
		)
	}
}
