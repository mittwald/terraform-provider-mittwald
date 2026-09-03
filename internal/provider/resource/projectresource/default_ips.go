package projectresource

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/mittwald/api-client-go/pkg/httperr"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiext"
)

// PollDefaultIPs waits for the default IP addresses of a project to become
// available, until the given context is done.
//
// A project's default ingress is provisioned asynchronously, so it may not
// exist yet (or may not have IP addresses assigned yet). This is not treated as
// an error: if the ingress does not become available in time, this function
// emits a warning and returns no addresses, so that the caller can still
// persist the remaining project attributes. The same applies when the ingress
// is not visible due to insufficient permissions.
func PollDefaultIPs(ctx context.Context, client apiext.ProjectClient, projectID string, timeoutHint string, res *diag.Diagnostics) []string {
	ips, err := client.PollProjectDefaultIPs(ctx, projectID)
	if err == nil {
		return ips
	}

	if errors.Is(err, apiext.ErrNoDefaultIngress) {
		res.AddWarning(
			"Project has no default IP addresses (yet)",
			"The default ingress of project "+projectID+" did not become available in time, so the "+
				"`default_ips` attribute is empty. This usually resolves itself; the addresses should show up on "+
				"the next `terraform refresh` or `terraform apply`.\n\n"+timeoutHint,
		)
		return nil
	}

	if notFound := new(httperr.ErrNotFound); errors.As(err, &notFound) {
		return nil
	}

	if permissionDenied := new(httperr.ErrPermissionDenied); errors.As(err, &permissionDenied) {
		return nil
	}

	res.AddError("error while reading project ips", err.Error())

	return nil
}
