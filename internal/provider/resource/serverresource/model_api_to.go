package serverresource

import (
	"context"

	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/contractclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/orderv2"
)

func (r *ResourceModel) ToAPICreateOrderRequest(ctx context.Context, client mittwaldv2.Client) (*contractclientv2.CreateOrderRequest, error) {
	machineType, err := r.QueryArticleMachineType(ctx, client)
	if err != nil {
		return nil, err
	}

	orderType := contractclientv2.CreateOrderRequestBodyOrderTypeServer
	orderRequest := contractclientv2.CreateOrderRequest{
		Body: contractclientv2.CreateOrderRequestBody{
			OrderType: &orderType,
			OrderData: &contractclientv2.CreateOrderRequestBodyOrderData{
				AlternativeServerOrder: &orderv2.ServerOrder{
					CustomerId:     r.CustomerID.ValueString(),
					Description:    r.Description.ValueString(),
					DiskspaceInGiB: float64(r.DiskspaceGB.ValueInt64()),
					MachineType:    machineType,
					UseFreeTrial:   r.UseFreeTrial.ValueBoolPointer(),
				},
			},
		},
	}

	return &orderRequest, nil
}

func (r *ResourceModel) ToAPIChangePlanRequest(ctx context.Context, client mittwaldv2.Client) (*contractclientv2.CreateTariffChangeRequest, error) {
	machineType, err := r.QueryArticleMachineType(ctx, client)
	if err != nil {
		return nil, err
	}

	changeType := contractclientv2.CreateTariffChangeRequestBodyTariffChangeTypeServer
	changeReq := contractclientv2.CreateTariffChangeRequest{
		Body: contractclientv2.CreateTariffChangeRequestBody{
			TariffChangeType: &changeType,
			TariffChangeData: &contractclientv2.CreateTariffChangeRequestBodyTariffChangeData{
				AlternativeServerTariffChange: &orderv2.ServerTariffChange{
					ContractId:     r.ContractID.ValueString(),
					DiskspaceInGiB: float64(r.DiskspaceGB.ValueInt64()),
					MachineType:    machineType,
				},
			},
		},
	}

	return &changeReq, nil
}
