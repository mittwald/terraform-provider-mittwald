package serverresource

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/contractclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/contractv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/orderv2"
)

func (m *ResourceModel) Reset() {
	m.ID = types.StringNull()
	m.ShortID = types.StringNull()
	m.CustomerID = types.StringNull()
	m.Description = types.StringNull()
	m.MachineType = nil
	m.VolumeSize = types.Int64Null()
	m.UseFreeTrial = types.BoolNull()
}

func (m *ResourceModel) FromAPIModel(ctx context.Context, contract *contractv2.Contract) (res diag.Diagnostics) {
	if contract == nil {
		m.Reset()
		return
	}

	baseItem := contract.BaseItem

	if baseItem.AggregateReference == nil {
		res.AddError("Invalid server contract", "Contract does not have an aggregate reference")
		return
	}

	m.ID = types.StringValue(baseItem.AggregateReference.Id)
	m.CustomerID = types.StringValue(contract.CustomerId)
	m.Description = types.StringValue(baseItem.Description)

	machineTypeArticle := findMachineTypeArticle(baseItem.Articles)
	if machineTypeArticle != nil {
		m.MachineType = &MachineTypeModel{
			Name: types.StringValue(machineTypeArticle.Name),
			CPU:  types.Float64Unknown(),
			RAM:  types.Float64Unknown(),
		}
	}

	diskspaceArticle := findDiskspaceArticle(baseItem.Articles)
	if diskspaceArticle != nil {
		m.VolumeSize = types.Int64Value(diskspaceArticle.Amount)
	}

	return
}

func (m *ResourceModel) ToCreateOrderRequest() contractclientv2.CreateOrderRequest {
	useFreeTrial := m.UseFreeTrial.ValueBoolPointer()
	orderType := contractclientv2.CreateOrderRequestBodyOrderTypeServer

	return contractclientv2.CreateOrderRequest{
		Body: contractclientv2.CreateOrderRequestBody{
			OrderType: &orderType,
			OrderData: &contractclientv2.CreateOrderRequestBodyOrderData{
				AlternativeServerOrder: &orderv2.ServerOrder{
					CustomerId:     m.CustomerID.ValueString(),
					Description:    m.Description.ValueString(),
					DiskspaceInGiB: float64(m.VolumeSize.ValueInt64()),
					MachineType:    m.MachineType.Name.ValueString(),
					UseFreeTrial:   useFreeTrial,
				},
			},
		},
	}
}

func findMachineTypeArticle(articles []contractv2.Article) *contractv2.Article {
	machineTypeTemplates := []string{"vServer", "dediServer"}

	for i, article := range articles {
		for _, template := range machineTypeTemplates {
			if article.ArticleTemplateId == template {
				return &articles[i]
			}
		}
	}

	return nil
}

func findDiskspaceArticle(articles []contractv2.Article) *contractv2.Article {
	diskspaceTemplate := "diskspace"

	for i, article := range articles {
		if article.ArticleTemplateId == diskspaceTemplate {
			return &articles[i]
		}
	}

	return nil
}

