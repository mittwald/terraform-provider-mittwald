package containerstackresource_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	containerstackresource "github.com/mittwald/terraform-provider-mittwald/internal/provider/resource/containerstack"
	. "github.com/onsi/gomega"
)

func TestCpusValidator(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	v := &containerstackresource.CpusValidator{}

	tests := []struct {
		name        string
		value       types.Float64
		expectError bool
	}{
		{
			name:        "valid positive value",
			value:       types.Float64Value(0.5),
			expectError: false,
		},
		{
			name:        "valid integer value",
			value:       types.Float64Value(2),
			expectError: false,
		},
		{
			name:        "zero value should error",
			value:       types.Float64Value(0),
			expectError: true,
		},
		{
			name:        "negative value should error",
			value:       types.Float64Value(-1),
			expectError: true,
		},
		{
			name:        "null value should not error",
			value:       types.Float64Null(),
			expectError: false,
		},
		{
			name:        "unknown value should not error",
			value:       types.Float64Unknown(),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.Float64Request{
				Path:        path.Root("cpus"),
				ConfigValue: tt.value,
			}
			resp := &validator.Float64Response{}

			v.ValidateFloat64(ctx, req, resp)

			if tt.expectError {
				g.Expect(resp.Diagnostics.HasError()).To(BeTrue())
			} else {
				g.Expect(resp.Diagnostics.HasError()).To(BeFalse())
			}
		})
	}
}

func TestMemoryValidator(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	v := &containerstackresource.MemoryValidator{}

	tests := []struct {
		name        string
		value       types.String
		expectError bool
	}{
		{
			name:        "valid memory with M suffix",
			value:       types.StringValue("512M"),
			expectError: false,
		},
		{
			name:        "valid memory with G suffix",
			value:       types.StringValue("1G"),
			expectError: false,
		},
		{
			name:        "valid memory with K suffix",
			value:       types.StringValue("1024K"),
			expectError: false,
		},
		{
			name:        "valid memory with T suffix",
			value:       types.StringValue("2T"),
			expectError: false,
		},
		{
			name:        "valid memory without suffix",
			value:       types.StringValue("1048576"),
			expectError: false,
		},
		{
			name:        "invalid memory starting with 0",
			value:       types.StringValue("0M"),
			expectError: true,
		},
		{
			name:        "valid memory with lowercase m suffix",
			value:       types.StringValue("512m"),
			expectError: false,
		},
		{
			name:        "valid memory with lowercase g suffix",
			value:       types.StringValue("1g"),
			expectError: false,
		},
		{
			name:        "valid memory with lowercase k suffix",
			value:       types.StringValue("1024k"),
			expectError: false,
		},
		{
			name:        "valid memory with lowercase t suffix",
			value:       types.StringValue("2t"),
			expectError: false,
		},
		{
			name:        "invalid memory with invalid suffix",
			value:       types.StringValue("512X"),
			expectError: true,
		},
		{
			name:        "invalid memory with decimal",
			value:       types.StringValue("1.5G"),
			expectError: true,
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
				Path:        path.Root("memory"),
				ConfigValue: tt.value,
			}
			resp := &validator.StringResponse{}

			v.ValidateString(ctx, req, resp)

			if tt.expectError {
				g.Expect(resp.Diagnostics.HasError()).To(BeTrue())
			} else {
				g.Expect(resp.Diagnostics.HasError()).To(BeFalse())
			}
		})
	}
}
