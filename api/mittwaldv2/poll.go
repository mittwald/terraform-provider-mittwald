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

	defer close(res)
	defer close(err)

	t := time.NewTicker(200 * time.Millisecond)

	go func() {
		for range t.C {
			r, e := f()
			if e != nil {
				if notFound := (ErrNotFound{}); errors.As(e, &notFound) {
					continue
				}
			} else {
				res <- r
			}
		}
	}()

	select {
	case <-ctx.Done():
		return null, ctx.Err()
	case r := <-res:
		return r, nil
	}
}
