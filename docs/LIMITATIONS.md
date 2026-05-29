# Known Limitations & Issues

## Input Formats

- **Only CSV is supported.** The reader detects `.json` and `.xml` extensions but no reader implementations exist for those formats. Attempting to use JSON or XML files will fail with "unsupported input format."

## Authentication

- **OAuth requires all 4 credentials.** PrivX does not support a simple 2-credential `client_credentials` grant for API clients. You must provide `PRIVX_CLIENT_ID`, `PRIVX_CLIENT_SECRET`, `PRIVX_OAUTH_CLIENT_ID`, and `PRIVX_OAUTH_CLIENT_SECRET`.
- **No token refresh.** The access token obtained at startup is used for the entire session. Long-running imports on large inventories may fail if the token expires mid-operation.
- **No custom CA certificate option.** TLS verification is enforced (TLS 1.2+) but there is no CLI flag or env var to specify a custom CA bundle. You must add the CA to the system trust store or use `SSL_CERT_FILE`.

## Performance & Scalability

- **All PrivX hosts are fetched on every run.** The tool downloads the entire host inventory from PrivX (paginated at 200 per request) regardless of scope. For large deployments (10,000+ hosts), this can be slow.
- **No server-side scope filtering.** Scope filtering happens client-side after fetching all hosts. The PrivX search API is called with an empty filter.
- **Sequential apply.** Host create/update/delete operations are executed one at a time. No concurrent API calls are made during the apply phase.
- **No retry logic.** Transient API failures (network timeouts, 5xx errors) are not retried. The tool reports the error and moves to the next host.
- **No rate limiting.** The tool does not throttle API requests. High-volume imports may trigger PrivX rate limits if configured.

## Data Handling

- **`disabled` field is a string, not a boolean.** The PrivX API returns `"FALSE"`, `"BY_ADMIN"`, or `"BY_LICENSE"` (uppercase). The field is treated as a string with a default of `"FALSE"`. This is a PrivX API quirk.
- **Host matching is first-match.** When comparing target hosts to current PrivX hosts, the first match by any shared identifier (id, common_name, addresses) is used. If multiple hosts share partial identifiers, matching may be non-deterministic.
- **Scalar merge conflicts are fatal per-host.** If two CSV rows for the same host have different values for a scalar field (e.g., different `comment`), the host is skipped entirely with a diagnostic error.
- **Empty string means "not set" for string fields.** There is no way to explicitly set a string field to empty to clear an existing value.

## Planning & Apply

- **Plan always requires PrivX connectivity.** There is no offline/dry-run mode. Even `plan` connects to PrivX to fetch current host state and resolve access group/role names.
- **Delete is scope-aware but not reversible.** In `sync` mode with `delete_mode = "delete"`, hosts within scope that are missing from input will be permanently deleted.
- **Disabled hosts are skipped for deletion.** If a host is already disabled and the delete mode is `disable`, no action is taken (avoids redundant API calls).
- **`stand_alone_host` field is ignored.** Marked as `management = "ignored"` in host.toml because the PrivX API does not support updating this field.

## Configuration

- **Default config-dir is `internal/config`.** This is a development-oriented default. In production, you should always specify `--config-dir` pointing to your deployment configuration.
- **`--config mapping.yaml` flag on plan command is unused.** The flag exists but the value is never read. Configuration comes from `--config-dir` only.
- **`--input` marked as required but unused.** Both `plan` and `apply` commands call `cmd.MarkFlagRequired("input")` but inputs come from positional arguments, not a `--input` flag. This has no user-visible effect.

## Field Catalog

- **SSH key validator is commented out.** The `ssh_key` validator exists in code but is commented out in `host.toml` for the `ssh_host_public_keys.key` field.
- **No field-level update modes.** You cannot configure per-field behavior (e.g., "only add to this list, never remove"). The overlay mechanism replaces entire field values.
- **Translator errors skip the field.** If a translator fails (e.g., access group name not found), the field value is kept as-is with a warning diagnostic. This may cause API errors later.

## API Compatibility

- **Tested against PrivX v42-v44.** The tool uses the Host Store API v1. Compatibility with older or newer PrivX versions is not guaranteed.
- **Host search response format.** There is a TODO noting uncertainty about whether v43 uses `"items"` or another collection field name in search responses.
- **`HostSearch` struct sends empty filter.** The generated `HostSearch` type has many filter fields but the tool always sends an empty object, relying on client-side filtering.

## Testing

- **Limited test coverage.** Unit tests exist only for the `field` package (merge, subset, overlay, traversal). No tests for: reader, config, plan, workflow, transform, translators, validators, privx client, or CLI commands.
- **No integration tests.** There are no tests that exercise the full pipeline against a mock or real PrivX instance.

## Output & UX

- **No machine-readable output.** Plan results are printed as human-readable text only. There is no JSON or structured output mode for CI/CD integration.
- **No progress indicator.** Large imports show no progress during the PrivX host fetch or apply phases.
- **Sensitive fields are masked in output.** SSH keys, certificates, and credentials are masked, but the masking is based on a hardcoded list of field names.

## Platform

- **No Windows-specific testing.** The tool should work on Windows but path handling and terminal output have not been specifically tested.
- **Makefile is empty.** No build automation is configured.
