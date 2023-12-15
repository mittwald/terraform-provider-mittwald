package mittwaldv2

import (
	"context"
	"errors"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"math"
	"time"
)

func poll[T any](ctx context.Context, f func() (T, error)) (T, error) {
	var null T

	res := make(chan T)
	err := make(chan error)

	d := 100 * time.Millisecond
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

			d = time.Duration(math.Max(float64(d)*1.1, float64(10*time.Second)))
			t.Reset(d)

			r, e := f()
			if e != nil {
				if notFound := (ErrNotFound{}); errors.As(e, &notFound) {
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
