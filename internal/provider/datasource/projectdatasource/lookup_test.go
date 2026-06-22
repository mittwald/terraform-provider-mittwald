package projectdatasource

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	. "github.com/onsi/gomega"
)

func TestProjectLookupID(t *testing.T) {
	tests := []struct {
		name      string
		id        types.String
		shortID   types.String
		expect    string
		expectErr bool
	}{
		{
			name:    "id only",
			id:      types.StringValue("p-full-id"),
			shortID: types.StringNull(),
			expect:  "p-full-id",
		},
		{
			name:    "short id only",
			id:      types.StringNull(),
			shortID: types.StringValue("p-abcdef"),
			expect:  "p-abcdef",
		},
		{
			name:      "both set is an error",
			id:        types.StringValue("p-full-id"),
			shortID:   types.StringValue("p-abcdef"),
			expectErr: true,
		},
		{
			name:      "neither set is an error",
			id:        types.StringNull(),
			shortID:   types.StringNull(),
			expectErr: true,
		},
		{
			name:    "unknown id is treated as set",
			id:      types.StringUnknown(),
			shortID: types.StringNull(),
			expect:  "",
		},
		{
			name:    "unknown short id is treated as set",
			id:      types.StringNull(),
			shortID: types.StringUnknown(),
			expect:  "",
		},
		{
			name:      "both unknown is an error",
			id:        types.StringUnknown(),
			shortID:   types.StringUnknown(),
			expectErr: true,
		},
		{
			name:      "unknown id and known short id is an error",
			id:        types.StringUnknown(),
			shortID:   types.StringValue("p-abcdef"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			result, err := projectLookupID(tt.id, tt.shortID)

			if tt.expectErr {
				g.Expect(err).To(HaveOccurred())
				g.Expect(result).To(BeEmpty())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(result).To(Equal(tt.expect))
			}
		})
	}
}
