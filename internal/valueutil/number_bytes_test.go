package valueutil

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
)

func TestParseByteSize(t *testing.T) {
	RegisterTestingT(t)

	tests := []struct {
		input    string
		expected int64
		hasError bool
	}{
		// Valid cases
		{"1024B", 1024, false},
		{"1KiB", 1024, false},
		{"1MiB", 1024 * 1024, false},
		{"1GiB", 1024 * 1024 * 1024, false},
		{"10Ki", 10 * 1024, false},
		{"5Mi", 5 * 1024 * 1024, false},
		{"2Gi", 2 * 1024 * 1024 * 1024, false},
		{"0B", 0, false},
		{"100", 100, false}, // Should default to bytes

		// Invalid cases
		{"1.5GiB", 0, true}, // Decimal numbers not supported
		{"GiB", 0, true},    // No numeric value
		{"1KB", 0, true},    // Unrecognized unit
		{"-1KiB", 0, true},  // Negative values not supported
		{"", 0, true},       // Empty string
		{"abc", 0, true},    // Completely invalid input
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseByteSize(tt.input)

			if tt.hasError {
				Expect(err).To(HaveOccurred(), fmt.Sprintf("Expected error for input %q, but got none", tt.input))
			} else {
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Did not expect error for input %q, but got %v", tt.input, err))
				Expect(result).To(Equal(tt.expected), fmt.Sprintf("Expected %d for input %q, but got %d", tt.expected, tt.input, result))
			}
		})
	}
}
