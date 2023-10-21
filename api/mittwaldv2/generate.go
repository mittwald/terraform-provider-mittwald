package mittwaldv2

// NOTE: This needs a patched version of oapi-codegen; PR #1178 [1] is
// needed to generate the correct code for the Mittwald API.
//
//   [1]: https://github.com/deepmap/oapi-codegen/pull/1178

//go:generate wget api.mittwald.de/openapi -O openapi.json && oapi-codegen -package mittwaldv2 openapi.json > mittwald.gen.go
