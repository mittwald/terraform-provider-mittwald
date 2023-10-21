package mittwaldv2

import (
	"fmt"
	"net/http"
)

type ErrNotFound struct{}

func (e ErrNotFound) Error() string {
	return "resource not found"
}

func errUnexpectedStatus(status int, body []byte) error {
	if status == http.StatusNotFound {
		return ErrNotFound{}
	}

	return fmt.Errorf("unexpected status code %d: %s", status, string(body))
}
