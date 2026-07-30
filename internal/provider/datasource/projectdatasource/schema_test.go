package projectdatasource

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	. "github.com/onsi/gomega"
)

// TestDataSourceModelMatchesSchema asserts that the data source model (which
// embeds the shared project model) can actually be filled from a config object
// built from the data source schema.
func TestDataSourceModelMatchesSchema(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	ds, ok := New().(*DataSource)
	g.Expect(ok).To(BeTrue())

	resp := datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, &resp)

	g.Expect(resp.Diagnostics.HasError()).To(BeFalse())

	objectType, ok := resp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	g.Expect(ok).To(BeTrue())
	attributes := make(map[string]tftypes.Value, len(objectType.AttributeTypes))

	for name, attributeType := range objectType.AttributeTypes {
		attributes[name] = tftypes.NewValue(attributeType, nil)
	}

	config := tfsdk.Config{
		Schema: resp.Schema,
		Raw:    tftypes.NewValue(objectType, attributes),
	}

	var data dataSourceModel

	g.Expect(config.Get(ctx, &data)).To(BeEmpty())
	g.Expect(data.ID.IsNull()).To(BeTrue())
	g.Expect(data.Timeouts.IsNull()).To(BeTrue())
}
