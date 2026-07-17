package valueutil

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestParseStorageGiB(t *testing.T) {
	tests := []struct {
		input     string
		expected  int64
		expectErr bool
	}{
		{input: "50Gi", expected: 50},
		{input: "20Gi", expected: 20},
		{input: "0Gi", expected: 0},
		{input: " 50Gi ", expected: 50},

		// Anything that is not a whole number of gibibytes must be rejected
		// rather than silently misinterpreted.
		{input: "50GiB", expectErr: true},
		{input: "512Mi", expectErr: true},
		{input: "1Ti", expectErr: true},
		{input: "50", expectErr: true},
		{input: "50.5Gi", expectErr: true},
		{input: "Gi", expectErr: true},
		{input: "", expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			g := NewWithT(t)

			result, err := ParseStorageGiB(tt.input)

			if tt.expectErr {
				g.Expect(err).To(HaveOccurred())
				return
			}

			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(result).To(Equal(tt.expected))
		})
	}
}
