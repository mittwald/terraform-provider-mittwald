package containerstackresource

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

// withExplicitNullField returns a request editor that rewrites a JSON
// request body so that fieldName is serialized as an explicit `null`,
// rather than omitted.
//
// This is needed because the generated api-client-go request types use
// `*T` with `json:"...,omitempty"` for nullable fields: a nil pointer
// always marshals to an omitted key, never to JSON `null`. Several PATCH
// endpoints (e.g. container-update-stack) give omission and explicit null
// different meanings ("leave unchanged" vs. "clear"), a distinction the
// generated struct cannot express on its own.
func withExplicitNullField(fieldName string) func(req *http.Request) error {
	return func(req *http.Request) error {
		raw, err := io.ReadAll(req.Body)
		if err != nil {
			return err
		}
		_ = req.Body.Close()

		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil {
			return err
		}
		fields[fieldName] = json.RawMessage("null")

		patched, err := json.Marshal(fields)
		if err != nil {
			return err
		}

		req.ContentLength = int64(len(patched))
		req.Body = io.NopCloser(bytes.NewReader(patched))
		return nil
	}
}
