package airesource

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/contractv2"
)

func (r *ResourceModel) FromAPIModel(ctx context.Context, apiModel *contractv2.Contract) diag.Diagnostics {
	if apiModel == nil {
		r.ContractID = types.StringNull()
		r.ArticleID = types.StringNull()
		r.UseFreeTrial = types.BoolUnknown()
		return nil
	}

	r.ContractID = types.StringValue(apiModel.ContractId)

	if len(apiModel.BaseItem.Articles) > 0 {
		r.ArticleID = types.StringValue(apiModel.BaseItem.Articles[0].Id)
	} else {
		r.ArticleID = types.StringNull()
	}

	return nil
}
