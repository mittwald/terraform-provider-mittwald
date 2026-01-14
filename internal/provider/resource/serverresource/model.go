package serverresource

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ResourceModel struct {
	ID           types.String      `tfsdk:"id"`
	ShortID      types.String      `tfsdk:"short_id"`
	CustomerID   types.String      `tfsdk:"customer_id"`
	Description  types.String      `tfsdk:"description"`
	MachineType  *MachineTypeModel `tfsdk:"machine_type"`
	VolumeSize   types.Int64       `tfsdk:"volume_size"`
	UseFreeTrial types.Bool        `tfsdk:"use_free_trial"`
}

type MachineTypeModel struct {
	Name types.String  `tfsdk:"name"`
	CPU  types.Float64 `tfsdk:"cpu"`
	RAM  types.Float64 `tfsdk:"ram"`
}

func (m *ResourceModel) Validate() (d diag.Diagnostics) {
	if m.CustomerID.IsNull() {
		d.AddError("Missing value", "customer_id is a required field")
	}

	return
}
