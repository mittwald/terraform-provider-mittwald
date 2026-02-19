package common_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/resource/common"
	. "github.com/onsi/gomega"
)

func TestUUIDValidator(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	v := &common.UUIDValidator{}

	tests := []struct {
		name         string
		value        types.String
		expectError  bool
		errorSummary string
	}{
		{
			name:        "valid UUID",
			value:       types.StringValue("10184af5-6716-4e82-81d7-4b1cd317d147"),
			expectError: false,
		},
		{
			name:        "valid UUID all lowercase",
			value:       types.StringValue("a1b2c3d4-e5f6-1234-5678-9abcdef01234"),
			expectError: false,
		},
		{
			name:         "server short ID",
			value:        types.StringValue("s-abc123"),
			expectError:  true,
			errorSummary: "Short ID Not Supported",
		},
		{
			name:         "project short ID",
			value:        types.StringValue("p-xyz789"),
			expectError:  true,
			errorSummary: "Short ID Not Supported",
		},
		{
			name:         "app short ID",
			value:        types.StringValue("a-def456"),
			expectError:  true,
			errorSummary: "Short ID Not Supported",
		},
		{
			name:         "container short ID",
			value:        types.StringValue("c-ghi012"),
			expectError:  true,
			errorSummary: "Short ID Not Supported",
		},
		{
			name:         "invalid UUID - missing dashes",
			value:        types.StringValue("10184af567164e8281d74b1cd317d147"),
			expectError:  true,
			errorSummary: "Invalid UUID Format",
		},
		{
			name:         "invalid UUID - wrong format",
			value:        types.StringValue("not-a-uuid"),
			expectError:  true,
			errorSummary: "Invalid UUID Format",
		},
		{
			name:         "invalid UUID - uppercase letters",
			value:        types.StringValue("10184AF5-6716-4E82-81D7-4B1CD317D147"),
			expectError:  true,
			errorSummary: "Invalid UUID Format",
		},
		{
			name:         "empty string",
			value:        types.StringValue(""),
			expectError:  true,
			errorSummary: "Invalid UUID Format",
		},
		{
			name:        "null value should not error",
			value:       types.StringNull(),
			expectError: false,
		},
		{
			name:        "unknown value should not error",
			value:       types.StringUnknown(),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("test_id"),
				ConfigValue: tt.value,
			}
			resp := &validator.StringResponse{}

			v.ValidateString(ctx, req, resp)

			if tt.expectError {
				g.Expect(resp.Diagnostics.HasError()).To(BeTrue())
				if tt.errorSummary != "" {
					g.Expect(resp.Diagnostics.Errors()[0].Summary()).To(Equal(tt.errorSummary))
				}
			} else {
				g.Expect(resp.Diagnostics.HasError()).To(BeFalse())
			}
		})
	}
}
