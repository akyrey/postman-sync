# Changelog

All notable changes to postman-sync are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses
[Semantic Versioning](https://semver.org/).

Releases take their notes from the matching `## [vX.Y.Z]` section below, so
add a section before pushing a tag.

## [v0.2.0] - 2026-10-09

### Added

- **Multi-file OpenAPI specs**: `openapi.path` can point to a directory. All
  external `$ref`s are resolved and bundled before import. The root spec is
  auto-detected by well-known names, or set with `openapi.root_file` /
  `--openapi-root-file`.
- **Three-way merge with baselines**: after each successful `openapi-sync`, the
  collection generated from the spec is saved under
  `<openapi.state_dir>/baselines/` (default `.postman-sync`). The next sync
  uses it as the common ancestor:
  - values and enabled/disabled state left untouched since the last sync follow
    the spec, while edited ones are kept, field by field, including inside raw
    JSON bodies;
  - entries added by hand in Postman are kept, and entries deleted by hand stay
    deleted;
  - untouched bodies are replaced by the spec's body.

  Requests without a baseline fall back to the two-way rules below.
- **Preserve request data on update** (`openapi.preserve_request_data`, default
  `true`; disable for one run with `--no-preserve`):
  - header, query param and form field values and enabled state are kept by key;
  - path variable values are kept;
  - hand-added entries are kept;
  - raw JSON bodies are merged field by field.
- **`--dry-run`** for `openapi-sync`: builds the merged collection exactly like a
  real update and prints what would change, without writing to Postman or
  saving a baseline. It reports added and removed requests and folders. Per
  request, it reports changes to the method and URL; to header, query, path
  variable and form field values, enabled state and descriptions; and to the
  body, auth, scripts and saved responses. It also reports collection-level
  auth and script changes.
- **Automatic backups**: before an existing collection is updated, it is saved
  to `<state_dir>/backups/<UTC timestamp>/collections/<name>.json`, so it can
  be restored with
  `import --input-dir <state_dir>/backups/<timestamp> --collections all --strategy overwrite`.
  - The most recent `openapi.backup.keep` backups are kept per collection
    (default 10; `0` keeps all).
  - A failed backup aborts the sync before anything changes.
  - Configured with `openapi.backup.enabled` or `--no-backup`. Dry runs never
    back up.
- `openapi.common_query_params`: query parameters injected into every request.
- `openapi.import_options`: options forwarded to Postman's OpenAPI importer.

### Changed

- Postman's importer is now called with `enableOptionalParameters: false` by
  default. Optional params and headers are generated disabled, and only those
  the spec marks as required are enabled. Override this with
  `openapi.import_options`.
- `common_headers` and `common_query_params` are re-applied after the merge, so
  the configured value and disabled state always override edits made in
  Postman.
- Merged folders keep their existing ID instead of taking the temporary import
  collection's ID.

### Fixed

- **Data loss on update, export and import**: collections were decoded into
  structs that modelled only part of the Postman format, so re-encoding them
  silently dropped:
  - header and parameter descriptions, which carry the `(Required)` marker and
    type info;
  - body options and form-data fields;
  - other unmodelled fields.

  Every collection and environment type now keeps unmodelled fields, so a
  fetch → update round trip is lossless.
- Required/optional state from the spec was not reflected in generated requests
  because Postman's importer enabled every optional parameter.
- Each request's raw URL lost its query string when the base URL was set,
  causing a spurious URL change on every sync. The raw URL is now rebuilt from
  the enabled query params, as Postman does.
- The temporary import collection is deleted even when fetching it fails.

### Upgrade notes

- Existing collections created with earlier versions have every optional
  parameter enabled. Run `openapi-sync --dry-run --no-preserve` to preview, then
  `openapi-sync --no-preserve` once. This resets requests to the spec and writes
  the first baseline. Later syncs use the three-way merge.
- Add `.postman-sync/` (or your `openapi.state_dir`) to `.gitignore`.

## [v0.1.0] - 2026-03-24

### Added

- `export` command: exports collections and/or environments from a Postman
  workspace to local JSON files (one file per entity, in the full API envelope).
- `import` command: imports collections and/or environments from local JSON
  files, with `overwrite` or `merge` strategies for collections.

### Changed

- The original sync behaviour moved to the `openapi-sync` subcommand. The root
  command now dispatches to `openapi-sync`, `export` and `import`, and
  `--config` is shared by all of them.
- The configuration file is split into per-command sections: `openapi:`,
  `export:` and `import:`. Global `postman_api_key` and `workspace_id` remain
  at the root.

[v0.2.0]: https://github.com/akyrey/postman-sync/compare/v0.1.0...v0.2.0
[v0.1.0]: https://github.com/akyrey/postman-sync/compare/v0.0.1...v0.1.0
