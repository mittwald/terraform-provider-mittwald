package valueutil

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func BoolPtrOrNull[T ~bool](s *T) types.Bool {
	if s == nil {
		return types.BoolNull()
	}
	return types.BoolValue(bool(*s))
}
