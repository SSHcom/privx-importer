# PrivX Inventory Import — Usage Guide

## Prerequisites

- **Go** 1.22+ installed ([download](https://go.dev/dl/))
- Network access to your PrivX instance
- PrivX API client credentials (see [Authentication](#authentication))

## Getting Started

### 1. Clone the repository

```bash
git clone <repository-url>
cd inventory_import_privx_dev
```

### 2. Install Go dependencies

```bash
go mod download
```

Or if `go.mod` / `go.sum` are missing or stale:

```bash
go mod tidy
```

### 3. Build (optional)

```bash
go build -o privx-import ./cmd/privx-import
```

This produces a `privx-import` binary in the current directory. You can also run directly with `go run` without building.

### 4. Verify the setup

```bash
go run cmd/privx-import/main.go --help
```

---

## Authentication

The tool supports two authentication methods. JWT takes priority if both are configured.

### Method 1: JWT Token

Generate a signed JWT and exchange it for a PrivX access token.

```bash
export PRIVX_BASE_URL=https://your-privx-instance.example.com

export PRIVX_JWT=$(python3 scripts/make_test_jwt.py \
    --key jwt_private.pem \
    --alg RS256 \
    --iss importer \
    --sub importer \
    --aud $PRIVX_BASE_URL)
```

**Requirements:**
- A private key file (RSA or Ed25519) registered in PrivX
- Python 3 with the `PyJWT` package (`pip install pyjwt cryptography`)

### Method 2: OAuth Client Credentials

Use API client credentials from the PrivX admin UI. This requires **all four** environment variables:

```bash
export PRIVX_BASE_URL=https://your-privx-instance.example.com
export PRIVX_OAUTH_CLIENT_ID=<oauth_client_id>
export PRIVX_OAUTH_CLIENT_SECRET=<oauth_client_secret>
export PRIVX_CLIENT_ID=<api_client_id>
export PRIVX_CLIENT_SECRET=<api_client_secret>
```

These values come from **Administration → Deployment → API Clients** in the PrivX GUI. Expand the "Credentials" section of your API client to find them.

Under the hood this calls `POST /auth/api/v1/oauth/token` with:
- HTTP Basic Auth: `oauth_client_id:oauth_client_secret`
- Form body: `grant_type=password&username=<api_client_id>&password=<api_client_secret>`

> **Note:** PrivX does not support a simple 2-credential `client_credentials` grant for API clients. All four credentials are always required for OAuth.

### View configured environment variables

```bash
go run cmd/privx-import/main.go show env
```

---

## Configuration

Configuration is loaded with the following priority (highest wins):

1. CLI flags (`--update-mode`, `--delete-mode`)
2. Environment variables (`PRIVX_*`)
3. Config file (`config.toml`)
4. Hard-coded defaults

The tool reads configuration from a directory (default: `internal/config`).

| File | Purpose |
|------|---------|
| `config.toml` | Runtime settings (update mode, delete mode, scope, allowed services) |
| `host.toml` | Field catalog defining all importable host fields and their types |

Override the config directory with `--config-dir`:

```bash
go run cmd/privx-import/main.go --config-dir ./my-config plan input.csv
```

### config.toml example

```toml
# Update mode: addonly | modify | sync
update_mode = "sync"

# Delete mode: disable | delete
# delete_mode = "disable"

# Default field management: managed | ignored
default_management_mode = "managed"

# Allowed service types for validation
allowed_services = ["SSH", "RDP", "VNC", "WEB", "DB"]

# Scope restricts which PrivX hosts can be modified
[scope]
tags = "import-managed"
```

### Update Modes

| Mode | Creates | Modifies | Deletes |
|------|---------|----------|---------|
| `addonly` | ✅ | ❌ | ❌ |
| `modify` | ✅ | ✅ | ❌ |
| `sync` | ✅ | ✅ | ✅ |


### Delete Modes

| Mode | Behavior |
|------|----------|
| `disable` | Hosts missing from input are disabled (default) |
| `delete` | Hosts missing from input are deleted |

Note that disabled hosts will not be deleted even in `delete` mode.

### Scope

The scope is defined by field values.
Only hosts that include those values will be managed by Importer.
List values, like `tags`, can include also other values.

If an existing host is matched by a row in CSV, but that host does not match the scope, the host in PrivX will not be modified.

Hosts created by the tool automatically get scope values applied, if they are not already present in the input.

### Field selection

By default, Importer will manage all fields.
It means that any field read from CSV will overwrite the corresponding value in PrivX.
The rest of the fields will retain their values.

This is the default management mode `managed`.

Individual field can be ignored by setting the `management = "ignored"` flag in the field catalog ([internal/config/host.toml](../internal/config/host.toml)).
This needs to be set at least for automatically updated timestamp fields like `created`.
Otherwise those fields will always trigger an update of the host.

If management mode is set to `ignored`, only those fields that have explicitly set `management = "managed"` will be handled.

---

## Commands

### plan — Preview changes without applying

```bash
go run cmd/privx-import/main.go plan <input-file> [input-file...]
```

Shows what would be created, updated, or deleted in PrivX.

**Examples:**

```bash
# Single file
go run cmd/privx-import/main.go plan testdata/simple.csv

# Multiple files (records are merged across files)
go run cmd/privx-import/main.go plan hosts-dc1.csv hosts-dc2.csv

# Verbose output
go run cmd/privx-import/main.go -vvv plan testdata/full.csv

# Override update mode
go run cmd/privx-import/main.go --update-mode sync plan hosts.csv
```

### apply — Apply changes to PrivX

```bash
go run cmd/privx-import/main.go apply <input-file> [input-file...]
```

Shows the plan, prompts for confirmation, then applies changes.

**Examples:**

```bash
# Interactive (prompts before applying)
go run cmd/privx-import/main.go apply testdata/simple.csv

# Skip confirmation prompt
go run cmd/privx-import/main.go apply --auto-approve testdata/simple.csv

# Override delete mode for this run
go run cmd/privx-import/main.go apply --delete-mode delete hosts.csv

# Verbose
go run cmd/privx-import/main.go -vvv apply testdata/full.csv
```

### show — Inspect PrivX data and configuration

```bash
# Show environment variables and their current values
go run cmd/privx-import/main.go show env

# Show active runtime configuration
go run cmd/privx-import/main.go show config

# Show all available CSV column names
go run cmd/privx-import/main.go show fields

# Show access groups from PrivX (requires auth)
go run cmd/privx-import/main.go show accessgroups

# Show roles from PrivX (requires auth)
go run cmd/privx-import/main.go show roles
```

---

## Input File Format

The tool reads CSV files with a header row. Each row represents a host or a service on a host.

### Minimal example

```csv
common_name,addresses,tags
web-01,"10.0.0.10",prod
db-01,"10.0.1.10",prod
```

### Host identification

Each row must contain at least one identifying field:

| Field | Description |
|-------|-------------|
| `common_name` (aliases: `host_name`, `hostname`) | DNS hostname |
| `id` (alias: `host_id`) | PrivX UUID |
| `addresses` (aliases: `ip`, `ip_address`) | IP address(es) |

Priority for matching: `id` > `common_name` > `addresses`

### Multi-row hosts

Multiple rows with the same host identifier are merged into a single host. List fields (like `tags`, `services`) are combined; scalar fields must not conflict.

```csv
common_name,services.service,services.port,services.address,tags
web-01,SSH,22,10.0.0.10,prod
web-01,RDP,3389,10.0.0.10,windows
```

### Supported field types

| Type | Behavior | Example fields |
|------|----------|----------------|
| `string` | Single value, conflicts produce an error | `common_name`, `comment`, `zone` |
| `bool` | Accepts true/1/yes/y/on or false/0/no/n/off | `audit_enabled`, `tofu`, `deployable` |
| `int` | Integer value | `services.port` |
| `list` | Comma-separated | `tags`, `addresses`, `scope` |
| `composite` | Nested object | `password_rotation`, `host_certificate` |
| `list_composite` | Array of nested objects | `services`, `principals`, `ssh_host_public_keys` |

Composite fields (objects) are specified in CSV with a column name in dot notation, 
for example `session_recording_options.disable_clipboard_recording`.

Likewise for list_composite fields. 
The values in the first row for the host will become first element of the list.
Subsequent rows will add elements to the list in order of appearance.

### Translators

Some fields support automatic name-to-UUID translation:
- `access_group_id` — Access group name → UUID
- `principals.roles.id` — Role name → UUID
- `services.service` — Case-insensitive normalization to allowed values

The UUID fields can be specified in CSV either in name form or directly as UUID.
They will be displayed in name form and sent to PrivX in UUID form.

---

## Verbosity Levels

| Flag | Level | Output |
|------|-------|--------|
| (none) | Normal | Plan summary, errors, warnings |
| `-v` | Info | Auth method, config path, unchanged hosts |
| `-vv` | Debug | Field catalog, validated data, API responses |
| `-vvv` | Trace | Full request/response bodies, translator details |

---

## Environment Variables Reference

| Variable | Required | Description |
|----------|----------|-------------|
| `PRIVX_BASE_URL` | Yes | PrivX instance URL |
| `PRIVX_JWT` | Auth method 1 | Signed JWT token |
| `PRIVX_OAUTH_CLIENT_ID` | Auth method 2 | OAuth client ID (Basic Auth) |
| `PRIVX_OAUTH_CLIENT_SECRET` | Auth method 2 | OAuth client secret (Basic Auth) |
| `PRIVX_CLIENT_ID` | Auth method 2 | API client ID (form body) |
| `PRIVX_CLIENT_SECRET` | Auth method 2 | API client secret (form body) |
| `PRIVX_DELETE_MODE` | No | `disable` (default) or `delete` |
| `PRIVX_UPDATE_MODE` | No | `addonly` (default), `modify`, or `sync` |
| `PRIVX_MANAGE_MODE` | No | `managed` (default) or `ignored` |
| `PRIVX_MAX_RECORDS` | No | Max CSV records per file (default: 100000) |
| `PRIVX_ALLOWED_SERVICES` | No | Comma-separated service list |
| `PRIVX_ALLOWED_DB_PROTOCOLS` | No | Comma-separated DB protocol list |

You can get an up to date list with current values with Importer command `show env`.

---

## Running Tests

```bash
go test ./...
```

---

## Troubleshooting

**"no required module provides package ... go.mod file not found"**
Run `go mod tidy` in the repository root.

**"no JWT or OAuth credentials"**
Set either `PRIVX_JWT` or all four OAuth variables. Run `show env` to check.

**"oauth client id empty" / "oauth client secret empty"**
OAuth requires all four credentials (`PRIVX_CLIENT_ID`, `PRIVX_CLIENT_SECRET`, `PRIVX_OAUTH_CLIENT_ID`, `PRIVX_OAUTH_CLIENT_SECRET`).

**400 Bad Request on /oauth/token**
Verify your credentials are correct and that the API client has the required roles assigned in PrivX.

**"max records exceeded"**
Your CSV file exceeds the configured limit. Increase `PRIVX_MAX_RECORDS` or split the file.

**"Update mode does not allow modification/deletion"**
Change `--update-mode` to `modify` or `sync` depending on what operations you need.

**TLS certificate errors**
If your PrivX instance uses a self-signed certificate, add the CA to your system trust store or set `SSL_CERT_FILE`.
