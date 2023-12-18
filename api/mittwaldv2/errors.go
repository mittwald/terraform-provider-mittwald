package mittwaldv2

import (
	"errors"
	"fmt"
	"net/http"
)

type ErrNotFound struct{}

type ErrPermissionDenied struct{}

func (e ErrNotFound) Error() string {
	return "resource not found"
}

func (e ErrPermissionDenied) Error() string {
	return "permission denied"
}

func errUnexpectedStatus(status int, body []byte) error {
	switch status {
	case http.StatusNotFound:
		return ErrNotFound{}
	case http.StatusForbidden:
		return ErrPermissionDenied{}
	default:
		return fmt.Errorf("unexpected status code %d: %s", status, string(body))
	}
}

func IsNotFound(err error) bool {
	notFound := ErrNotFound{}
	return errors.As(err, &notFound)
}
