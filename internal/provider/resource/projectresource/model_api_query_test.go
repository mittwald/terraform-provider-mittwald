package projectresource

import (
	"encoding/json"
	"testing"

	. "github.com/onsi/gomega"
)

// TestArticleSpecToOrderSpec covers the two shapes the order API accepts. A
// project-hosting article describes either a named machine type, or the
// hardware it provides (as the "Webhosting" articles do).
func TestArticleSpecToOrderSpec(t *testing.T) {
	t.Run("machine type", func(t *testing.T) {
		g := NewWithT(t)

		machineType := "prospace.2cpu.4gb"
		spec := (&articleSpec{machineType: &machineType}).toOrderSpec()

		g.Expect(spec.AlternativeHardwareSpec).To(BeNil())
		g.Expect(spec.AlternativeMachineTypeSpec).NotTo(BeNil())
		g.Expect(spec.AlternativeMachineTypeSpec.MachineType).To(HaveValue(Equal(machineType)))

		// The API models the spec as a oneOf, so only the machine type may be
		// serialized.
		g.Expect(marshalJSON(t, &spec)).To(Equal(`{"machineType":"prospace.2cpu.4gb"}`))
	})

	t.Run("hardware spec", func(t *testing.T) {
		g := NewWithT(t)

		ram, vcpu := 1.0, 1.0
		spec := (&articleSpec{ram: &ram, vcpu: &vcpu}).toOrderSpec()

		g.Expect(spec.AlternativeMachineTypeSpec).To(BeNil())
		g.Expect(spec.AlternativeHardwareSpec).NotTo(BeNil())
		g.Expect(spec.AlternativeHardwareSpec.Ram).To(HaveValue(Equal(1.0)))
		g.Expect(spec.AlternativeHardwareSpec.Vcpu).To(HaveValue(Equal(1.0)))

		g.Expect(marshalJSON(t, &spec)).To(Equal(`{"ram":1,"vcpu":1}`))
	})
}

func TestArticleSpecToTariffChangeSpec(t *testing.T) {
	t.Run("machine type", func(t *testing.T) {
		g := NewWithT(t)

		machineType := "prospace.2cpu.4gb"
		spec := (&articleSpec{machineType: &machineType}).toTariffChangeSpec()

		g.Expect(spec.AlternativeHardwareSpec).To(BeNil())
		g.Expect(spec.AlternativeMachineTypeSpec).NotTo(BeNil())
		g.Expect(spec.AlternativeMachineTypeSpec.MachineType).To(HaveValue(Equal(machineType)))
	})

	t.Run("hardware spec", func(t *testing.T) {
		g := NewWithT(t)

		ram, vcpu := 4.0, 2.0
		spec := (&articleSpec{ram: &ram, vcpu: &vcpu}).toTariffChangeSpec()

		g.Expect(spec.AlternativeMachineTypeSpec).To(BeNil())
		g.Expect(spec.AlternativeHardwareSpec).NotTo(BeNil())
		g.Expect(spec.AlternativeHardwareSpec.Ram).To(HaveValue(Equal(4.0)))
		g.Expect(spec.AlternativeHardwareSpec.Vcpu).To(HaveValue(Equal(2.0)))
	})
}

// TestArticleSpecFromAttributes documents which article attributes select which
// spec shape. The "hardware spec" case mirrors the attributes of the real
// "Webhosting" (WH25-0007) article, which has no machine type at all.
func TestArticleSpecFromAttributes(t *testing.T) {
	tests := []struct {
		name        string
		attributes  map[string]string
		machineType string
		ram         float64
		vcpu        float64
		expectErr   bool
	}{
		{
			name: "webhosting article",
			attributes: map[string]string{
				"ram":                     "1",
				"vcpu":                    "1",
				"storage":                 "20",
				"description":             "Webhosting",
				"spec.hardware_spec.ram":  "1",
				"spec.hardware_spec.vcpu": "1",
				"allowed_features":        "[4]",
			},
			ram:  1,
			vcpu: 1,
		},
		{
			name:        "article with a prefixed machine type",
			attributes:  map[string]string{"spec.machine_type": "prospace.2cpu.4gb"},
			machineType: "prospace.2cpu.4gb",
		},
		{
			name:        "article with an unprefixed machine type",
			attributes:  map[string]string{"machine_type": "psplus-shared"},
			machineType: "psplus-shared",
		},
		{
			name:       "hardware spec with only a vcpu count",
			attributes: map[string]string{"spec.hardware_spec.vcpu": "2"},
			vcpu:       2,
		},
		{
			name:       "article describing no resources",
			attributes: map[string]string{"ram": "1", "vcpu": "1"},
			expectErr:  true,
		},
		{
			name:       "unparseable hardware spec",
			attributes: map[string]string{"spec.hardware_spec.ram": "lots"},
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			spec, err := articleSpecFromAttributes("WH25-0007", tt.attributes)

			if tt.expectErr {
				g.Expect(err).To(HaveOccurred())
				return
			}

			g.Expect(err).NotTo(HaveOccurred())

			if tt.machineType != "" {
				g.Expect(spec.machineType).To(HaveValue(Equal(tt.machineType)))
				return
			}

			g.Expect(spec.machineType).To(BeNil())

			if tt.ram == 0 {
				g.Expect(spec.ram).To(BeNil())
			} else {
				g.Expect(spec.ram).To(HaveValue(Equal(tt.ram)))
			}

			if tt.vcpu == 0 {
				g.Expect(spec.vcpu).To(BeNil())
			} else {
				g.Expect(spec.vcpu).To(HaveValue(Equal(tt.vcpu)))
			}
		})
	}
}

// TestArticleSpecFromAttributesErrorListsAttributes checks that an article that
// does not describe any resources produces an error naming what the article
// does have, since that is the only way to tell what went wrong.
func TestArticleSpecFromAttributesErrorListsAttributes(t *testing.T) {
	g := NewWithT(t)

	_, err := articleSpecFromAttributes("WH25-0007", map[string]string{"vcpu": "1", "ram": "1"})

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("WH25-0007"))
	g.Expect(err.Error()).To(ContainSubstring("ram, vcpu"))
}

func marshalJSON(t *testing.T, v any) string {
	t.Helper()

	encoded, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("could not marshal %T: %v", v, err)
	}

	return string(encoded)
}
