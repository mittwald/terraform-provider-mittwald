package containerstackresource

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"strings"
)

var _ planmodifier.String = &StripLibraryPrefixFromImage{}

// StripLibraryPrefixFromImage is a planmodifier.String that strips the "library/"
// prefix from the image name. This is done because most users will probably not
// want to specify the library prefix in their configuration, but the API implicitly
// adds it. For this reason, we need to unify the image name in the plan and state.
type StripLibraryPrefixFromImage struct {
}

func (s *StripLibraryPrefixFromImage) Description(_ context.Context) string {
	return "Strips the library prefix from the image name"
}

func (s *StripLibraryPrefixFromImage) MarkdownDescription(ctx context.Context) string {
	return s.Description(ctx)
}

func (s *StripLibraryPrefixFromImage) PlanModifyString(_ context.Context, request planmodifier.StringRequest, response *planmodifier.StringResponse) {
	if !request.PlanValue.IsUnknown() && !request.PlanValue.IsNull() {
		imageWithoutLibrary := strings.TrimPrefix(request.PlanValue.ValueString(), "library/")
		response.PlanValue = types.StringValue(imageWithoutLibrary)
	}
}
