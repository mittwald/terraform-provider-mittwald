package provider

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
