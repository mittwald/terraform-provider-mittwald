package aiapikeyresource

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	. "github.com/onsi/gomega"
)

func TestKeepInferredContractID(t *testing.T) {
	state := ResourceModel{
		CustomerID: types.StringValue("customer-1"),
		ContractID: types.StringValue("contract-1"),
	}

	tests := []struct {
		name             string
		plan             ResourceModel
		config           ResourceModel
		expectedContract types.String
	}{
		{
			name: "keeps inferred contract when customer is unchanged",
			plan: ResourceModel{
				CustomerID: types.StringValue("customer-1"),
				ContractID: types.StringUnknown(),
			},
			config:           ResourceModel{ContractID: types.StringNull()},
			expectedContract: types.StringValue("contract-1"),
		},
		{
			name: "leaves contract unknown when customer changes",
			plan: ResourceModel{
				CustomerID: types.StringValue("customer-2"),
				ContractID: types.StringUnknown(),
			},
			config:           ResourceModel{ContractID: types.StringNull()},
			expectedContract: types.StringUnknown(),
		},
		{
			name: "leaves contract unknown when customer is not yet known",
			plan: ResourceModel{
				CustomerID: types.StringUnknown(),
				ContractID: types.StringUnknown(),
			},
			config:           ResourceModel{ContractID: types.StringNull()},
			expectedContract: types.StringUnknown(),
		},
		{
			name: "does not override a configured but not yet known contract",
			plan: ResourceModel{
				CustomerID: types.StringValue("customer-1"),
				ContractID: types.StringUnknown(),
			},
			config:           ResourceModel{ContractID: types.StringUnknown()},
			expectedContract: types.StringUnknown(),
		},
		{
			name: "does not override a known planned contract",
			plan: ResourceModel{
				CustomerID: types.StringValue("customer-1"),
				ContractID: types.StringValue("contract-2"),
			},
			config:           ResourceModel{ContractID: types.StringValue("contract-2")},
			expectedContract: types.StringValue("contract-2"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			plan := tt.plan
			keepInferredContractID(&plan, &tt.config, &state)

			g.Expect(plan.ContractID).To(Equal(tt.expectedContract))
		})
	}
}
