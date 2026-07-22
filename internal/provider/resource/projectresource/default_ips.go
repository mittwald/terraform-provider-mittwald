package projectresource

import (
	"context"
	"errors"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/mittwald/api-client-go/pkg/httperr"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiext"
)

const (
	// DefaultCreateTimeout is the default timeout for creating a project. Most
	// of it is spent waiting for the project's default ingress (and its IP
	// addresses) to be provisioned, which is done asynchronously by the
	// platform and can occasionally take several minutes.
	DefaultCreateTimeout = 10 * time.Minute

	// DefaultReadTimeout is the default timeout for reading a project. It also
	// covers waiting for a not-yet-provisioned default ingress; if the ingress
	// does not show up in time, the read degrades to an empty list of default
	// IP addresses instead of failing.
	DefaultReadTimeout = 2 * time.Minute

	// createTimeoutHint and readTimeoutHint are appended to diagnostics that are
	// caused by an exhausted timeout, to point users at the knob they can turn.
	createTimeoutHint = "If this happens regularly, increase the `timeouts.create` value on this resource."
	readTimeoutHint   = "If this happens regularly, increase the `timeouts.read` value on this resource."
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
