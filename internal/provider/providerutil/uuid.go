package providerutil

import (
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func ParseUUID(in string, d *diag.Diagnostics) uuid.UUID {
	return Try[uuid.UUID](d, "Invalid app ID").
		DoVal(uuid.Parse(in))
}
