# CLAUDE.md

Project context for Claude Code sessions.

## What is this project?

`postman-sync` is a Go CLI tool that syncs an OpenAPI specification into a Postman collection while preserving manual customizations (auth, scripts, saved responses). It uses the Postman API to import, transform, merge, and update collections.

## Tech stack

- **Language**: Go 1.22+
- **Dependencies**: `gopkg.in/yaml.v3` (only external dependency)
- **Build**: `make build` (outputs to `bin/postman-sync`) or `go build -o bin/postman-sync ./cmd/postman-sync`
- **Tests**: `go test ./...`

## Project structure

```
cmd/postman-sync/main.go     # CLI entrypoint, --config and --version flags, pipeline orchestration
internal/config/config.go    # YAML config loading, env var overrides, validation
internal/openapi/loader.go   # Load JSON/YAML OpenAPI specs, enum sanitization
internal/postman/types.go    # Postman Collection v2.1 Go structs (CollectionItem union type)
internal/postman/client.go   # Postman API HTTP client (import, get, update, delete, list)
internal/postman/transform.go# Collection transformers (flatten, sort, headers, auth, scripts, base URL, doc links)
internal/postman/merge.go    # Name-based recursive merge preserving auth/events/responses
```

## Key design decisions

- **CollectionItem union type**: A single struct with `Items *[]CollectionItem` field. `nil` = leaf request, non-nil = folder. Check via `IsFolder()`.
- **Merge preserves customizations**: When merging old + new items by name, `auth`, `event` (scripts), and `response` (saved examples) are kept from the old collection. Request URL/method/body/headers come from the new spec. Items removed from the spec are dropped.
- **Config over code**: Auth types, scripts, headers, base URL, folder overrides, and auth propagation are all driven by the YAML config file. No hardcoded values.
- **Auth propagation**: `auth.propagation: "inherit"` clears `CollectionItem.Auth` and `Request.Auth` on every folder and leaf request (except `noauth` items and folders with a `folder_override`) so they inherit from the collection. Implemented in `PropagateAuthInherit` in `internal/postman/transform.go`, called after `ApplyFolderOverrides`.
- **Client testability**: The `Client` struct has an unexported `baseURL` field and `withBaseURL()` method. Client tests use `httptest.NewServer` and are white-box (same package) to inject the test server URL.
- **Transform functions are pure**: They take `[]CollectionItem` and return/mutate in place. No API calls. Easy to test.
- **Version injection**: The `version` variable in `cmd/postman-sync/main.go` is set at build time via `-ldflags "-X main.version=x.y.z"`. Defaults to `"dev"`.

## Commands

```bash
# Build (outputs to bin/postman-sync)
make build

# Run
./bin/postman-sync --config postman-sync.yaml

# Print version
./bin/postman-sync --version

# Test all packages
make test          # or: go test ./...

# Test with race detector
make test-race

# Test with coverage report
make test-cover

# Lint
make lint          # requires golangci-lint

# Format
make fmt

# Vet
make vet

# Tidy dependencies
make tidy

# Remove build artifacts
make clean

# Show all targets
make help
```

## Config

The tool reads `postman-sync.yaml` (or path from `--config` flag). Secrets can be overridden via `POSTMAN_API_KEY` and `POSTMAN_WORKSPACE_ID` environment variables. See `postman-sync.example.yaml` for the full config reference.

## Testing conventions

- `internal/config/` and `internal/openapi/` tests use `package <name>_test` (black-box).
- `internal/postman/` transform, merge, and types tests use `package postman_test` (black-box).
- `internal/postman/client_test.go` uses `package postman` (white-box) to access `withBaseURL()`.
- All HTTP client tests use `net/http/httptest` servers.
- Test helpers (`leafItem`, `folderItem`, `writeTemp`, `writeFile`) are defined in the respective test files.

## Pipeline flow

1. Load config (`config.Load`)
2. Load OpenAPI spec (`openapi.Load`), optionally sanitize enums
3. Import spec into Postman API -> temporary collection UID
4. Fetch the generated collection, delete the temporary one
5. Transform: flatten -> sort -> headers -> auth -> scripts -> folder overrides -> auth propagation (if `inherit`) -> base URL -> doc links
6. Look up existing collection by name in workspace
7. If exists: fetch old, `MergeItems(old, new)`, update
8. If not: strip IDs, create new collection

## Release

Releases are automated via GoReleaser (`.goreleaser.yaml`) and triggered by pushing a `v*` tag. The `release.yml` GitHub Actions workflow runs tests first, then GoReleaser.

Secrets required in the GitHub repository:
- `CODECOV_TOKEN`: for coverage uploads in CI
- `HOMEBREW_TAP_GITHUB_TOKEN`: personal access token with `repo` scope, needed for GoReleaser to push the Homebrew formula to `akyrey/homebrew-tap`
