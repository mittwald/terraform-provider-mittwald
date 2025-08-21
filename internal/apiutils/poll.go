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

func Poll[TReq any, TRes any](ctx context.Context, o PollOpts, f func(context.Context, TReq, ...func(req *http.Request) error) (TRes, *http.Response, error), req TReq) (TRes, error) {
	var null TRes

	res := make(chan TRes)
	err := make(chan error)

	o.applyDefaults()

	d := o.InitialDelay
	t := time.NewTicker(d)

	defer func() {
		t.Stop()
		close(res)
		close(err)
	}()

	go func() {
		for {
			if _, ok := <-t.C; !ok {
				return
			}

			d = time.Duration(math.Max(float64(d)*o.BackoffFactor, float64(o.MaxDelay)))
			t.Reset(d)

			r, _, e := f(ctx, req)
			if e == nil {
				res <- r
				return
			}

			if errors.Is(e, ErrPollShouldRetry) {
				continue
			} else if notFound := new(httperr.ErrNotFound); errors.As(e, &notFound) {
				continue
			} else if permissionDenied := new(httperr.ErrPermissionDenied); errors.As(e, &permissionDenied) {
				continue
			} else if errors.Is(e, context.DeadlineExceeded) {
				return
			} else {
				err <- e
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		return null, ctx.Err()
	case r := <-res:
		return r, nil
	case e := <-err:
		tflog.Debug(ctx, "polling failed", map[string]any{"error": e})
		return null, e
	}
}
