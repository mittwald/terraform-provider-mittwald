package airesource

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/contractv2"
)

func (r *ResourceModel) FromAPIModel(ctx context.Context, apiModel *contractv2.Contract) diag.Diagnostics {
	if apiModel == nil {
		r.OrderID = types.StringNull()
		r.ArticleId = types.StringNull()
		r.UseFreeTrial = types.BoolUnknown()
		return nil
	}

	if apiModel.BaseItem.OrderId != nil {
		r.OrderID = types.StringValue(*apiModel.BaseItem.OrderId)
	} else {
		r.OrderID = types.StringNull()
	}

	r.ContractID = types.StringValue(apiModel.ContractId)

	if len(apiModel.BaseItem.Articles) > 0 {
		r.ArticleId = types.StringValue(apiModel.BaseItem.Articles[0].Id)
	} else {
		r.ArticleId = types.StringNull()
	}

	return nil
}
