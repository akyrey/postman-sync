# postman-sync

A Go CLI for managing Postman workspaces:

- **`openapi-sync`** — sync a Postman collection from an OpenAPI spec **without** overwriting team-specific customizations (auth, scripts, saved responses, etc.)
- **`export`** — export collections and environments to local JSON files
- **`import`** — import collections and environments from local JSON files back into Postman

Inspired by [dmiska25/postman_sync.py](https://gist.github.com/dmiska25/e807fe4642f97170d0c1ab7f5bbf113e), rewritten in Go with a YAML-driven configuration.

## How openapi-sync works

1. Loads an OpenAPI spec (JSON or YAML) from disk
2. Optionally sanitizes enum values to reduce diff noise
3. Imports the spec into Postman via the API (creates a temporary collection)
4. Downloads the generated collection, then deletes the temporary one
5. Transforms the collection: flattens single-request folders, sorts alphabetically, injects configured headers/auth/scripts, sets the base URL, and adds doc links
6. Merges the transformed collection into an existing Postman collection of the same name, **preserving** auth, pre-request scripts, test scripts, and saved responses from previous manual edits
7. Endpoints removed from the spec are removed from the collection (spec is source of truth)

## Prerequisites

- Go 1.22+
- A [Postman API key](https://learning.postman.com/docs/developer/postman-api/authentication/)
- A Postman workspace ID

## Installation

```bash
go install github.com/akyrey/postman-sync@latest
```

Or build from source:

```bash
git clone https://github.com/akyrey/postman-sync.git
cd postman-sync
make build
```

## Configuration

Copy the example config and fill in your values:

```bash
cp postman-sync.example.yaml postman-sync.yaml
```

Secrets can be set via environment variables instead of the config file:

```bash
export POSTMAN_API_KEY=your-api-key
export POSTMAN_WORKSPACE_ID=your-workspace-id
```

### Config structure

The config file has a global section and three optional command-specific sections:

```yaml
# Global (required)
postman_api_key: ""    # or POSTMAN_API_KEY env var
workspace_id: ""       # or POSTMAN_WORKSPACE_ID env var

# openapi-sync command settings
openapi:
  path: "./openapi.json"
  base_url: "{{baseUrl}}"
  sanitize_enums: true
  # ... auth, scripts, headers, folder_overrides, doc_links

# export command settings
export:
  output_dir: "./postman-export"
  collections: ["all"]
  environments: ["all"]
  pretty: true

# import command settings
import:
  input_dir: "./postman-export"
  collections:
    names: ["all"]
    strategy: "overwrite"
  environments:
    names: ["all"]
```

### Global fields

| Field | Required | Description |
|---|---|---|
| `postman_api_key` | Yes | Postman API key (or `POSTMAN_API_KEY` env var) |
| `workspace_id` | Yes | Postman workspace ID (or `POSTMAN_WORKSPACE_ID` env var) |

### openapi section

| Field | Default | Description |
|---|---|---|
| `openapi.path` | `./openapi.json` | Path to the OpenAPI spec (JSON or YAML) |
| `openapi.base_url` | `{{baseUrl}}` | Base URL for all requests (Postman variable recommended) |
| `openapi.sanitize_enums` | `true` | Replace enum values with `<enum>` to reduce diff noise |
| `openapi.doc_links.base_url` | — | Base URL for documentation links (omit to disable) |
| `openapi.common_headers` | `[]` | Headers injected into every request |
| `openapi.auth` | — | Collection-level authentication |
| `openapi.auth.propagation` | — | Set to `"inherit"` to clear auth on all folders/requests so they inherit from the collection |
| `openapi.scripts` | — | Collection-level pre-request and test scripts |
| `openapi.folder_overrides` | `{}` | Per-folder (tag) auth and script overrides |

### export section

| Field | Default | Description |
|---|---|---|
| `export.output_dir` | `./postman-export` | Directory for exported files |
| `export.collections` | `[]` | Names to export, or `["all"]`. Omit to skip. |
| `export.environments` | `[]` | Names to export, or `["all"]`. Omit to skip. |
| `export.pretty` | `true` | Pretty-print JSON (recommended for git diffs) |

### import section

| Field | Default | Description |
|---|---|---|
| `import.input_dir` | `./postman-export` | Directory to read files from |
| `import.collections.names` | — | Names to import, or `["all"]` |
| `import.collections.strategy` | `overwrite` | `overwrite` or `merge` (see below) |
| `import.environments.names` | — | Names to import, or `["all"]` |
| `import.environments.strategy` | `overwrite` | `overwrite` only (environments have no merge logic) |

## Usage

```bash
# Sync an OpenAPI spec into Postman
./bin/postman-sync openapi-sync
./bin/postman-sync openapi-sync --openapi-path ./api.yaml   # override spec path
./bin/postman-sync openapi-sync --config /path/to/config.yaml

# Export collections and environments to disk
./bin/postman-sync export
./bin/postman-sync export --collections all --environments all
./bin/postman-sync export --collections "My API,Other API" --output-dir ./backup

# Import files back into Postman
./bin/postman-sync import
./bin/postman-sync import --collections all --strategy overwrite
./bin/postman-sync import --collections all --strategy merge --environments all
./bin/postman-sync import --input-dir ./backup

# Print version
./bin/postman-sync --version

# Using env vars for secrets
POSTMAN_API_KEY=xxx POSTMAN_WORKSPACE_ID=yyy ./bin/postman-sync openapi-sync
```

All flags override the corresponding config file values for that run. The `--config` flag is global and applies to all commands.

## Export file layout

```
postman-export/
  collections/
    My API.json           # full Postman CollectionWrapper JSON
    Other API.json
  environments/
    Production.json       # full Postman EnvironmentWrapper JSON
    Staging.json
```

Files are the complete Postman API envelope, so they round-trip cleanly through import. Pretty-printed JSON by default — friendly for git diffs.

## openapi-sync merge behavior

When a collection with the same name already exists in the workspace:

| What | Source |
|---|---|
| Which endpoints exist | New spec (removed endpoints are dropped) |
| Endpoint order | Alphabetical (from transform step) |
| Request URL, method, body, headers | New spec |
| Auth (item-level and folder-level) | **Preserved** from existing collection (unless `auth.propagation: inherit`) |
| Pre-request and test scripts | **Preserved** from existing collection |
| Saved example responses | **Preserved** from existing collection |
| Collection-level auth | Config file (if set), otherwise preserved from existing |
| Collection-level scripts | Config file (if set), otherwise preserved from existing |

On first sync (no existing collection), config-defined auth/scripts are applied as defaults.

## import merge strategy

When `import.collections.strategy: merge`, the import command fetches the existing collection from Postman and runs the same `MergeItems` logic used by `openapi-sync`:

- Auth, scripts, and saved responses are preserved from the existing Postman collection.
- Request definitions (URL, method, body, headers) come from the imported file.
- Endpoints not in the file are dropped.

Use `overwrite` to replace the entire collection without merging.

## Auth propagation (openapi-sync)

By default, Postman inherits auth from the parent for any item that has no explicit auth. However, after merge cycles, individual folders and requests often accumulate their own auth objects — preventing them from picking up changes to the collection-level auth.

Setting `openapi.auth.propagation: "inherit"` makes the tool clear auth on every folder and leaf request so they all inherit from the collection. Exceptions:

- Items with `auth.type: "noauth"` are left untouched.
- Folders listed in `folder_overrides` keep their explicit auth (their children are still processed).

```yaml
openapi:
  auth:
    type: "oauth2"
    attributes:
      - key: "accessToken"
        value: "{{oauth2_access_token}}"
        type: "string"
      # ... other oauth2 attributes
    propagation: "inherit"
```

## Auth types

`openapi.auth.type` supports: `apikey`, `basic`, `bearer`, `oauth1`, `oauth2`, `digest`, `ntlm`, `hawk`, `awsv4`, `edgegrid`, `noauth`.

## OAuth2 configuration

OAuth2 has two sections in Postman: the **current token** (sent with requests) and the **new token** (OAuth2 flow config). Both are set via `attributes` key/value pairs.

```yaml
openapi:
  auth:
    type: "oauth2"
    attributes:
      # --- Current Token ---
      - key: "accessToken"
        value: "{{oauth2_access_token}}"
        type: "string"
      - key: "tokenType"
        value: "Bearer"
        type: "string"
      - key: "addTokenTo"
        value: "header"
        type: "string"
      - key: "headerPrefix"
        value: "Bearer"
        type: "string"
      # --- New Token (OAuth2 flow) ---
      - key: "grant_type"
        value: "client_credentials"
        type: "string"
      - key: "accessTokenUrl"
        value: "https://auth.example.com/oauth/token"
        type: "string"
      - key: "clientId"
        value: "{{oauth2_client_id}}"
        type: "string"
      - key: "clientSecret"
        value: "{{oauth2_client_secret}}"
        type: "string"
      - key: "scope"
        value: "openid profile"
        type: "string"
```

> All attribute values must be strings (including booleans like `"false"`).

## Project structure

```
postman-sync/
├── cmd/postman-sync/
│   ├── main.go             # Cobra root command, global flags, shared helpers
│   ├── openapi_sync.go     # openapi-sync subcommand + pipeline
│   ├── export.go           # export subcommand + pipeline
│   ├── import_cmd.go       # import subcommand + pipeline
│   └── fileutil.go         # Shared file I/O helpers
├── internal/
│   ├── config/
│   │   └── config.go       # Config structs, loading, per-command validation
│   ├── openapi/
│   │   └── loader.go       # Load OpenAPI spec (JSON/YAML) + enum sanitization
│   └── postman/
│       ├── types.go        # Postman Collection v2.1 + Environment types
│       ├── client.go       # Postman API HTTP client (collections + environments)
│       ├── transform.go    # Collection transformers
│       └── merge.go        # Name-based recursive merge
├── postman-sync.example.yaml
├── Makefile
├── .goreleaser.yaml
├── go.mod
└── go.sum
```

## Testing

```bash
make test         # run all tests
make test-race    # with race detector
make test-cover   # with coverage report
```

## License

GNU General Public License
