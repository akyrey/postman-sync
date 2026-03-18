# postman-sync

Sync a Postman collection with an OpenAPI specification **without** overwriting team-specific customizations (auth, pre/post scripts, headers, saved responses, etc.).

Inspired by [dmiska25/postman_sync.py](https://gist.github.com/dmiska25/e807fe4642f97170d0c1ab7f5bbf113e), rewritten in Go with a YAML-driven configuration for auth, scripts, headers, base URL, and per-folder overrides.

## How it works

1. Loads an OpenAPI spec (JSON or YAML) from disk
2. Optionally sanitizes enum values to reduce diff noise
3. Imports the spec into Postman via the API (creates a temporary collection)
4. Downloads the generated collection, then deletes the temporary one
5. Transforms the collection: flattens single-request folders, sorts alphabetically, injects configured headers/auth/scripts, sets the base URL, and adds doc links
6. Merges the transformed collection into an existing Postman collection of the same name, **preserving** auth, pre-request scripts, test scripts, and saved responses from previous manual edits
7. Endpoints removed from the spec are removed from the collection (spec is source of truth)

## Prerequisites

- Go 1.22+ (uses `max` builtin)
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
go build -o postman-sync .
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

### Config reference

| Field | Required | Default | Description |
|---|---|---|---|
| `postman_api_key` | Yes | - | Postman API key (or `POSTMAN_API_KEY` env var) |
| `workspace_id` | Yes | - | Postman workspace ID (or `POSTMAN_WORKSPACE_ID` env var) |
| `openapi_path` | No | `./openapi.json` | Path to the OpenAPI spec file (JSON or YAML) |
| `base_url` | No | `{{baseUrl}}` | Base URL for all requests (Postman variable recommended) |
| `sanitize_enums` | No | `true` | Replace enum values with `<enum>` to reduce diff noise |
| `doc_links.base_url` | No | - | Base URL for documentation links (omit to disable) |
| `common_headers` | No | `[]` | Headers injected into every request |
| `auth` | No | - | Collection-level authentication |
| `auth.propagation` | No | - | Set to `"inherit"` to clear auth on all folders/requests so they inherit from the collection (see below) |
| `scripts` | No | - | Collection-level pre-request and test scripts |
| `folder_overrides` | No | `{}` | Per-folder (tag) auth and script overrides |

### Auth types

The `auth.type` field supports: `apikey`, `basic`, `bearer`, `oauth1`, `oauth2`, `digest`, `ntlm`, `hawk`, `awsv4`, `edgegrid`, `noauth`.

### Auth propagation

By default, Postman inherits auth from the parent (collection or folder) for any item that has no explicit auth set. However, after a merge cycle, individual folders and requests often accumulate their own explicit auth objects from previous manual edits — which prevents them from picking up changes to the collection-level auth.

Setting `auth.propagation: "inherit"` makes the tool clear the auth field on every folder and leaf request (both `CollectionItem.Auth` and `Request.Auth`), so they all inherit from the collection. The following are always left untouched:

- Items whose auth type is `"noauth"` — they explicitly opt out and should stay that way.
- Folders listed in `folder_overrides` — the folder keeps its explicitly configured auth. Its children are still processed and will inherit from the overridden folder.

```yaml
auth:
  type: "oauth2"
  attributes:
    - key: "accessToken"
      value: "{{oauth2_access_token}}"
      type: "string"
    # ... other oauth2 attributes
  propagation: "inherit"   # clear auth on all folders/requests so they inherit from here
```

This runs after `folder_overrides` are applied in the pipeline, so override auth is always respected.

### OAuth2 configuration

OAuth2 has two conceptual sections in Postman: the **current token** (sent with requests) and the **new token** (the OAuth2 flow used to fetch a token). Both are configured via `attributes` key/value pairs.

```yaml
auth:
  type: "oauth2"
  attributes:
    # --- Current Token (what gets sent with requests) ---
    - key: "accessToken"
      value: "{{oauth2_access_token}}"
      type: "string"
    - key: "tokenType"
      value: "Bearer"
      type: "string"
    - key: "addTokenTo"
      value: "header"          # "header" or "queryParams"
      type: "string"
    - key: "headerPrefix"
      value: "Bearer"
      type: "string"

    # --- New Token (OAuth2 flow configuration) ---
    - key: "grant_type"
      value: "client_credentials"   # authorization_code | implicit | password | client_credentials
      type: "string"
    - key: "tokenName"
      value: "My API Token"
      type: "string"
    - key: "authUrl"
      value: "https://auth.example.com/oauth/authorize"
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
      value: "openid profile email"
      type: "string"
    - key: "clientAuth"
      value: "header"          # "header" (Basic auth) or "body"
      type: "string"
    - key: "redirect_uri"
      value: "https://oauth.pstmn.io/v1/callback"
      type: "string"
    - key: "useBrowser"
      value: "false"
      type: "string"
```

Required keys vary by grant type:

| Grant type | Required keys |
|---|---|
| `client_credentials` | `accessTokenUrl`, `clientId`, `clientSecret`, `scope` |
| `authorization_code` | `authUrl`, `accessTokenUrl`, `clientId`, `clientSecret`, `scope`, `redirect_uri` |
| `implicit` | `authUrl`, `clientId`, `scope`, `redirect_uri` |
| `password` | `accessTokenUrl`, `clientId`, `clientSecret`, `username`, `password`, `scope` |

> All attribute values must be strings (including booleans like `"false"`). Postman accepts this correctly.

### Example config

```yaml
postman_api_key: ""
workspace_id: ""
openapi_path: "./openapi.json"
base_url: "{{baseUrl}}"
sanitize_enums: true

common_headers:
  - key: "X-Tenant"
    value: "{{tenantId}}"
    disabled: false

auth:
  type: "bearer"
  attributes:
    - key: "token"
      value: "{{authToken}}"
      type: "string"
  # propagation: "inherit"   # uncomment to clear auth on all folders/requests

scripts:
  prerequest: |
    console.log("Pre-request running");
  test: |
    pm.test("Status is 2xx", function() {
      pm.expect(pm.response.code).to.be.within(200, 299);
    });

folder_overrides:
  "Authentication":
    auth:
      type: "noauth"
    scripts:
      test: |
        pm.test("Returns token", function() {
          pm.expect(pm.response.json().token).to.exist;
        });
```

## Usage

```bash
# Run with defaults (reads ./postman-sync.yaml)
./postman-sync

# Run with a custom config path
./postman-sync --config /path/to/config.yaml

# Using env vars for secrets
POSTMAN_API_KEY=xxx POSTMAN_WORKSPACE_ID=yyy ./postman-sync
```

## Merge behavior

When a collection with the same name already exists in the workspace:

| What | Source |
|---|---|
| Which endpoints exist | New spec (removed endpoints are dropped) |
| Endpoint order | Alphabetical (from transform step) |
| Request URL, method, body, headers | New spec |
| Auth (item-level and folder-level) | **Preserved** from existing collection (unless `auth.propagation: inherit` is set) |
| Pre-request and test scripts | **Preserved** from existing collection |
| Saved example responses | **Preserved** from existing collection |
| Collection-level auth | Config file (if set), otherwise preserved from existing |
| Collection-level scripts | Config file (if set), otherwise preserved from existing |

On first sync (no existing collection), the config-defined auth/scripts are applied as defaults.

> **Note on `auth.propagation: inherit`**: when this is enabled, item-level and folder-level auth is cleared during the transform step (before merge), so the merge will see `nil` auth on all items and preserve that. Folders with explicit `folder_overrides` keep their auth. Items with `noauth` are never touched.

## Project structure

```
postman-sync/
├── main.go                     # CLI entrypoint and pipeline orchestration
├── config/
│   └── config.go               # YAML config struct, loader, validation
├── openapi/
│   └── loader.go               # Load OpenAPI spec (JSON/YAML) + enum sanitization
├── postman/
│   ├── types.go                # Postman Collection v2.1 Go types
│   ├── client.go               # Postman API HTTP client
│   ├── transform.go            # Collection transformers (flatten, sort, headers, auth, scripts, base URL, doc links)
│   └── merge.go                # Name-based recursive merge preserving customizations
├── postman-sync.example.yaml   # Example configuration
├── go.mod
└── go.sum
```

## Testing

```bash
go test ./...
```

With coverage:

```bash
go test ./... -cover
```

## License

GNU General Public License
