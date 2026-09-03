package projectresource_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	. "github.com/onsi/gomega"

	"github.com/mittwald/terraform-provider-mittwald/internal/provider/resource/projectresource"
)

// TestResourceModelMatchesSchema asserts that the resource model (including the
// embedded shared model and the timeouts block) can actually be filled from a
// state object built from the resource schema.
func TestResourceModelMatchesSchema(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	res := projectresource.New()

	resp := resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, &resp)

	g.Expect(resp.Diagnostics.HasError()).To(BeFalse())

	objectType, ok := resp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	g.Expect(ok).To(BeTrue())
	attributes := make(map[string]tftypes.Value, len(objectType.AttributeTypes))

	for name, attributeType := range objectType.AttributeTypes {
		attributes[name] = tftypes.NewValue(attributeType, nil)
	}

	state := tfsdk.State{
		Schema: resp.Schema,
		Raw:    tftypes.NewValue(objectType, attributes),
	}

	var data projectresource.ResourceModelWithTimeouts

	g.Expect(state.Get(ctx, &data)).To(BeEmpty())
	g.Expect(data.ID.IsNull()).To(BeTrue())
	g.Expect(data.Timeouts.IsNull()).To(BeTrue())
}
