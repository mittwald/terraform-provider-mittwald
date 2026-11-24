package aiapikeyresource

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/aihostingv2"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiext"
)

// resolvePlan determines the AI hosting plan (and its associated contract)
// that a key should be scoped to. If configuredContractID is set, the plan
// whose contract matches it is used. Otherwise, the customer's only active AI
// hosting plan is used.
func (r *Resource) resolvePlan(ctx context.Context, customerID, configuredContractID string) (planID, contractID string, diags diag.Diagnostics) {
	client := apiext.NewAIHostingClient(r.client)

	plans, err := client.ListAllCustomerPlans(ctx, customerID)
	if err != nil {
		diags.AddError("Error listing AI hosting plans", err.Error())
		return
	}

	plans = activePlans(plans)

	if configuredContractID != "" {
		planID, diags = findPlanByContractID(ctx, client, customerID, configuredContractID, plans)
		return planID, configuredContractID, diags
	}

	planID, diags = selectOnlyPlan(customerID, plans)
	if diags.HasError() {
		return
	}

	contractID, err = client.GetPlanContractID(ctx, customerID, planID)
	if err != nil {
		diags.AddError("Error reading AI hosting contract", err.Error())
	}

	return
}

// activePlans filters out AI hosting plans that have already been deleted.
// New keys can't be scoped to a deleted plan, so these must neither count
// towards an ambiguous automatic plan selection nor be matched against a
// configured contract ID.
func activePlans(plans []aihostingv2.CustomerPlan) []aihostingv2.CustomerPlan {
	active := make([]aihostingv2.CustomerPlan, 0, len(plans))
	for _, plan := range plans {
		if plan.DeletedAt == nil {
			active = append(active, plan)
		}
	}

	return active
}

// selectOnlyPlan returns the ID of the only plan in plans, or an error if
// there is not exactly one plan to choose from.
func selectOnlyPlan(customerID string, plans []aihostingv2.CustomerPlan) (string, diag.Diagnostics) {
	var diags diag.Diagnostics

	switch len(plans) {
	case 1:
		return plans[0].PlanId, diags
	case 0:
		diags.AddAttributeError(
			path.Root("contract_id"),
			"No AI hosting plan found",
			fmt.Sprintf("Customer %s has no AI hosting plan. Create a mittwald_ai resource for this customer first, or set contract_id explicitly.", customerID),
		)
	default:
		diags.AddAttributeError(
			path.Root("contract_id"),
			"Ambiguous AI hosting plan",
			fmt.Sprintf("Customer %s has %d AI hosting plans. Set contract_id to the contract_id of the mittwald_ai resource this key should be scoped to.", customerID, len(plans)),
		)
	}

	return "", diags
}

// findPlanByContractID returns the ID of the plan (out of plans) that belongs
// to the given contract, or an error if there is none.
func findPlanByContractID(ctx context.Context, client apiext.AIHostingClient, customerID, contractID string, plans []aihostingv2.CustomerPlan) (string, diag.Diagnostics) {
	var diags diag.Diagnostics

	planID, err := client.FindPlanIDByContractID(ctx, customerID, contractID, plans)
	if err != nil {
		diags.AddError("Error reading AI hosting contract", err.Error())
		return "", diags
	}

	if planID == "" {
		diags.AddAttributeError(
			path.Root("contract_id"),
			"AI hosting plan not found",
			fmt.Sprintf("No AI hosting plan found for contract %s on customer %s.", contractID, customerID),
		)
	}

	return planID, diags
}
