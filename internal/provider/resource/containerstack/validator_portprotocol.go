package containerstackresource

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = &PortProtocolValidator{}

var allowedPorts = []string{"tcp"}

// PortProtocolValidator is a type used to validate that a port protocol is valid and supported, currently limited to "tcp".
type PortProtocolValidator struct{}

func (p *PortProtocolValidator) Description(_ context.Context) string {
	return "Asserts that the port protocol is a valid (and supported) network protocol."
}

func (p *PortProtocolValidator) MarkdownDescription(ctx context.Context) string {
	return p.Description(ctx)
}

func (p *PortProtocolValidator) ValidateString(_ context.Context, request validator.StringRequest, response *validator.StringResponse) {
	port := request.ConfigValue.ValueString()

	for _, allowed := range allowedPorts {
		if port == allowed {
			return
		}
	}

	response.Diagnostics.AddAttributeError(request.Path, "Invalid Port Protocol", fmt.Sprintf("The port protocol must be one of: %v", allowedPorts))
}
