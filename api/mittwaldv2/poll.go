package mittwaldv2

import (
	"context"
	"errors"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"math"
	"time"
)

var errPollShouldRetry = errors.New("poll should retry")

type pollOpts struct {
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
}

func (o *pollOpts) applyDefaults() {
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

func poll[T any](ctx context.Context, o pollOpts, f func() (T, error)) (T, error) {
	var null T

	res := make(chan T)
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

			r, e := f()
			if e != nil {
				if errors.Is(e, errPollShouldRetry) {
					continue
				} else if notFound := (ErrNotFound{}); errors.As(e, &notFound) {
					continue
				} else if permissionDenied := (ErrPermissionDenied{}); errors.As(e, &permissionDenied) {
					continue
				} else if errors.Is(e, context.DeadlineExceeded) {
					return
				} else {
					err <- e
					return
				}
			} else {
				res <- r
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		return null, ErrNotFound{}
	case r := <-res:
		return r, nil
	case e := <-err:
		tflog.Debug(ctx, "polling failed", map[string]any{"error": e})
		return null, e
	}
}
