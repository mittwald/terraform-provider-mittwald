package appresource

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	. "github.com/onsi/gomega"
)

func TestInstallationPathValidator(t *testing.T) {
	ctx := context.Background()
	v := &installationPathValidator{}

	tests := []struct {
		name        string
		value       types.String
		expectError bool
	}{
		{name: "leading slash", value: types.StringValue("/developer-portal"), expectError: false},
		{name: "nested path with leading slash", value: types.StringValue("/apps/developer-portal"), expectError: false},
		{name: "missing leading slash", value: types.StringValue("developer-portal"), expectError: true},
		{name: "empty string", value: types.StringValue(""), expectError: true},
		{name: "null", value: types.StringNull(), expectError: false},
		{name: "unknown", value: types.StringUnknown(), expectError: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			req := validator.StringRequest{
				Path:        path.Root("installation_path"),
				ConfigValue: tt.value,
			}
			resp := &validator.StringResponse{}

			v.ValidateString(ctx, req, resp)

			g.Expect(resp.Diagnostics.HasError()).To(Equal(tt.expectError))
		})
	}
}
