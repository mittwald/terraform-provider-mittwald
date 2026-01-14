package serverresource

import (
	"context"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/contractclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/orderv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/projectv2"
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

func (m *ResourceModel) FromAPIModel(ctx context.Context, server *projectv2.Server) (res diag.Diagnostics) {
	if server == nil {
		m.Reset()
		return
	}

	m.ID = types.StringValue(server.Id)
	m.ShortID = types.StringValue(server.ShortId)
	m.CustomerID = types.StringValue(server.CustomerId)
	m.Description = types.StringValue(server.Description)

	cpu := parseCPU(server.MachineType.Cpu)
	memory := parseMemory(server.MachineType.Memory)

	m.MachineType = &MachineTypeModel{
		Name: types.StringValue(server.MachineType.Name),
		CPU:  types.Float64Value(cpu),
		RAM:  types.Float64Value(memory),
	}

	volumeSize := parseStorage(server.Storage)
	m.VolumeSize = types.Int64Value(volumeSize)

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

func parseCPU(cpu string) float64 {
	cpu = strings.TrimSpace(cpu)
	value, err := strconv.ParseFloat(cpu, 64)
	if err != nil {
		return 0
	}
	return value
}

func parseMemory(memory string) float64 {
	memory = strings.TrimSpace(memory)
	memory = strings.TrimSuffix(memory, "GiB")
	memory = strings.TrimSuffix(memory, "GB")
	memory = strings.TrimSpace(memory)

	value, err := strconv.ParseFloat(memory, 64)
	if err != nil {
		return 0
	}
	return value
}

func parseStorage(storage string) int64 {
	storage = strings.TrimSpace(storage)
	storage = strings.TrimSuffix(storage, "GiB")
	storage = strings.TrimSuffix(storage, "GB")
	storage = strings.TrimSpace(storage)

	value, err := strconv.ParseInt(storage, 10, 64)
	if err != nil {
		return 0
	}
	return value
}
