package projectresource

import "time"

const (
	// DefaultCreateTimeout is the default timeout for creating a project. Most
	// of it is spent waiting for the project's default ingress (and its IP
	// addresses) to be provisioned, which is done asynchronously by the
	// platform. Usually, this takes just a few seconds; the timeout is only
	// this generous to also cover the rare occasions on which it takes minutes.
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
