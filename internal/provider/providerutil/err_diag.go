package providerutil

import "github.com/hashicorp/terraform-plugin-framework/diag"

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
	diag    *diag.Diagnostics
	summary string
}

func (w *WrappedError[T]) Do(err error) {
	if err != nil {
		w.diag.AddError(w.summary, err.Error())
	}
}

func (w *WrappedError[T]) DoVal(res T, err error) T {
	w.Do(err)
	return res
}

func Try[T any](d *diag.Diagnostics, summary string) *WrappedError[T] {
	return &WrappedError[T]{d, summary}
}

func EmbedDiag[T any](resultValue T, resultDiag diag.Diagnostics) func(outDiag *diag.Diagnostics) T {
	return func(out *diag.Diagnostics) T {
		out.Append(resultDiag...)
		return resultValue
	}
}
