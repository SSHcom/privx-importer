# Changelog

All notable changes to this project will be documented in this file.

---

## [1.0.0] — Initial Release

PrivX Inventory Importer is a CLI tool that imports infrastructure inventory data into PrivX from CSV sources. It normalizes records, validates them, compares the desired state against current PrivX configuration, and produces a safe execution plan before applying changes.

### Highlights

- **Plan/Apply workflow** — Preview all changes before committing them to PrivX. The `plan` command shows what would be created, updated, or deleted; `apply` executes the plan after confirmation.
- **CSV-based inventory import** — Define your infrastructure in CSV files with support for multi-row hosts, nested fields (dot notation), and list values.
- **Configurable field catalog** — Declare field types, hierarchy, and management modes via TOML configuration. Supports string, bool, int, list, composite, and list_composite field types.
- **Validation and normalization pipeline** — Input data is normalized, translated, restructured, and validated before planning.
- **Human ↔ machine value translators** — Automatically resolve access group names to UUIDs, role names to UUIDs, and normalize service types.
- **Scope-restricted updates** — Define a scope (e.g., by tags) to limit which PrivX hosts the tool can manage. Hosts outside scope are never modified.
- **Multiple update modes** — `addonly` (create only), `modify` (create + update), `sync` (create + update + delete).
- **Multiple delete modes** — `disable` (mark missing hosts as disabled) or `delete` (permanently remove).
- **Dual authentication** — Supports JWT token exchange and OAuth client credentials (4-credential flow).
- **TLS enforcement** — All PrivX API communication requires TLS 1.2+.
- **Verbose output levels** — Four verbosity levels (normal, info, debug, trace) for troubleshooting.
- **Show commands** — Inspect environment variables, active configuration, available fields, access groups, and roles.

### Commands

| Command | Description |
|---------|-------------|
| `plan <files...>` | Preview changes without applying |
| `apply <files...>` | Apply changes to PrivX (with confirmation prompt) |
| `show env` | Display configured environment variables |
| `show config` | Display active runtime configuration |
| `show fields` | List all available CSV column names |
| `show accessgroups` | List access groups from PrivX |
| `show roles` | List roles from PrivX |

### Configuration

- Runtime settings via `config.toml` (update mode, delete mode, scope, allowed services)
- Field catalog via `host.toml` (field types, hierarchy, validators, translators, management mode)
- Override config directory with `--config-dir`
- CLI flags and environment variables override file-based configuration

### Requirements

- Go 1.22+
- Network access to a PrivX instance (v42–v44 tested)
- Valid API credentials (JWT or OAuth)

### Known Limitations

- CSV is the only supported input format (JSON/XML readers are not yet implemented)
- No token refresh — long-running imports may fail if the token expires
- All PrivX hosts are fetched on every run (no server-side scope filtering)
- Sequential apply — no concurrent API calls during the apply phase
- No retry logic for transient API failures
- No machine-readable output format (JSON/structured output not yet available)
- Limited test coverage (unit tests for the `field` package only)

See [docs/LIMITATIONS.md](docs/LIMITATIONS.md) for the full list.

### License

Apache License 2.0 — Copyright 2026 SSH Communications Security Oyj
