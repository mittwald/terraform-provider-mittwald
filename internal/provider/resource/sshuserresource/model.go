package sshuserresource

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// ResourceModel describes the resource data model.
type ResourceModel struct {
	ID                types.String `tfsdk:"id"`
	ProjectID         types.String `tfsdk:"project_id"`
	Description       types.String `tfsdk:"description"`
	Username          types.String `tfsdk:"username"`
	Active            types.Bool   `tfsdk:"active"`
	ExpiresAt         types.String `tfsdk:"expires_at"`
	PublicKeys        types.Set    `tfsdk:"public_keys"`
	PasswordWO        types.String `tfsdk:"password_wo"`
	PasswordWOVersion types.Int64  `tfsdk:"password_wo_version"`
	CreatedAt         types.String `tfsdk:"created_at"`
}

// PublicKeyModel describes a single SSH public key.
type PublicKeyModel struct {
	Key     types.String `tfsdk:"key"`
	Comment types.String `tfsdk:"comment"`
}

// PublicKeyAttrTypes returns the attribute types for the public key object.
func PublicKeyAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"key":     types.StringType,
		"comment": types.StringType,
	}
}

// GetPublicKeys extracts the public keys from the model.
func (m *ResourceModel) GetPublicKeys(ctx context.Context, d *diag.Diagnostics) []PublicKeyModel {
	if m.PublicKeys.IsNull() || m.PublicKeys.IsUnknown() {
		return nil
	}

	publicKeys := []PublicKeyModel{}
	d.Append(m.PublicKeys.ElementsAs(ctx, &publicKeys, false)...)
	return publicKeys
}

// SetPublicKeys sets the public keys on the model.
func (m *ResourceModel) SetPublicKeys(ctx context.Context, d *diag.Diagnostics, keys []PublicKeyModel) {
	if keys == nil {
		m.PublicKeys = types.SetNull(types.ObjectType{AttrTypes: PublicKeyAttrTypes()})
		return
	}

	setValue, diags := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: PublicKeyAttrTypes()}, keys)
	d.Append(diags...)
	m.PublicKeys = setValue
}

// AsObject converts the PublicKeyModel to a types.Object.
func (p *PublicKeyModel) AsObject(ctx context.Context, d *diag.Diagnostics) types.Object {
	obj, diags := types.ObjectValueFrom(ctx, PublicKeyAttrTypes(), p)
	d.Append(diags...)
	return obj
}

// FromObject populates the PublicKeyModel from a types.Object.
func (p *PublicKeyModel) FromObject(ctx context.Context, obj types.Object, d *diag.Diagnostics) {
	d.Append(obj.As(ctx, p, basetypes.ObjectAsOptions{})...)
}
