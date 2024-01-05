package valueutil

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"reflect"
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
	if isReallyNil(s) {
		return types.StringNull()
	}
	return types.StringValue(s.String())
}

func isReallyNil(v any) bool {
	if v == nil {
		return true
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return rv.IsNil()
	case reflect.Array:
		return rv.Len() == 0
	default:
		return false
	}
}
