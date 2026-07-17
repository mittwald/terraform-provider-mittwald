package projectresource

import (
	"context"

	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/contractclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/projectclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/orderv2"
)

// ToCreateRequest builds the request for creating a project on an existing
// server.
func (m *ResourceModel) ToCreateRequest() projectclientv2.CreateProjectRequest {
	return projectclientv2.CreateProjectRequest{
		ServerID: m.ServerID.ValueString(),
		Body: projectclientv2.CreateProjectRequestBody{
			Description: m.Description.ValueString(),
		},
	}
}

// ToAPICreateOrderRequest builds the order for a stand-alone project.
func (m *ResourceModel) ToAPICreateOrderRequest(ctx context.Context, client mittwaldv2.Client) (*contractclientv2.CreateOrderRequest, error) {
	spec, err := m.QueryArticleSpec(ctx, client)
	if err != nil {
		return nil, err
	}

	orderType := contractclientv2.CreateOrderRequestBodyOrderTypeProjectHosting
	orderRequest := contractclientv2.CreateOrderRequest{
		Body: contractclientv2.CreateOrderRequestBody{
			OrderType: &orderType,
			OrderData: &contractclientv2.CreateOrderRequestBodyOrderData{
				AlternativeProjectHostingOrder: &orderv2.ProjectHostingOrder{
					CustomerId:     m.CustomerID.ValueString(),
					Description:    m.Description.ValueString(),
					DiskspaceInGiB: float64(m.DiskspaceGB.ValueInt64()),
					Spec:           spec.toOrderSpec(),
					UseFreeTrial:   m.UseFreeTrial.ValueBoolPointer(),
				},
			},
		},
	}

	return &orderRequest, nil
}

// ToAPIChangePlanRequest builds the tariff change for an already-ordered
// stand-alone project.
func (m *ResourceModel) ToAPIChangePlanRequest(ctx context.Context, client mittwaldv2.Client) (*contractclientv2.CreateTariffChangeRequest, error) {
	spec, err := m.QueryArticleSpec(ctx, client)
	if err != nil {
		return nil, err
	}

	changeType := contractclientv2.CreateTariffChangeRequestBodyTariffChangeTypeProjectHosting
	changeReq := contractclientv2.CreateTariffChangeRequest{
		Body: contractclientv2.CreateTariffChangeRequestBody{
			TariffChangeType: &changeType,
			TariffChangeData: &contractclientv2.CreateTariffChangeRequestBodyTariffChangeData{
				AlternativeProjectHostingTariffChange: &orderv2.ProjectHostingTariffChange{
					ContractId:     m.ContractID.ValueString(),
					DiskspaceInGiB: float64(m.DiskspaceGB.ValueInt64()),
					Spec:           spec.toTariffChangeSpec(),
				},
			},
		},
	}

	return &changeReq, nil
}
