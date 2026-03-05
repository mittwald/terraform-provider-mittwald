package tlscertificateresource

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// certImportFieldsValidator enforces two rules:
//  1. `certificate`, `private_key_wo`, and `private_key_wo_version` must all be
//     set together, or not at all (they form a required group for certificate import).
//  2. When none of those fields are set (DNS-validation mode), `common_name` MUST
//     be provided.
type certImportFieldsValidator struct{}

func (v certImportFieldsValidator) Description(_ context.Context) string {
	return "Validates that certificate import fields are all set together, and that " +
		"common_name is provided when not using certificate import."
}

func (v certImportFieldsValidator) MarkdownDescription(_ context.Context) string {
	return "Validates that `certificate`, `private_key_wo`, and `private_key_wo_version` " +
		"are either all set or all unset, and that `common_name` is required when " +
		"none of the import fields are set."
}

func (v certImportFieldsValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var certificate types.String
	var privateKeyWO types.String
	var privateKeyWOVersion types.Int64
	var commonName types.String

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("certificate"), &certificate)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("private_key_wo"), &privateKeyWO)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("private_key_wo_version"), &privateKeyWOVersion)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("common_name"), &commonName)...)
	if resp.Diagnostics.HasError() {
		return
	}

	certSet := !certificate.IsNull() && !certificate.IsUnknown()
	keySet := !privateKeyWO.IsNull() && !privateKeyWO.IsUnknown()
	versionSet := !privateKeyWOVersion.IsNull() && !privateKeyWOVersion.IsUnknown()

	importFieldCount := 0
	if certSet {
		importFieldCount++
	}
	if keySet {
		importFieldCount++
	}
	if versionSet {
		importFieldCount++
	}

	// Rule 1: all-or-nothing for the import group.
	if importFieldCount > 0 && importFieldCount < 3 {
		if !certSet {
			resp.Diagnostics.AddAttributeError(
				path.Root("certificate"),
				"Missing certificate",
				"When importing a certificate, all three of `certificate`, `private_key_wo`, and `private_key_wo_version` must be set.",
			)
		}
		if !keySet {
			resp.Diagnostics.AddAttributeError(
				path.Root("private_key_wo"),
				"Missing private_key_wo",
				"When importing a certificate, all three of `certificate`, `private_key_wo`, and `private_key_wo_version` must be set.",
			)
		}
		if !versionSet {
			resp.Diagnostics.AddAttributeError(
				path.Root("private_key_wo_version"),
				"Missing private_key_wo_version",
				"When importing a certificate, all three of `certificate`, `private_key_wo`, and `private_key_wo_version` must be set.",
			)
		}
		return
	}

	// Rule 2: when not importing, common_name is required (DNS validation mode).
	if importFieldCount == 0 {
		if commonName.IsNull() || commonName.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("common_name"),
				"Missing common_name",
				"When not importing a certificate (DNS validation mode), `common_name` must be set.",
			)
		}
	}
}
