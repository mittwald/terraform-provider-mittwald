package apiutils

import (
	"context"
	"errors"
	"math"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/mittwald/api-client-go/pkg/httperr"
)

var ErrPollShouldRetry = errors.New("poll should retry")

type PollOpts struct {
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
}

func (o *PollOpts) applyDefaults() {
	if o.InitialDelay == 0 {
		o.InitialDelay = 100 * time.Millisecond
	}

	if o.MaxDelay == 0 {
		o.MaxDelay = 10 * time.Second
	}

	if o.BackoffFactor == 0 {
		o.BackoffFactor = 1.1
	}
}

func PollRequest[TReq any, TRes any](ctx context.Context, o PollOpts, f func(context.Context, TReq, ...func(req *http.Request) error) (TRes, *http.Response, error), req TReq) (TRes, error) {
	return Poll[TReq, TRes](ctx, o, func(ctx context.Context, req TReq) (TRes, error) {
		res, _, err := f(ctx, req)
		return res, err
	}, req)
}

func Poll[TParam any, TRes any](ctx context.Context, o PollOpts, f func(context.Context, TParam) (TRes, error), param TParam) (TRes, error) {
	var null TRes

	o.applyDefaults()

	d := o.InitialDelay
	t := time.NewTicker(d)

	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return null, ctx.Err()
		case <-t.C:
		}

		d = time.Duration(math.Min(float64(d)*o.BackoffFactor, float64(o.MaxDelay)))
		t.Reset(d)

		r, e := f(ctx, param)

		// The polling function may have been in flight while the context was
		// cancelled; in that case, the cancellation takes precedence over
		// whatever it ended up returning.
		if ctxErr := ctx.Err(); ctxErr != nil {
			return null, ctxErr
		}

		if e == nil {
			return r, nil
		}

		if errors.Is(e, ErrPollShouldRetry) {
			continue
		} else if notFound := new(httperr.ErrNotFound); errors.As(e, &notFound) {
			continue
		} else if permissionDenied := new(httperr.ErrPermissionDenied); errors.As(e, &permissionDenied) {
			continue
		}

		tflog.Debug(ctx, "polling failed", map[string]any{"error": e})
		return null, e
	}
}
