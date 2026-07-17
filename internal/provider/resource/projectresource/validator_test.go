package projectresource

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	. "github.com/onsi/gomega"
)

const (
	someServerID   = "5c1e5b0e-1e1e-4b0e-8e1e-1e1e5b0e1e1e"
	someCustomerID = "9d2f6c1f-2f2f-4c1f-9f2f-2f2f6c1f2f2f"
	someArticleID  = "a-article-id"
)

func TestProjectPlacementValidator(t *testing.T) {
	tests := []struct {
		name      string
		config    map[string]tftypes.Value
		expectErr bool
	}{
		{
			name: "project on a server",
			config: map[string]tftypes.Value{
				"server_id": strVal(someServerID),
			},
		},
		{
			name: "stand-alone project",
			config: map[string]tftypes.Value{
				"customer_id":  strVal(someCustomerID),
				"article_id":   strVal(someArticleID),
				"diskspace_gb": numVal(20),
			},
		},
		{
			name: "stand-alone project with all order options",
			config: map[string]tftypes.Value{
				"customer_id":    strVal(someCustomerID),
				"article_id":     strVal(someArticleID),
				"diskspace_gb":   numVal(40),
				"use_free_trial": boolVal(true),
			},
		},
		{
			name: "article without diskspace",
			config: map[string]tftypes.Value{
				"customer_id": strVal(someCustomerID),
				"article_id":  strVal(someArticleID),
			},
			expectErr: true,
		},
		{
			name: "server and article conflict",
			config: map[string]tftypes.Value{
				"server_id":  strVal(someServerID),
				"article_id": strVal(someArticleID),
			},
			expectErr: true,
		},
		{
			name:      "neither server nor article",
			config:    map[string]tftypes.Value{},
			expectErr: true,
		},
		{
			name: "article without customer",
			config: map[string]tftypes.Value{
				"article_id":   strVal(someArticleID),
				"diskspace_gb": numVal(20),
			},
			expectErr: true,
		},
		{
			name: "server and customer conflict",
			config: map[string]tftypes.Value{
				"server_id":   strVal(someServerID),
				"customer_id": strVal(someCustomerID),
			},
			expectErr: true,
		},
		{
			name: "free trial on a server project",
			config: map[string]tftypes.Value{
				"server_id":      strVal(someServerID),
				"use_free_trial": boolVal(true),
			},
			expectErr: true,
		},
		{
			name: "diskspace on a server project",
			config: map[string]tftypes.Value{
				"server_id":    strVal(someServerID),
				"diskspace_gb": numVal(40),
			},
			expectErr: true,
		},

		// An unknown value means the attribute is configured; its value is
		// merely not known yet. It must never be treated as unset.
		{
			name: "unknown server id alone",
			config: map[string]tftypes.Value{
				"server_id": unknownStr(),
			},
		},
		{
			name: "unknown article id with customer",
			config: map[string]tftypes.Value{
				"customer_id":  strVal(someCustomerID),
				"article_id":   unknownStr(),
				"diskspace_gb": numVal(20),
			},
		},
		{
			name: "unknown server id still conflicts with article",
			config: map[string]tftypes.Value{
				"server_id":    unknownStr(),
				"customer_id":  strVal(someCustomerID),
				"article_id":   strVal(someArticleID),
				"diskspace_gb": numVal(20),
			},
			expectErr: true,
		},
		{
			name: "unknown customer id satisfies the article requirement",
			config: map[string]tftypes.Value{
				"customer_id":  unknownStr(),
				"article_id":   strVal(someArticleID),
				"diskspace_gb": numVal(20),
			},
		},
		{
			name: "unknown diskspace satisfies the article requirement",
			config: map[string]tftypes.Value{
				"customer_id":  strVal(someCustomerID),
				"article_id":   strVal(someArticleID),
				"diskspace_gb": unknownNum(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			ctx := context.Background()

			req := resource.ValidateConfigRequest{Config: configFor(t, tt.config)}
			resp := &resource.ValidateConfigResponse{}

			projectPlacementValidator{}.ValidateResource(ctx, req, resp)

			g.Expect(resp.Diagnostics.HasError()).To(Equal(tt.expectErr))
		})
	}
}

func TestDiskspaceValidator(t *testing.T) {
	tests := []struct {
		name      string
		value     types.Int64
		expectErr bool
	}{
		{name: "minimum", value: types.Int64Value(20)},
		{name: "multiple of the increment", value: types.Int64Value(80)},
		{name: "below the minimum", value: types.Int64Value(10), expectErr: true},
		{name: "zero", value: types.Int64Value(0), expectErr: true},
		{name: "negative", value: types.Int64Value(-20), expectErr: true},
		{name: "not a multiple of the increment", value: types.Int64Value(30), expectErr: true},

		// Null means unset, and an unknown value can only be checked once it is
		// resolved; neither is an error.
		{name: "null", value: types.Int64Null()},
		{name: "unknown", value: types.Int64Unknown()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			req := validator.Int64Request{
				Path:        path.Root("diskspace_gb"),
				ConfigValue: tt.value,
			}
			resp := &validator.Int64Response{}

			diskspaceValidator{}.ValidateInt64(context.Background(), req, resp)

			g.Expect(resp.Diagnostics.HasError()).To(Equal(tt.expectErr))
		})
	}
}

// configFor builds a resource config from the resource's real schema, with every
// attribute not named in values set to null.
func configFor(t *testing.T, values map[string]tftypes.Value) tfsdk.Config {
	t.Helper()

	ctx := context.Background()

	schemaResp := &resource.SchemaResponse{}
	(&Resource{}).Schema(ctx, resource.SchemaRequest{}, schemaResp)

	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("could not build the resource schema: %v", schemaResp.Diagnostics)
	}

	objType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatalf("expected the schema to be an object type, but got %T", schemaResp.Schema.Type().TerraformType(ctx))
	}

	attrs := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name, attrType := range objType.AttributeTypes {
		attrs[name] = tftypes.NewValue(attrType, nil)
	}

	for name, value := range values {
		if _, ok := attrs[name]; !ok {
			t.Fatalf("the schema does not have an attribute named %q", name)
		}
		attrs[name] = value
	}

	return tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(objType, attrs),
	}
}

func strVal(s string) tftypes.Value { return tftypes.NewValue(tftypes.String, s) }
func numVal(n int64) tftypes.Value  { return tftypes.NewValue(tftypes.Number, n) }
func boolVal(b bool) tftypes.Value  { return tftypes.NewValue(tftypes.Bool, b) }
func unknownStr() tftypes.Value     { return tftypes.NewValue(tftypes.String, tftypes.UnknownValue) }
func unknownNum() tftypes.Value     { return tftypes.NewValue(tftypes.Number, tftypes.UnknownValue) }
