# PrivX Inventory Import — Architecture & Data Flow

> **Last updated:** May 2026 — reflects current codebase including composite field types, translators, validators, update modes, and OAuth authentication.

## 1. High-Level Overview

```mermaid
graph TB
    subgraph External["External Systems"]
        CMDB["CMDB / Asset<br/>Management"]
        PRIVX_API["PrivX PAM<br/>API Server"]
    end

    subgraph Tool["privx-import CLI Tool"]
        CLI["CLI Layer<br/>(cobra commands)"]
        WF["Workflow<br/>Orchestrator"]
    end

    CMDB -->|"CSV export files"| CLI
    CLI --> WF
    WF <-->|"REST API<br/>(token auth)"| PRIVX_API

    style CMDB fill:#e8f4fd,stroke:#2196F3
    style PRIVX_API fill:#fff3e0,stroke:#FF9800
    style CLI fill:#e8f5e9,stroke:#4CAF50
    style WF fill:#e8f5e9,stroke:#4CAF50
```

**Purpose:** `privx-import` is a CLI tool that synchronizes host inventory from CMDB exports (CSV files) into SSH.com's PrivX privileged access management system. It follows a plan-then-apply workflow similar to Terraform.

---

## 2. Complete Pipeline Architecture

```mermaid
flowchart LR
    subgraph INPUT["① Input"]
        CSV["CSV Files"]
        CFG["Config<br/>(host.toml,<br/>config.toml)"]
    end

    subgraph READ["② Read"]
        MR["MultiReader"]
        CR["CSVReader"]
    end

    subgraph TRANSFORM["③ Transform"]
        NORM["Normalize<br/>+ Translate<br/>+ Fold"]
        VAL["Validate<br/>+ Merge Rows"]
    end

    subgraph PLAN["④ Plan"]
        TS["Build<br/>TargetSpec"]
        HP["Build<br/>HostPlans<br/>(Overlay)"]
    end

    subgraph PRIVX["⑤ PrivX"]
        PREP["Prepare<br/>(fetch groups,<br/>roles)"]
        FETCH["Fetch Current<br/>Hosts"]
        APPLY["Apply<br/>Changes"]
    end

    CSV --> MR --> CR
    CR -->|"RawRecords"| NORM
    CFG -->|"FieldCatalog"| NORM
    PREP -->|"Translators"| NORM
    NORM -->|"NormalizedRecords<br/>(typed Fields)"| VAL
    VAL -->|"ValidatedResult"| TS
    TS -->|"TargetSpecs"| HP
    FETCH -->|"CurrentHosts"| HP
    HP -->|"HostPlans"| APPLY

    style INPUT fill:#e3f2fd,stroke:#1565C0
    style READ fill:#e8f5e9,stroke:#2E7D32
    style TRANSFORM fill:#fff8e1,stroke:#F9A825
    style PLAN fill:#fce4ec,stroke:#C62828
    style PRIVX fill:#f3e5f5,stroke:#6A1B9A
```

---

## 3. Package Dependency Graph

```mermaid
graph TD
    MAIN["cmd/privx-import<br/><i>main.go</i>"]
    CLI["internal/cli<br/><i>root, plan, apply, show</i>"]
    CLIOUT["internal/cli/output<br/><i>lipgloss formatting</i>"]
    CONFIG["internal/config<br/><i>TOML loader, field catalog,<br/>decode_json, folding</i>"]
    RUNTIME["internal/runtime<br/><i>Runtime state container</i>"]
    WORKFLOW["internal/workflow<br/><i>orchestrator</i>"]
    READER["internal/reader<br/><i>CSV reader, multi-reader</i>"]
    TRANSFORM["internal/transform<br/><i>normalize, validate,<br/>translate</i>"]
    TRANSLATORS["internal/translators<br/><i>LookupTranslator,<br/>AllowedValuesTranslator</i>"]
    VALIDATORS["internal/validators<br/><i>services, disabled,<br/>ssh_key</i>"]
    PLAN["internal/plan<br/><i>planner, targetspec</i>"]
    PRIVX["internal/privx<br/><i>API client, auth,<br/>hosts, access_groups, roles</i>"]
    DOMAIN["internal/domain<br/><i>diagnostic</i>"]
    HOST["internal/domain/host<br/><i>HostKey, match/merge</i>"]
    FIELD["internal/field<br/><i>string, bool, int, list,<br/>composite, list_composite</i>"]
    OUTPUT["internal/output<br/><i>Output interface</i>"]
    UTIL["internal/util<br/><i>helpers</i>"]

    MAIN --> CLI
    CLI --> CLIOUT
    CLI --> CONFIG
    CLI --> RUNTIME
    CLI --> WORKFLOW
    RUNTIME --> CONFIG
    RUNTIME --> PRIVX
    RUNTIME --> TRANSFORM
    WORKFLOW --> READER
    WORKFLOW --> TRANSFORM
    WORKFLOW --> PLAN
    WORKFLOW --> PRIVX
    WORKFLOW --> TRANSLATORS
    TRANSFORM --> CONFIG
    TRANSFORM --> DOMAIN
    TRANSFORM --> HOST
    TRANSFORM --> FIELD
    TRANSFORM --> VALIDATORS
    TRANSLATORS --> FIELD
    PLAN --> DOMAIN
    PLAN --> HOST
    PLAN --> FIELD
    PLAN --> TRANSFORM
    PRIVX --> HOST
    PRIVX --> FIELD
    PRIVX --> CONFIG
    PRIVX --> TRANSLATORS
    HOST --> FIELD
    READER --> DOMAIN
    CONFIG --> FIELD
    CONFIG --> UTIL
    FIELD --> UTIL

    style MAIN fill:#bbdefb,stroke:#1565C0
    style CLI fill:#c8e6c9,stroke:#2E7D32
    style WORKFLOW fill:#fff9c4,stroke:#F9A825
    style DOMAIN fill:#ffccbc,stroke:#BF360C
    style HOST fill:#ffccbc,stroke:#BF360C
    style FIELD fill:#ffccbc,stroke:#BF360C
    style PRIVX fill:#e1bee7,stroke:#6A1B9A
    style TRANSLATORS fill:#e1bee7,stroke:#6A1B9A
```

---

## 4. Detailed Data Flow (Sequence)

```mermaid
sequenceDiagram
    participant User
    participant CLI as CLI (cobra)
    participant Config as Config Loader
    participant WF as Workflow
    participant Reader as Reader (CSV)
    participant Norm as Normalize
    participant Val as Validate
    participant TS as TargetSpec Builder
    participant PrivX as PrivX Client
    participant Planner as Planner
    participant Output as Output Printer

    User->>CLI: privx-import plan hosts.csv
    CLI->>Config: LoadConfig(configDir)
    Config-->>CLI: FieldCatalog + HostFilter

    CLI->>CLI: LoadConfigFromEnv() → PRIVX_URL, PRIVX_TOKEN
    CLI->>PrivX: New(ctx, cfg) → token exchange
    PrivX-->>CLI: Authenticated Client

    CLI->>WF: PrepareHostPlans(ctx, client, inputs, catalog)

    WF->>Reader: BuildInputs(paths) + ReadAll(ctx, inputs)
    Reader-->>WF: ReadResult (RawRecords[])

    WF->>Norm: NormalizeReadResult(readResult)
    Norm-->>WF: ManyNormalizedResults

    WF->>Val: Validate(normalized, catalog)
    Val-->>WF: ManyValidatedResults

    WF->>TS: BuildTargetSpec(validated)
    TS-->>WF: map[string]TargetSpec

    WF->>PrivX: ListHosts(ctx, filter, catalog)
    PrivX-->>WF: map[string]TargetSpec (current)

    WF->>Planner: BuildHostPlans(cfg, current, target)
    Planner-->>WF: map[string]HostPlan

    WF-->>CLI: PreparedPlan
    CLI->>Output: PrintPreparedPlan(plan)
    Output-->>User: Display changes (create/update/delete)
```

---

## 5. Apply Flow (with Confirmation)

```mermaid
sequenceDiagram
    participant User
    participant CLI as CLI (apply)
    participant WF as Workflow
    participant PrivX as PrivX API

    User->>CLI: privx-import apply hosts.csv
    CLI->>WF: PrepareHostPlans(...)
    WF-->>CLI: PreparedPlan

    CLI->>CLI: PrintPreparedPlan()
    CLI->>User: "Do you want to apply? [y/N]"
    User->>CLI: "y"

    CLI->>WF: ApplyPlans(ctx, client, prepared)

    loop For each HostPlan
        alt Action = Create
            WF->>PrivX: POST /hosts (fields)
            PrivX-->>WF: host ID
        else Action = Update
            WF->>PrivX: PUT /hosts/{id} (fields)
            PrivX-->>WF: OK
        else Action = Delete
            WF->>WF: Skip (not implemented)
        end
    end

    WF-->>CLI: Diagnostics[]
    CLI-->>User: Results summary
```

---

## 6. Data Transformation Pipeline (Record Level)

```mermaid
flowchart TD
    subgraph CSV_ROW["CSV Row (raw)"]
        RAW["hostname, IP Address, tags<br/>web-01, 10.0.0.10, 'prod,web'"]
    end

    subgraph RAW_RECORD["RawRecord"]
        RR["{line: 2, fields:<br/>{hostname: 'web-01',<br/> IP Address: '10.0.0.10',<br/> tags: 'prod,web'}}"]
    end

    subgraph NORM_RECORD["NormalizedRecord"]
        NR["{line: 2, fields:<br/>{host_name: 'web-01',<br/> ip_address: '10.0.0.10',<br/> tags: 'prod,web'}}"]
    end

    subgraph VAL_RECORD["ValidatedRecord"]
        VR["{line: 2,<br/> keys: {hostname: 'web-01',<br/>         ip: '10.0.0.10'},<br/> fields: {<br/>   host_name: StringField('web-01'),<br/>   ip_address: StringField('10.0.0.10'),<br/>   tags: ListField(['prod','web'])<br/> }}"]
    end

    subgraph TARGET["TargetSpec"]
        TS["Keys: {hostname: 'web-01', ip: '10.0.0.10'}<br/>Key: 'hostname:web-01'<br/>Fields: {host_name, ip_address, tags, ...}"]
    end

    subgraph HOST_PLAN["HostPlan"]
        HP["Action: CREATE<br/>Keys: {hostname: 'web-01'}<br/>Fields: all merged fields"]
    end

    CSV_ROW -->|"CSVReader.Read()"| RAW_RECORD
    RAW_RECORD -->|"normalizeFieldName()<br/>normalizeFieldValue()"| NORM_RECORD
    NORM_RECORD -->|"Validate()<br/>field.Parse()"| VAL_RECORD
    VAL_RECORD -->|"BuildTargetSpec()<br/>merge rows"| TARGET
    TARGET -->|"BuildHostPlan()<br/>compare with current"| HOST_PLAN

    style CSV_ROW fill:#e3f2fd,stroke:#1565C0
    style RAW_RECORD fill:#e8f5e9,stroke:#2E7D32
    style NORM_RECORD fill:#fff8e1,stroke:#F9A825
    style VAL_RECORD fill:#fce4ec,stroke:#C62828
    style TARGET fill:#f3e5f5,stroke:#6A1B9A
    style HOST_PLAN fill:#e0f2f1,stroke:#00695C
```

---

## 7. Field Type System

```mermaid
classDiagram
    class Field {
        <<interface>>
        +Name() string
        +Kind() FieldKind
        +Value() any
        +Equals(other Field) bool
        +EqualsSubset(other Field) bool
        +IsEmpty() bool
        +Text() string
        +String() string
        +Encode() any
        +Set(value any) Field
        +Merge(value any) Field
        +Parse(value string) Field
        +Overlay(other Field, fn) Field, []string
        +Walk(fn WalkFunc) error
        +Transform(fn TransformFunc) Field
    }

    class StringField {
        -FieldName string
        -FieldValue string
        +Merge(): error if conflict
    }

    class BoolField {
        -FieldName string
        -FieldValue *bool
        +Parse(): true/1/yes/y/on → true
        +Merge(): error if conflict
    }

    class IntField {
        -FieldName string
        -FieldValue int
        +Parse(): strconv.Atoi
        +Merge(): error if conflict
    }

    class ListField {
        -path string
        -list []string
        +Parse(): split by comma
        +Merge(): append + deduplicate
    }

    class CompositeField {
        -path string
        -fields Fields
        +Merge(): recursive child merge
        +AddChild(Field)
    }

    class ListCompositeField {
        -path string
        -items []Fields
        +Merge(): append items
        +AddItem(Fields)
    }

    Field <|.. StringField
    Field <|.. BoolField
    Field <|.. IntField
    Field <|.. ListField
    Field <|.. CompositeField
    Field <|.. ListCompositeField

    class FieldCatalog {
        +Fields map[string]FieldDef
        +InputKeys map[string]string
        +ResolveInputKey(input) string
        +FieldDef(path) FieldDef
        +DecodeFields(raw) Fields
        +RestructureFields(fields)
        +ManagedFieldNames() map
    }

    class FieldDef {
        +Name string
        +Kind FieldKind
        +InputKeys []string
        +Children []FieldDef
        +Default string
        +Translator TranslatorID
        +Validators []ValidatorID
        +Management Management
    }

    FieldCatalog --> FieldDef : contains
    FieldDef --> Field : creates via NewField()
```

---

## 8. Host Identification & Matching

```mermaid
flowchart TD
    subgraph KEYS["Host Key Priority (strongest → weakest)"]
        ID["ID (PrivX UUID)"]
        EXT["ExternalID (CMDB ID)"]
        INST["InstanceID (Cloud ID)"]
        HOST["Hostname (DNS)"]
        IP["IPAddress"]
    end

    subgraph MATCH["Match Logic"]
        M["Any shared non-empty<br/>identifier = MATCH"]
    end

    subgraph MERGE["Merge Logic"]
        MG["Combine identifiers<br/>from both keys.<br/>Error if same field<br/>has different values."]
    end

    subgraph BEST["BestKey() Selection"]
        B["Returns canonical key string:<br/>id:xxx > external_id:xxx ><br/>instance_id:xxx > hostname:xxx ><br/>ip_address:xxx"]
    end

    KEYS --> MATCH
    MATCH --> MERGE
    MERGE --> BEST

    style KEYS fill:#e8f5e9,stroke:#2E7D32
    style MATCH fill:#fff8e1,stroke:#F9A825
    style MERGE fill:#fce4ec,stroke:#C62828
    style BEST fill:#e3f2fd,stroke:#1565C0
```

---

## 9. Multi-Row Merge Example

```mermaid
flowchart LR
    subgraph INPUT["Input CSV Rows"]
        R1["Row 1: web-01, ssh, port=22,<br/>tags='prod,web'"]
        R2["Row 2: web-01, http, port=80,<br/>tags='frontend'"]
    end

    subgraph MERGE["Merge by HostKey"]
        MG["Same hostname 'web-01'<br/>→ merge into single TargetSpec"]
    end

    subgraph RESULT["Merged TargetSpec"]
        TS["host_name: 'web-01'<br/>service_name: ERROR (conflict!)<br/>service_port: ERROR (conflict!)<br/>tags: ['prod','web','frontend']"]
    end

    R1 --> MG
    R2 --> MG
    MG --> TS

    style INPUT fill:#e3f2fd,stroke:#1565C0
    style MERGE fill:#fff8e1,stroke:#F9A825
    style RESULT fill:#fce4ec,stroke:#C62828
```

> **Note:** Scalar fields (string/bool) produce a conflict diagnostic when merged with different values. List fields merge by appending and deduplicating.

---

## 10. CLI Command Structure

```mermaid
graph TD
    ROOT["privx-import<br/>(root command)"]
    PLAN["plan &lt;files...&gt;<br/>Calculate changes"]
    APPLY["apply &lt;files...&gt;<br/>Apply changes"]
    SHOW["show<br/>Inspect data"]
    SHOW_ENV["environment<br/>(alias: env)"]
    SHOW_CFG["config<br/>(alias: cfg)"]
    SHOW_AG["accessgroups<br/>(alias: ag)"]
    SHOW_FIELDS["fields<br/>(alias: field)"]
    SHOW_ROLES["roles<br/>(alias: role)"]

    ROOT --> PLAN
    ROOT --> APPLY
    ROOT --> SHOW
    SHOW --> SHOW_ENV
    SHOW --> SHOW_CFG
    SHOW --> SHOW_AG
    SHOW --> SHOW_FIELDS
    SHOW --> SHOW_ROLES

    subgraph FLAGS_ROOT["Global Flags"]
        V["-v / -vv / -vvv<br/>verbosity"]
        CD["--config-dir<br/>config directory"]
        UM["--update-mode<br/>addonly|modify|sync"]
    end

    subgraph FLAGS_APPLY["Apply Flags"]
        AA["--auto-approve<br/>skip confirmation"]
        DM["--delete-mode<br/>disable|delete"]
    end

    ROOT --- FLAGS_ROOT
    APPLY --- FLAGS_APPLY

    style ROOT fill:#bbdefb,stroke:#1565C0
    style PLAN fill:#c8e6c9,stroke:#2E7D32
    style APPLY fill:#ffcdd2,stroke:#C62828
    style SHOW fill:#fff9c4,stroke:#F9A825
```

---

## 11. Error & Diagnostic System

```mermaid
flowchart TD
    subgraph FATAL["Fatal Errors (stop execution)"]
        F1["File not found"]
        F2["Auth failure"]
        F3["Config parse error"]
    end

    subgraph DIAG["Collected Diagnostics (continue)"]
        subgraph WARN["⚠️ Warnings"]
            W1["csv_row_short/long"]
            W2["normalize_duplicate_field"]
        end
        subgraph ERR["❌ Errors"]
            E1["validate_unknown_field"]
            E2["validate_missing_identifier"]
            E3["validate_conflicting_scalar"]
            E4["host_key_conflict"]
            E5["apply_host_failed"]
        end
    end

    subgraph FLOW["Processing Continues"]
        NEXT["Next record / Next host"]
    end

    FATAL -->|"return err"| STOP["❌ Program exits"]
    DIAG -->|"append to []Diagnostic"| FLOW

    style FATAL fill:#ffcdd2,stroke:#C62828
    style WARN fill:#fff9c4,stroke:#F9A825
    style ERR fill:#ffccbc,stroke:#BF360C
    style FLOW fill:#c8e6c9,stroke:#2E7D32
```

---

## 12. PrivX API Integration

```mermaid
sequenceDiagram
    participant Tool as privx-import
    participant Auth as /auth/api/v1
    participant HostStore as /host-store/api/v1

    Note over Tool,Auth: Authentication
    Tool->>Auth: POST /token/login<br/>{token: JWT}
    Auth-->>Tool: {access_token: "..."}

    Note over Tool,HostStore: Fetch Current State
    Tool->>HostStore: POST /hosts/search<br/>{offset: 0, limit: 250}
    HostStore-->>Tool: {count: N, items: [...]}
    Tool->>HostStore: POST /hosts/search<br/>{offset: 250, limit: 250}
    HostStore-->>Tool: {count: N, items: [...]}

    Note over Tool,HostStore: Apply Changes
    Tool->>HostStore: POST /hosts<br/>{common_name, addresses, ...}
    HostStore-->>Tool: {id: "new-uuid"}

    Tool->>HostStore: PUT /hosts/{id}<br/>{common_name, addresses, ...}
    HostStore-->>Tool: 200 OK
```

---

## 13. Configuration Files

```mermaid
graph LR
    subgraph CONFIG_DIR["Config Directory"]
        CT["config.toml<br/>[host_filter]<br/>tags = [...]"]
        HT["host.toml<br/>[[field]]<br/>name, kind, input_keys"]
    end

    subgraph ENV["Environment Variables"]
        URL["PRIVX_URL"]
        TOKEN["PRIVX_TOKEN"]
    end

    CT -->|"HostFilterConfig"| RUNTIME["Runtime"]
    HT -->|"FieldCatalog"| RUNTIME
    ENV -->|"PrivX Config"| RUNTIME

    style CONFIG_DIR fill:#e8f5e9,stroke:#2E7D32
    style ENV fill:#fff8e1,stroke:#F9A825
    style RUNTIME fill:#e3f2fd,stroke:#1565C0
```

---

## Summary

| Phase | Package | Input | Output |
|-------|---------|-------|--------|
| **1. Init** | `cli`, `config`, `runtime` | CLI args, TOML files, env vars | Runtime (config, catalog, client) |
| **2. Prepare** | `workflow`, `privx` | PrivX API | Translators (access groups, roles, services) |
| **3. Read** | `reader` | CSV file paths | `ReadResult` (raw records) |
| **4. Normalize** | `transform` | Raw records + catalog + translators | `NormalizedResults` (typed Fields, translated, folded) |
| **5. Validate** | `transform`, `validators` | Normalized records + config | `ValidatedResult` (merged by host key, validated) |
| **6. Target** | `plan` | Validated records | `map[string]TargetSpec` (desired state) |
| **7. Current** | `privx` | PrivX API | `map[string]TargetSpec` (actual state) |
| **8. Plan** | `plan` | Target + Current + scope + update mode | `map[string]HostPlan` (create/update/delete) |
| **9. Apply** | `workflow`, `privx` | HostPlans + delete mode | API calls + diagnostics |

