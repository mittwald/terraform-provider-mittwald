//go:build generate_api

package mittwaldv2

// NOTE: This needs a patched version of oapi-codegen; PR #1178 [1] is
// needed to generate the correct code for the Mittwald API.
//
// Run with `go generate -tags generate_api ./api/...` to generate the
// Mittwald API client.
//
//   [1]: https://github.com/deepmap/oapi-codegen/pull/1178

//go:generate wget https://api.mittwald.de/openapi -Oopenapi.json
//go:generate oapi-codegen -package mittwaldv2 -o mittwald.gen.go openapi.json
