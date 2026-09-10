package valueutil

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func TestTimePtrToRFC3339OrNull(t *testing.T) {
	ref := time.Date(2026, 9, 10, 12, 30, 0, 0, time.UTC)

	tests := []struct {
		name     string
		input    *time.Time
		expected string
		isNull   bool
	}{
		{name: "nil pointer", input: nil, isNull: true},
		{name: "valid time", input: &ref, expected: "2026-09-10T12:30:00Z"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			result := TimePtrToRFC3339OrNull(tt.input)

			if tt.isNull {
				g.Expect(result.IsNull()).To(BeTrue())
			} else {
				g.Expect(result.IsNull()).To(BeFalse())
				g.Expect(result.ValueString()).To(Equal(tt.expected))
			}
		})
	}
}
