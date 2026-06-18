package projectdatasource

import (
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// projectLookupID returns the identifier to use when looking up a project,
// validating that exactly one of id/short_id is set. The mittwald API resolves
// both full and short IDs through the same GetProject endpoint, so either value
// can be passed straight through.
func projectLookupID(id, shortID types.String) (string, error) {
	hasID := !id.IsNull() && id.ValueString() != ""
	hasShortID := !shortID.IsNull() && shortID.ValueString() != ""

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
