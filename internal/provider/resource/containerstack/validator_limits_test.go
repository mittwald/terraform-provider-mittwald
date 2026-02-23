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
			name:        "valid memory with mb suffix",
			value:       types.StringValue("512mb"),
			expectError: false,
		},
		{
			name:        "valid memory with gb suffix",
			value:       types.StringValue("1gb"),
			expectError: false,
		},
		{
			name:        "valid memory with kb suffix",
			value:       types.StringValue("1024kb"),
			expectError: false,
		},
		{
			name:        "valid memory with m suffix",
			value:       types.StringValue("512m"),
			expectError: false,
		},
		{
			name:        "valid memory with g suffix",
			value:       types.StringValue("1g"),
			expectError: false,
		},
		{
			name:        "valid memory with k suffix",
			value:       types.StringValue("1024k"),
			expectError: false,
		},
		{
			name:        "valid memory with b suffix",
			value:       types.StringValue("1048576b"),
			expectError: false,
		},
		{
			name:        "invalid memory without suffix (bytes)",
			value:       types.StringValue("1048576"),
			expectError: true,
		},
		{
			name:        "valid memory starting with 0",
			value:       types.StringValue("0m"),
			expectError: false,
		},
		{
			name:        "invalid memory with uppercase M suffix",
			value:       types.StringValue("512M"),
			expectError: true,
		},
		{
			name:        "invalid memory with uppercase G suffix",
			value:       types.StringValue("1G"),
			expectError: true,
		},
		{
			name:        "invalid memory with uppercase K suffix",
			value:       types.StringValue("1024K"),
			expectError: true,
		},
		{
			name:        "invalid memory with uppercase T suffix",
			value:       types.StringValue("2T"),
			expectError: true,
		},
		{
			name:        "invalid memory with lowercase t suffix",
			value:       types.StringValue("2t"),
			expectError: true,
		},
		{
			name:        "invalid memory with invalid suffix",
			value:       types.StringValue("512X"),
			expectError: true,
		},
		{
			name:        "invalid memory with decimal",
			value:       types.StringValue("1.5g"),
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
