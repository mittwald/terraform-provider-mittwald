package containerstackresource

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

// withExplicitNullUpdateSchedule rewrites a JSON request body so that
// "updateSchedule" is serialized as an explicit `null`, rather than omitted.
//
// This is needed because the generated api-client-go request types use
// `*T` with `json:"...,omitempty"` for nullable fields: a nil pointer
// always marshals to an omitted key, never to JSON `null`. The
// container-update-stack endpoint gives omission and explicit null
// different meanings ("leave unchanged" vs. "clear"), a distinction the
// generated struct cannot express on its own.
func withExplicitNullUpdateSchedule(req *http.Request) error {
	raw, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}
	_ = req.Body.Close()

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	fields["updateSchedule"] = json.RawMessage("null")

	patched, err := json.Marshal(fields)
	if err != nil {
		return err
	}

	req.ContentLength = int64(len(patched))
	req.Body = io.NopCloser(bytes.NewReader(patched))
	return nil
}
