package valueutil

import (
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TimePtrToRFC3339OrNull converts a *time.Time into an RFC 3339-formatted
// types.String, or types.StringNull() if the pointer is nil.
func TimePtrToRFC3339OrNull(t *time.Time) types.String {
	if t == nil {
		return types.StringNull()
	}
	return types.StringValue(t.Format(time.RFC3339))
}
