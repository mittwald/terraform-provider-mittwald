package valueutil

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func StringOrNull[T string](s T) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(string(s))
}

func StringPtrOrNull[T ~string](s *T) types.String {
	if s == nil {
		return types.StringNull()
	}
	return types.StringValue(string(*s))
}

func StringerOrNull(s fmt.Stringer) types.String {
	if s == nil {
		return types.StringNull()
	}
	return types.StringValue(s.String())
}
