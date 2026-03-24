# CLAUDE.md

Project context for Claude Code sessions.

## What is this project?

`postman-sync` is a Go CLI tool with three commands for managing Postman workspaces:

- **`openapi-sync`** — imports an OpenAPI spec, applies transforms, and merges the result into an existing Postman collection while preserving manual customizations (auth, scripts, saved responses).
- **`export`** — exports collections and/or environments from a Postman workspace to local JSON files.
- **`import`** — imports collections and/or environments from local JSON files back into Postman (overwrite or merge strategies).

## Tech stack

- **Language**: Go 1.22+
- **Dependencies**: `gopkg.in/yaml.v3`, `github.com/spf13/cobra`
- **Build**: `make build` (outputs to `bin/postman-sync`) or `go build -o bin/postman-sync ./cmd/postman-sync`
- **Tests**: `go test ./...`

## Project structure

```
cmd/postman-sync/
  main.go          # Cobra root command, --config persistent flag, splitCSV/resolveEntities helpers
  openapi_sync.go  # openapi-sync subcommand + runOpenAPISync pipeline + stripIDs
  export.go        # export subcommand + runExport pipeline
  import_cmd.go    # import subcommand + runImport pipeline
  fileutil.go      # Shared file I/O helpers: sanitizeFilename, writeJSON, readJSON, listEntityFiles, shouldImport
internal/config/config.go    # YAML config structs (nested: OpenAPIConfig, ExportConfig, ImportConfig), loading, per-command validation
internal/openapi/loader.go   # Load JSON/YAML OpenAPI specs, enum sanitization
internal/postman/types.go    # Postman Collection v2.1 types + Environment/EnvironmentValue types
internal/postman/client.go   # Postman API HTTP client (collections + environments CRUD)
internal/postman/transform.go# Collection transformers (flatten, sort, headers, auth, scripts, base URL, doc links)
internal/postman/merge.go    # Name-based recursive merge preserving auth/events/responses
```

## Key design decisions

- **Cobra subcommands**: root command dispatches to `openapi-sync`, `export`, `import`. `--config` is a persistent flag shared by all. Per-command flags override config file values.
- **Config structure**: global fields (`postman_api_key`, `workspace_id`) at root; OpenAPI-specific config under `openapi:`; export config under `export:`; import config under `import:`. Validation is per-command (`ValidateGlobal`, `ValidateOpenAPISync`, `ValidateExport`, `ValidateImport`).
- **CollectionItem union type**: A single struct with `Items *[]CollectionItem` field. `nil` = leaf request, non-nil = folder. Check via `IsFolder()`.
- **Merge preserves customizations**: When merging old + new items by name, `auth`, `event` (scripts), and `response` (saved examples) are kept from the old collection. Request URL/method/body/headers come from the new spec. Items removed from the spec are dropped.
- **Auth propagation**: `openapi.auth.propagation: "inherit"` clears `CollectionItem.Auth` and `Request.Auth` on every folder and leaf request (except `noauth` items and folders with a `folder_override`) so they inherit from the collection. Implemented in `PropagateAuthInherit` in `internal/postman/transform.go`, called after `ApplyFolderOverrides`.
- **Client testability**: The `Client` struct has an unexported `baseURL` field and `withBaseURL()` method. Client tests use `httptest.NewServer` and are white-box (same package) to inject the test server URL.
- **Transform functions are pure**: They take `[]CollectionItem` and return/mutate in place. No API calls. Easy to test.
- **Export format**: One JSON file per entity under `<output_dir>/collections/` and `<output_dir>/environments/`. Files are the full Postman API envelope (e.g. `{"collection": {...}}`), enabling direct round-trip import. Pretty-printed by default.
- **Version injection**: The `version` variable in `cmd/postman-sync/main.go` is set at build time via `-ldflags "-X main.version=x.y.z"`. Defaults to `"dev"`.

## Commands

```bash
# Build (outputs to bin/postman-sync)
make build

# openapi-sync: import OpenAPI spec and update the Postman collection
./bin/postman-sync openapi-sync --config postman-sync.yaml
./bin/postman-sync openapi-sync --openapi-path ./api.yaml   # override spec path

# export: save collections/environments to disk
./bin/postman-sync export --collections all --environments all
./bin/postman-sync export --collections "My API,Other API" --output-dir ./backup

# import: push files back to Postman
./bin/postman-sync import --collections all --strategy overwrite
./bin/postman-sync import --collections all --strategy merge --environments all

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

Minimal structure:

```yaml
postman_api_key: ""    # or POSTMAN_API_KEY env var
workspace_id: ""       # or POSTMAN_WORKSPACE_ID env var

openapi:               # config for openapi-sync command
  path: "./openapi.json"
  base_url: "{{baseUrl}}"
  sanitize_enums: true
  # auth, scripts, common_headers, folder_overrides, doc_links...

export:                # config for export command
  output_dir: "./postman-export"
  collections: ["all"]
  environments: ["all"]
  pretty: true

import:                # config for import command
  input_dir: "./postman-export"
  collections:
    names: ["all"]
    strategy: "overwrite"   # or "merge" (collections only)
  environments:
    names: ["all"]
```

## Testing conventions

- `internal/config/` and `internal/openapi/` tests use `package <name>_test` (black-box).
- `internal/postman/` transform, merge, and types tests use `package postman_test` (black-box).
- `internal/postman/client_test.go` uses `package postman` (white-box) to access `withBaseURL()`.
- All HTTP client tests use `net/http/httptest` servers.
- Test helpers (`leafItem`, `folderItem`, `writeTemp`, `writeFile`) are defined in the respective test files.

## openapi-sync pipeline flow

1. Load config (`config.Load`) + `ValidateGlobal` + `ValidateOpenAPISync`
2. Load OpenAPI spec (`openapi.Load`), optionally sanitize enums
3. Import spec into Postman API → temporary collection UID
4. Fetch the generated collection, delete the temporary one
5. Transform: flatten → sort → headers → auth → scripts → folder overrides → auth propagation (if `inherit`) → base URL → doc links
6. Look up existing collection by name in workspace
7. If exists: fetch old, `MergeItems(old, new)`, update
8. If not: strip IDs, create new collection

## export pipeline flow

1. Load config + `ValidateGlobal` + `ValidateExport`
2. For collections: list workspace → filter by selection → fetch each → write `<output_dir>/collections/<name>.json`
3. For environments: list workspace → filter by selection → fetch each → write `<output_dir>/environments/<name>.json`

## import pipeline flow

1. Load config + `ValidateGlobal` + `ValidateImport`
2. For collections: list `.json` files in `<input_dir>/collections/` → filter by selection → for each: if exists in workspace → update (overwrite) or merge; if not → create
3. For environments: list `.json` files in `<input_dir>/environments/` → filter by selection → for each: if exists → update; if not → create

## Release

Releases are automated via GoReleaser (`.goreleaser.yaml`) and triggered by pushing a `v*` tag. The `release.yml` GitHub Actions workflow runs tests first, then GoReleaser.

Secrets required in the GitHub repository:
- `CODECOV_TOKEN`: for coverage uploads in CI
- `HOMEBREW_TAP_GITHUB_TOKEN`: personal access token with `repo` scope, needed for GoReleaser to push the Homebrew formula to `akyrey/homebrew-tap`
