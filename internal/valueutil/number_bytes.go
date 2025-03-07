package valueutil

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"strconv"
)

func ParseByteSize(input string) (int64, error) {
	// Define multipliers for binary byte units
	unitMap := map[string]int64{
		"B":  1,
		"Ki": 1024, "KiB": 1024,
		"Mi": 1024 * 1024, "MiB": 1024 * 1024,
		"Gi": 1024 * 1024 * 1024, "GiB": 1024 * 1024 * 1024,
	}

	// Find where the numeric part ends
	var numStr string
	var unitStr string
	for i, r := range input {
		if r < '0' || r > '9' {
			numStr = input[:i]
			unitStr = input[i:]
			break
		}
	}

	// If no unit found, assume bytes
	if unitStr == "" {
		numStr = input
		unitStr = "B"
	}

	// Convert the numeric part to an integer
	num, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number format: %v", err)
	}

	// Look up the unit multiplier
	multiplier, found := unitMap[unitStr]
	if !found {
		return 0, fmt.Errorf("unknown unit: %s", unitStr)
	}

	// Calculate the final byte size
	return num * multiplier, nil
}

func Int64FromByteQuantity(qty string, d *diag.Diagnostics) types.Int64 {
	val, err := ParseByteSize(qty)
	if err != nil {
		d.Append(diag.NewErrorDiagnostic("error converting byte quantity", err.Error()))
		return types.Int64Unknown()
	}

	return types.Int64Value(val)
}
