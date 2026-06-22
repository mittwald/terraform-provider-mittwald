package projectdatasource

import (
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// projectLookupID returns the identifier to use when looking up a project,
// validating that exactly one of id/short_id is set. The mittwald API resolves
// both full and short IDs through the same GetProject endpoint, so either value
// can be passed straight through.
//
// Presence is determined by null-ness only: an unknown value (e.g. a selector
// that references a not-yet-known value at plan time) is treated as set, never
// as unset.
func projectLookupID(id, shortID types.String) (string, error) {
	hasID := !id.IsNull()
	hasShortID := !shortID.IsNull()

	switch {
	case hasID && hasShortID:
		return "", errors.New("exactly one of `id` or `short_id` must be set, but both were provided")
	case !hasID && !hasShortID:
		return "", errors.New("exactly one of `id` or `short_id` must be set, but neither was provided")
	case hasID:
		return id.ValueString(), nil
	default:
		return shortID.ValueString(), nil
	}
}
