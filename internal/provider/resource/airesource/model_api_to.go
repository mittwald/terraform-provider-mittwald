package airesource

import (
	"context"

	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/contractclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/orderv2"
)

func (r *ResourceModel) ToAPICreateOrderRequest(ctx context.Context, client mittwaldv2.Client) (*contractclientv2.CreateOrderRequest, error) {
	monthlyTokens, requestsPerMinute, err := r.QueryArticleFeatures(ctx, client)
	if err != nil {
		return nil, err
	}

	orderType := contractclientv2.CreateOrderRequestBodyOrderTypeAIHosting
	orderRequest := contractclientv2.CreateOrderRequest{
		Body: contractclientv2.CreateOrderRequestBody{
			OrderType: &orderType,
			OrderData: &contractclientv2.CreateOrderRequestBodyOrderData{
				AlternativeAIHostingOrder: &orderv2.AIHostingOrder{
					CustomerId:        r.CustomerID.ValueString(),
					UseFreeTrial:      r.UseFreeTrial.ValueBoolPointer(),
					MonthlyTokens:     monthlyTokens,
					RequestsPerMinute: requestsPerMinute,
				},
			},
		},
	}

	return &orderRequest, nil
}

func (r *ResourceModel) ToAPIChangePlanRequest(ctx context.Context, client mittwaldv2.Client) (*contractclientv2.CreateTariffChangeRequest, error) {
	monthlyTokens, requestsPerMinute, err := r.QueryArticleFeatures(ctx, client)
	if err != nil {
		return nil, err
	}

	changeType := contractclientv2.CreateTariffChangeRequestBodyTariffChangeTypeAIHosting
	changeReq := contractclientv2.CreateTariffChangeRequest{
		Body: contractclientv2.CreateTariffChangeRequestBody{
			TariffChangeType: &changeType,
			TariffChangeData: &contractclientv2.CreateTariffChangeRequestBodyTariffChangeData{
				AlternativeAIHostingTariffChange: &orderv2.AIHostingTariffChange{
					ContractId:        r.ContractID.ValueString(),
					MonthlyTokens:     monthlyTokens,
					RequestsPerMinute: requestsPerMinute,
				},
			},
		},
	}

	return &changeReq, nil
}
