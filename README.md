# PrivX inventory import module

Import infrastructure inventory data into PrivX from CSV and other external sources.

The tool normalizes imported records, validates them, compares the desired state
against current PrivX configuration, and produces a safe execution plan before applying changes.

## Features

- CSV-based inventory import
- Configurable field catalog
- Structured and nested field support
- Validation and normalization pipeline
- Safe plan/apply workflow
- Scope-restricted updates
- PrivX API integration
- Human ↔ machine value translators


## Documentation

📖 **[Usage Guide](docs/USAGE.md)** — Setup, authentication, commands, and examples.

⚠️ **[Known Limitations](docs/LIMITATIONS.md)** — Current constraints and known issues.

🏗️ **[Architecture](docs/ARCHITECTURE_DETAIL.md)** — Detailed diagrams of data flow and components.

📖 **[Documentation Index](docs/README.md)** — Architecture, development, and technical documentation.

## Workflow

```text
Input files
  -> normalize
  -> validate
  -> build target specification
  -> fetch current PrivX state
  -> create change plan
  -> apply changes
```

## Quick Start

```bash
go run cmd/privx-import/main.go plan testdata/simple.csv
```

```bash
go run cmd/privx-import/main.go apply testdata/simple.csv
```
## Known issues

- PrivX API is strict about format, so if PrivX complains of a misformed request, it is up to the user to check that the values are exactly in the right format in the CSV.
- switching to delete mode will not delete hosts that are already disabled.
- if a host has new services, principals or other multivalue list items added manually in PrivX between already existing ones, the update can corrupt those lines.

## Support & Commercial Services
This is a public open-source project licensed under Apache 2.0 and not covered by standard support SLA. Community feedback and contributions are welcome. Support is provided on a best-effort basis only. For dedicated support, customisations, or enterprise assistance, please raise a ticket via our support portal ( https://care.ssh.com) or via your local support partner. Any requests would be assigned to your account manager.