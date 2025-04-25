package providerutil

import (
	"errors"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/mittwald/api-client-go/pkg/httperr"
	"net/http"
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
		if notFound := new(httperr.ErrNotFound); errors.As(err, &notFound) && w.ignoreNotFound {
			return
		}
		if validation := new(httperr.ErrValidation); errors.As(err, &validation) {
			for _, issue := range validation.ValidationError.ValidationErrors {
				w.diag.AddError(w.summary, "Validation error at "+issue.Path+": "+issue.Message)
			}
			return
		}
		if permissionDenied := new(httperr.ErrPermissionDenied); errors.As(err, &permissionDenied) && w.ignoreNotFound {
			return
		}
		w.diag.AddError(w.summary, err.Error())
	}
}

func (w *WrappedError[T]) DoResp(_ *http.Response, err error) {
	if err != nil {
		if notFound := new(httperr.ErrNotFound); errors.As(err, &notFound) && w.ignoreNotFound {
			return
		}
		if permissionDenied := new(httperr.ErrPermissionDenied); errors.As(err, &permissionDenied) && w.ignoreNotFound {
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

func (w *WrappedError[T]) DoValResp(res T, _ *http.Response, err error) T {
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
