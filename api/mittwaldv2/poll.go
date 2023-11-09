package mittwaldv2

import (
	"context"
	"errors"
	"time"
)

func poll[T any](ctx context.Context, f func() (T, error)) (T, error) {
	var null T

	res := make(chan T)
	err := make(chan error)

	t := time.NewTicker(200 * time.Millisecond)

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

			r, e := f()
			if e != nil {
				if notFound := (ErrNotFound{}); errors.As(e, &notFound) {
					continue
				} else if permissionDenied := (ErrPermissionDenied{}); errors.As(e, &permissionDenied) {
					continue
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
		return null, ctx.Err()
	case r := <-res:
		return r, nil
	case e := <-err:
		return null, e
	}
}
