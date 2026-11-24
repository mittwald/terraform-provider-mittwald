package aiapikeyresource

import (
	"testing"
	"time"

	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/aihostingv2"
	. "github.com/onsi/gomega"
)

func plan(id string, deleted bool) aihostingv2.CustomerPlan {
	p := aihostingv2.CustomerPlan{PlanId: id}
	if deleted {
		deletedAt := time.Now()
		p.DeletedAt = &deletedAt
	}
	return p
}

func TestActivePlans(t *testing.T) {
	g := NewWithT(t)

	active := activePlans([]aihostingv2.CustomerPlan{
		plan("plan-1", false),
		plan("plan-2", true),
		plan("plan-3", false),
	})

	g.Expect(active).To(HaveLen(2))
	g.Expect(active[0].PlanId).To(Equal("plan-1"))
	g.Expect(active[1].PlanId).To(Equal("plan-3"))
}

func TestSelectOnlyPlan(t *testing.T) {
	tests := []struct {
		name          string
		plans         []aihostingv2.CustomerPlan
		expectedID    string
		expectedError string
	}{
		{
			name:          "no plans",
			plans:         nil,
			expectedError: "No AI hosting plan found",
		},
		{
			name:       "exactly one plan",
			plans:      []aihostingv2.CustomerPlan{plan("plan-1", false)},
			expectedID: "plan-1",
		},
		{
			name:          "multiple plans",
			plans:         []aihostingv2.CustomerPlan{plan("plan-1", false), plan("plan-2", false)},
			expectedError: "Ambiguous AI hosting plan",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			planID, diags := selectOnlyPlan("customer-1", tt.plans)

			g.Expect(planID).To(Equal(tt.expectedID))
			if tt.expectedError == "" {
				g.Expect(diags.HasError()).To(BeFalse())
			} else {
				g.Expect(diags.HasError()).To(BeTrue())
				g.Expect(diags.Errors()[0].Summary()).To(Equal(tt.expectedError))
			}
		})
	}
}
