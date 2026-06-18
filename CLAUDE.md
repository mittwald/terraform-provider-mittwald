# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Building and Installation
- `go install` - Build and install the provider binary
- `go generate` - Generate documentation and format example files

### Testing
- `make testacc` - Run acceptance tests (creates real resources, may cost money)
- `TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m` - Run acceptance tests with custom arguments
- `golangci-lint run` - Run linter checks

### Inspecting the API client
- To look up types and method signatures from the `api-client-go` dependency, use `go doc` (e.g. `go doc <module-dir>/mittwaldv2/generated/schemas/projectv2 Project`), getting the module path via `go list -m -f '{{.Dir}}' github.com/mittwald/api-client-go`. Do not grep or `sed` through the module cache.

### Development Setup
After building with `go install`, create a `~/.terraformrc` file for development overrides:
```
provider_installation {
    dev_overrides {
        "registry.terraform.io/mittwald/mittwald" = "/path/to/your/GOPATH/bin/terraform-provider-mittwald"
    }
    direct {}
}
```

## Architecture Overview

This is a Terraform provider for the mittwald cloud platform, built using the Terraform Plugin Framework v1.15.1. The codebase follows a standard provider structure:

### Core Components
- **Main Provider**: `internal/provider/provider.go` - Defines provider configuration, resources, data sources, and ephemeral resources
- **API Client**: Uses `github.com/mittwald/api-client-go` for mittwald API interactions
- **Resources**: Located in `internal/provider/resource/` with subdirectories for each resource type
- **Data Sources**: Located in `internal/provider/datasource/` with subdirectories for each data source
- **API Extensions**: `internal/apiext/` contains extended API client functionality for polling and readiness checks

### Resource Types
The provider supports these resources:
- Projects (`mittwald_project`)
- Applications (`mittwald_app`) 
- MySQL databases (`mittwald_mysql_database`)
- Redis databases (`mittwald_redis_database`)
- Cron jobs (`mittwald_cronjob`)
- Virtual hosts (`mittwald_virtualhost`)
- Container stacks (`mittwald_container_stack`)
- Container registries (`mittwald_container_registry`)
- Email outboxes (`mittwald_email_outbox`)
- Remote files (`mittwald_remote_file`)

### Key Patterns
- Each resource follows a consistent structure with separate files for models, API mapping, and resource implementation
- API polling utilities in `internal/apiutils/poll.go` for async operations
- Provider testing utilities in `internal/provider/providertesting/`
- Value conversion utilities in `internal/valueutil/`
- Custom validators and plan modifiers for complex resource types

### Authentication
- Uses `MITTWALD_API_TOKEN` environment variable for API authentication
- Provider configuration supports custom API endpoints for testing

### Dependencies
- Go 1.23.7+ required
- Terraform 1.14+ required for development
- Uses Terraform Plugin Framework v1.15.1 (not the older SDK)

## Coding style and conventions

- For sensitive attributes like passwords and API keys, use write-only attributes with a versioning field whenever possible. By convention, these attribute should be named `<attribute>_wo` and `<attribute>_wo_version`.
- Many resources require extensive mapping code between Terraform schema and API models. Follow existing patterns in the codebase for consistency, and use a `model_api_to.go` for code mapping Terraform schema _to_ API models, and `model_api_from.go` for mapping API models _from_ API responses to Terraform schema.
- Follow the documented best-practices for naming conventions as documented in https://developer.hashicorp.com/terraform/plugin/best-practices/naming
- A resource and a data source with the same name (e.g. `mittwald_project`) should mirror each other: the data source should expose the same attributes as the resource and reuse the same model-mapping helpers. When you add or change attributes on a resource, check whether a same-named data source exists and keep it in sync (and vice versa).
- Prefer looking up a single object by a server-resolvable identifier over listing and filtering client-side. Several mittwald endpoints (e.g. `GetProject`) accept both the full ID and the short ID and resolve either for you.

### Adding a resource or data source

When adding a new resource or data source, complete all of these:
- Register the constructor in `internal/provider/provider.go`.
- Add an example under `examples/{resources,data-sources}/<name>/`.
- Add it to the list in `README.md`.
- Run `go generate ./...` to regenerate the `docs/` pages.
- Prefer deprecating a superseded resource/data source (via `DeprecationMessage`) over removing it outright, since removal is a breaking change.

## Additional instructions

### Version control

- Use the conventional commit format for commit messages

### Documentation

- Run `go generate ./...` to regenerate documentation and format example files
- Under no circumstances should you edit the generated documentation files in `docs/` directly

### Test cases

- When modifying a test file (ending with `_test.go`), run the test case to make sure you didn't break anything.
