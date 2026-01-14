package airesource

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ResourceModel describes the resource data model.
type ResourceModel struct {
	ContractID   types.String `tfsdk:"contract_id"`
	CustomerID   types.String `tfsdk:"customer_id"`
	ArticleID    types.String `tfsdk:"article_id"`
	UseFreeTrial types.Bool   `tfsdk:"use_free_trial"`
}
