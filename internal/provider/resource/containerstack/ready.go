package containerstackresource

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/mittwald/terraform-provider-mittwald/internal/apiext"
)

// waitUntilStackIsReady waits for the given stack's containers to reach the
// `running` state, until the given context is done.
//
// Running out of time is not treated as an error: at this point, the stack has
// already been created (or updated), and a hard failure would abort the
// operation before the state is written — leaving the stack untracked, or the
// state describing containers that no longer match reality. Instead, a warning
// is emitted, and the containers' actual state is picked up by the read that
// follows.
//
// Containers that are in an error state, and any other API error, are still
// reported as errors.
func waitUntilStackIsReady(ctx context.Context, client apiext.ContainerClient, stackID string, containerNames []string, timeoutHint string, d *diag.Diagnostics) {
	err := client.WaitUntilStackIsReady(ctx, stackID, containerNames)
	if err == nil {
		return
	}

	if errors.Is(err, context.DeadlineExceeded) {
		d.AddWarning(
			"Container stack did not become ready in time",
			"Not all containers of stack "+stackID+" reached the `running` state in time. They may still be "+
				"starting up; their current state will be picked up by the next `terraform plan` or "+
				"`terraform apply`.\n\n"+timeoutHint,
		)

		return
	}

	d.AddError("API error while waiting for stack to be ready", err.Error())
}
