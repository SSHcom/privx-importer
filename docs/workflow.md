# Workflow

The command-line interface is implemented using Cobra.

Implementation:
- `internal/cli/root.go`

## Configuration

Configuration is loaded from four sources in the following order. Later sources override values from earlier ones.

1. Built-in defaults
2. Configuration file
3. Environment variables
4. Command-line options

Most configuration values are initially handled as strings. They are then applied to the runtime configuration object through a common configuration layer.

Implementation:
- `internal/config/config.go`

Configuration options are defined as dedicated Go types. The same type definitions, validation rules, and helper methods are used for configuration files, environment variables, and command-line options to ensure consistent behavior across all configuration sources.

For example, command-line help texts obtain the list of allowed values directly from the corresponding type definition.

## Field Catalog

The host data structure and field definitions are loaded from:

- `internal/config/host.toml`

The definitions are parsed into a field catalog, which is used throughout the import process for field lookup, normalization, validation, encoding, and decoding.

Implementation:
- `internal/config/field_catalog.go`

## Input Processing

Input data is read from one or more source files.

The current implementation primarily supports CSV files, but the reader layer is designed so that additional source formats can be added later.

Implementation:
- `internal/reader/`

The reader phase produces raw records consisting of key-value pairs exactly as they appear in the input source.

## Normalization

Raw records are normalized into canonical field names and values.

The normalization phase performs the following operations:

1. Maps input column names to canonical field names using the field catalog.
2. Applies default values defined in the field catalog.
3. Converts values into configured field types.
4. Runs configured translators.
5. Folds flat field paths into structured field hierarchies.

The result is a collection of normalized records containing only known fields.

Implementation:
- `internal/transform/normalize.go`

## Validation

Normalized records are validated using the validators configured in the field catalog.

Validation errors are reported as diagnostics. Invalid records are excluded from further processing.

Implementation:
- `internal/transform/validate.go`

## Target Specification Construction

Validated records are grouped into target host specifications.

Multiple input records may contribute data to the same target host. Records are matched using the configured host identity fields.

During this phase field values from multiple records are merged together into a single host specification.

Implementation:
- `internal/workflow/`
- `internal/domain/`

## Translator Dictionaries

Some translators require data from PrivX in order to convert human-readable values into machine identifiers.

Examples include:

- Access group names to UUIDs
- Role names to UUIDs

Required dictionaries are loaded from PrivX before translation takes place in workflow.Prepare() phase and cached for the duration of the run.

Implementation:
- `internal/privx/`
- `internal/transform/`

## Fetching Current PrivX State

The current state of matching hosts is fetched from PrivX.

Host records returned by PrivX are decoded using the same field catalog used for input processing.

This ensures that imported data and existing PrivX data use the same internal representation.

Implementation:
- `internal/privx/`
- `internal/config/field_catalog.go`

## Planning

The planner compares target host specifications against the current PrivX state.

A host plan is generated for each host.

Possible actions include:

- Create
- Update
- Delete
- No change

The planner also applies scope restrictions and configuration rules such as update mode and delete mode.

Implementation:
- `internal/plan/`

## Apply

The apply phase executes the generated host plans against PrivX.

Depending on the selected configuration, changes may include:

- Host creation
- Host updates
- Host disabling
- Host deletion

The plan generated during the planning phase is the sole source of truth for changes applied to PrivX.

Implementation:
- `internal/apply/`
- `internal/privx/`

## Diagnostics

All phases may produce diagnostics.

Diagnostics are used to report:

- Validation failures
- Unsupported field values
- Scope violations
- PrivX API errors
- Planning conflicts

Diagnostics are displayed through the output layer and may also affect whether processing continues.

Implementation:
- `internal/domain/diagnostic.go`
- `internal/output/`

## Output

User-visible output is handled through a common output abstraction.

Output verbosity controls the amount of information displayed during execution.

Typical output includes:

- Progress information
- Diagnostics
- Generated plans
- Debug dumps

Implementation:
- `internal/output/`