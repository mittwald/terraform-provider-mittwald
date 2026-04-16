package cronjobresource

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	. "github.com/onsi/gomega"
)

func TestCronjobDestinationValidator(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	v := &cronjobDestinationValidator{}

	req := validator.ObjectRequest{
		Path: path.Root("destination"),
		ConfigValue: mustDestinationObject(t, map[string]attr.Value{
			"url":               types.StringNull(),
			"command":           types.ObjectNull(map[string]attr.Type{"interpreter": types.StringType, "path": types.StringType, "parameters": types.ListType{ElemType: types.StringType}}),
			"container_command": types.ListNull(types.StringType),
		}),
	}
	resp := &validator.ObjectResponse{}
	v.ValidateObject(ctx, req, resp)
	g.Expect(resp.Diagnostics.HasError()).To(BeTrue())

	req.ConfigValue = mustDestinationObject(t, map[string]attr.Value{
		"url":               types.StringNull(),
		"command":           types.ObjectNull(map[string]attr.Type{"interpreter": types.StringType, "path": types.StringType, "parameters": types.ListType{ElemType: types.StringType}}),
		"container_command": mustListValue(t, []string{"echo", "ok"}),
	})
	resp = &validator.ObjectResponse{}
	v.ValidateObject(ctx, req, resp)
	g.Expect(resp.Diagnostics.HasError()).To(BeFalse())
}

func mustDestinationObject(t *testing.T, values map[string]attr.Value) types.Object {
	t.Helper()
	obj, diags := types.ObjectValue(
		map[string]attr.Type{
			"url":               types.StringType,
			"command":           types.ObjectType{AttrTypes: map[string]attr.Type{"interpreter": types.StringType, "path": types.StringType, "parameters": types.ListType{ElemType: types.StringType}}},
			"container_command": types.ListType{ElemType: types.StringType},
		},
		values,
	)
	if diags.HasError() {
		t.Fatalf("failed to build destination object: %v", diags)
	}
	return obj
}
