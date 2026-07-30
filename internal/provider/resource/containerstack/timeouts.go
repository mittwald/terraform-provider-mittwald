package containerstackresource

import "time"

const (
	// DefaultCreateTimeout is the default timeout for creating a container
	// stack. Most of it is spent waiting for the containers to actually reach
	// the `running` state, which involves pulling the respective images; how
	// long this takes depends entirely on the images in question.
	DefaultCreateTimeout = 30 * time.Minute

	// DefaultReadTimeout is the default timeout for reading a container stack.
	// This is an upper bound for a handful of (usually near-instant) API calls.
	DefaultReadTimeout = 2 * time.Minute

	// DefaultUpdateTimeout is the default timeout for updating a container
	// stack. Like DefaultCreateTimeout, most of it is spent waiting for
	// (possibly recreated) containers to reach the `running` state again.
	DefaultUpdateTimeout = 30 * time.Minute

	// DefaultDeleteTimeout is the default timeout for deleting a container
	// stack. No polling is involved here, so this is just an upper bound for a
	// single API call.
	DefaultDeleteTimeout = 10 * time.Minute

	// createTimeoutHint, readTimeoutHint and updateTimeoutHint are appended to
	// diagnostics that are caused by an exhausted timeout, to point users at the
	// knob they can turn.
	createTimeoutHint = "If this happens regularly, increase the `timeouts.create` value on this resource."
	readTimeoutHint   = "If this happens regularly, increase the `timeouts.read` value on this resource."
	updateTimeoutHint = "If this happens regularly, increase the `timeouts.update` value on this resource."
)
