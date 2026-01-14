package airesource

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ResourceModel describes the resource data model.
type ResourceModel struct {
	OrderID      types.String `tfsdk:"order_id"`
	ContractID   types.String `tfsdk:"contract_id"`
	CustomerId   types.String `tfsdk:"customer_id"`
	ArticleId    types.String `tfsdk:"article_id"`
	UseFreeTrial types.Bool   `tfsdk:"use_free_trial"`
}
