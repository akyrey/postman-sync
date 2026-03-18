# CLAUDE.md

Project context for Claude Code sessions.

## What is this project?

`postman-sync` is a Go CLI tool that syncs an OpenAPI specification into a Postman collection while preserving manual customizations (auth, scripts, saved responses). It uses the Postman API to import, transform, merge, and update collections.

## Tech stack

- **Language**: Go 1.22+
- **Dependencies**: `gopkg.in/yaml.v3` (only external dependency)
- **Build**: `go build -o postman-sync .`
- **Tests**: `go test ./...`

## Project structure

```
main.go                  # CLI entrypoint, --config flag, pipeline orchestration
config/config.go         # YAML config loading, env var overrides, validation
openapi/loader.go        # Load JSON/YAML OpenAPI specs, enum sanitization
postman/types.go         # Postman Collection v2.1 Go structs (CollectionItem union type)
postman/client.go        # Postman API HTTP client (import, get, update, delete, list)
postman/transform.go     # Collection transformers (flatten, sort, headers, auth, scripts, base URL, doc links)
postman/merge.go         # Name-based recursive merge preserving auth/events/responses
```

## Key design decisions

- **CollectionItem union type**: A single struct with `Items *[]CollectionItem` field. `nil` = leaf request, non-nil = folder. Check via `IsFolder()`.
- **Merge preserves customizations**: When merging old + new items by name, `auth`, `event` (scripts), and `response` (saved examples) are kept from the old collection. Request URL/method/body/headers come from the new spec. Items removed from the spec are dropped.
- **Config over code**: Auth types, scripts, headers, base URL, folder overrides, and auth propagation are all driven by the YAML config file. No hardcoded values.
- **Auth propagation**: `auth.propagation: "inherit"` clears `CollectionItem.Auth` and `Request.Auth` on every folder and leaf request (except `noauth` items and folders with a `folder_override`) so they inherit from the collection. Implemented in `PropagateAuthInherit` in `postman/transform.go`, called after `ApplyFolderOverrides`.
- **Client testability**: The `Client` struct has an unexported `baseURL` field and `withBaseURL()` method. Client tests use `httptest.NewServer` and are white-box (same package) to inject the test server URL.
- **Transform functions are pure**: They take `[]CollectionItem` and return/mutate in place. No API calls. Easy to test.

## Commands

```bash
# Build
go build -o postman-sync .

# Run
./postman-sync --config postman-sync.yaml

# Test all packages
go test ./...

# Test with verbose output
go test ./... -v

# Test with coverage
go test ./... -cover

# Lint
go vet ./...
```

## Config

The tool reads `postman-sync.yaml` (or path from `--config` flag). Secrets can be overridden via `POSTMAN_API_KEY` and `POSTMAN_WORKSPACE_ID` environment variables. See `postman-sync.example.yaml` for the full config reference.

## Testing conventions

- `config/` and `openapi/` tests use `package <name>_test` (black-box).
- `postman/` transform, merge, and types tests use `package postman_test` (black-box).
- `postman/client_test.go` uses `package postman` (white-box) to access `withBaseURL()`.
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
