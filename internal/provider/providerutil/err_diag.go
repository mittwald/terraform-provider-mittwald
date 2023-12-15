package providerutil

import (
	"errors"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/mittwald/terraform-provider-mittwald/api/mittwaldv2"
)

func ErrorValueToDiag[T any](res T, err error) func(d *diag.Diagnostics, summary string) T {
	return func(d *diag.Diagnostics, summary string) T {
		if err != nil {
			d.AddError(summary, err.Error())
		}
		return res
	}
}

func ErrorToDiag(err error) func(d *diag.Diagnostics, summary string) {
	return func(d *diag.Diagnostics, summary string) {
		if err != nil {
			d.AddError(summary, err.Error())
		}
	}
}

type WrappedError[T any] struct {
	diag           *diag.Diagnostics
	summary        string
	ignoreNotFound bool
}

func (w *WrappedError[T]) Do(err error) {
	if err != nil {
		if notFound := (mittwaldv2.ErrNotFound{}); errors.As(err, &notFound) && w.ignoreNotFound {
			return
		}
		if permissionDenied := (mittwaldv2.ErrPermissionDenied{}); errors.As(err, &permissionDenied) && w.ignoreNotFound {
			return
		}
		w.diag.AddError(w.summary, err.Error())
	}
}

func (w *WrappedError[T]) IgnoreNotFound() *WrappedError[T] {
	w.ignoreNotFound = true
	return w
}

func (w *WrappedError[T]) DoVal(res T, err error) T {
	w.Do(err)
	return res
}

func Try[T any](d *diag.Diagnostics, summary string) *WrappedError[T] {
	return &WrappedError[T]{diag: d, summary: summary}
}

func EmbedDiag[T any](resultValue T, resultDiag diag.Diagnostics) func(outDiag *diag.Diagnostics) T {
	return func(out *diag.Diagnostics) T {
		out.Append(resultDiag...)
		return resultValue
	}
}
