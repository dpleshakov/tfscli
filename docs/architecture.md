# Architecture — tfscli

## Tech Stack

### Language: Go

**Rationale:**
- Compiles to a single static binary with zero runtime dependencies. Users download one file and it works.
- Cross-compilation built into the toolchain (`GOOS=windows GOARCH=amd64 go build`). Primary target is Windows, but Linux/macOS binaries are free.
- Fast cold start (~10ms). Important for a tool called repeatedly by an AI agent.
- `net/http` in standard library is sufficient for REST API calls — no need for third-party HTTP clients.
- `encoding/json` in standard library handles JSON parsing and serialization.

**Considered alternatives:**

| Alternative | Why rejected |
|---|---|
| C# | Initial candidate due to SSPI/`UseDefaultCredentials` for Windows auth. After choosing PAT over SSPI, the main advantage disappeared. Requires .NET runtime (unless using Native AOT, which increases build complexity). Larger binary size (~15-30 MB self-contained vs ~5-10 MB Go). |
| PowerShell | Good for prototyping. Rejected for distribution: Execution Policy blocks scripts on corporate machines, slow startup (~300-500ms), encoding/output quirks when called by an agent, harder to distribute as a single unit. |
| Rust | Comparable binary properties to Go. Rejected due to steeper learning curve and slower development speed for a project of this scope. No meaningful runtime advantage for an I/O-bound CLI. |

**Known risks:**
- Go module ecosystem is less mature than npm/NuGet for some corporate tooling. Mitigated by minimal dependency count (2 external packages total).

---

### CLI Framework: cobra (`github.com/spf13/cobra`)

**Rationale:**
- De-facto standard for Go CLI tools. Used by kubectl, gh (GitHub CLI), docker CLI, hugo.
- Native support for nested subcommands — fits the `tfscli <resource> <action>` pattern.
- Auto-generated help and usage output. AI agents and humans can call `tfscli --help`, `tfscli workitem --help`, `tfscli workitem get --help`.
- Pairs with `viper` for config file / env / flag merging, but viper is not required — cobra works standalone.

**Considered alternatives:**

| Alternative | Why rejected |
|---|---|
| `urfave/cli` | Capable, but less idiomatic for deep subcommand trees. Cobra's command registration model (one file per command) maps directly to the "one file per API domain" architecture goal. |
| No framework (bare `flag` package) | Feasible for MVP, but becomes painful when adding subcommands, per-command flags, and help generation. Not worth the initial savings. |

---

### HTML-to-Markdown: `github.com/JohannesKaufmann/html-to-markdown/v2`

**Rationale:**
- Mature, well-tested library (golden file tests, active maintenance, v2 released).
- Handles bold, italic, lists, tables, code blocks, links — all of which appear in TFS work item rich text fields.
- Extensible via plugins and custom rules. If TFS generates non-standard HTML in rich text fields, custom rules can handle edge cases without modifying core conversion logic.
- MIT licensed.

**Considered alternatives:**

| Alternative | Why rejected |
|---|---|
| `github.com/mrjoshuak/go-markdownify` | Port of Python `markdownify`. Less mature, no plugin system, zero downstream importers. |
| Manual regex stripping | Fragile, doesn't handle nested HTML structures, would need constant maintenance. |
| Returning raw HTML | Defeats the purpose — wastes tokens, AI agents parse HTML worse than markdown. |

---

### HTTP Client: Go standard library (`net/http`)

**Rationale:**
- All tfscli needs is `GET` and `POST` with JSON body, `Authorization` header, and TLS. Standard library covers this completely.
- No external dependency for core functionality.

**Considered alternatives:**

| Alternative | Why rejected |
|---|---|
| `github.com/go-resty/resty` | Convenient fluent API, but adds a dependency for features tfscli doesn't need (retries, middleware, file upload). |
| `github.com/hashicorp/go-retryablehttp` | Retry logic is useful, but for v1 simplicity — if the request fails, report the error. Retry can be added later without changing the interface. |

---

### Configuration: `encoding/json` (stdlib)

**Rationale:**
- Config file is `~/.tfscli/config.json` — a flat JSON object with 5 fields. Standard library `encoding/json` reads and writes it in a few lines.
- Config merging (file → env → flags) is handled by cobra's flag binding + a small custom resolver. No need for viper.

**Considered alternatives:**

| Alternative | Why rejected |
|---|---|
| `github.com/spf13/viper` | Full-featured config library (YAML, TOML, env, remote config). Massive dependency tree for a 5-field config. Adds complexity without proportional value. |
| TOML / YAML config format | JSON is sufficient for a flat config. Every language and tool reads JSON natively. No benefit from TOML/YAML for this use case. |

---

### Summary

| Component | Choice | External dependency? |
|---|---|---|
| Language | Go | — |
| CLI framework | cobra | Yes (`github.com/spf13/cobra`) |
| HTML → Markdown | html-to-markdown v2 | Yes (`github.com/JohannesKaufmann/html-to-markdown/v2`) |
| HTTP client | `net/http` | No (stdlib) |
| JSON parsing | `encoding/json` | No (stdlib) |
| Config format | JSON via `encoding/json` | No (stdlib) |

Total external dependencies: **2** (cobra, html-to-markdown). Minimal dependency footprint for a CLI tool.

---

## Architecture

This section describes the conceptual module structure, their responsibilities, the interfaces between them, and the data flow for the MVP command. Directory layout and file naming are decided separately in the "Project Structure" section.

### Overview diagram

```
        ┌──────────────────┐
        │  cobra CLI       │   flags: -p, --fields, --verbose, --api-version, …
        │  (cli)           │
        └────────┬─────────┘
                 │ effective Config
                 ▼
        ┌──────────────────┐
        │  config          │   merge: file < env < flag, validate required
        └────────┬─────────┘
                 │ Config{URL, Collection, PAT, Project, APIVersion,
                 │        InsecureSkipVerify, CABundle}
                 ▼
        ┌──────────────────┐       ┌─────────────────────┐
        │  apiclient       │◄─────►│  TFS REST API       │
        │  net/http + PAT  │  TLS  │  (7.2 default)      │
        │  + TLS + logger  │       └─────────────────────┘
        └────────┬─────────┘
                 │ raw JSON / *tfserr.Error
                 ▼
        ┌──────────────────┐
        │  workitem        │   json.Unmarshal + tag FieldKind
        │  (domain)        │   (HTML field allowlist lives here)
        └────────┬─────────┘
                 │ *WorkItem
                 ▼
        ┌──────────────────┐      ┌────────────────────┐
        │  cli printer     │─────►│  htmlmd            │
        │  (markdown)      │ HTML │  Converter         │
        └────────┬─────────┘      └────────────────────┘
                 │ markdown
                 ▼
              stdout

  At any step: *tfserr.Error ─► tfserr.Print(stderr) ─► exit ≠0
  --verbose: apiclient round-trip ─► log.LogRequest ─► stderr (no bodies, no PAT)
```

### Modules and responsibilities

- **cli** — cobra commands; persistent flags (`--verbose`, config overrides); orchestrates the chain config → apiclient → domain → printer → exit. Owns the markdown printer functions (one per resource) plus small per-kind scalar formatters for identity and datetime values used by those printers. No `Renderer` interface in v1 — printers are plain functions. When `--json` is added later, the interface and a second renderer can be introduced here without touching the domain layer.
- **config** — loads `~/.tfscli/config.json`, applies environment overrides (`TFSCLI_*`), applies cobra flag overrides; validates that all required fields are set for the command being run. Holds `URL`, `Collection`, `PAT`, `Project`, `APIVersion`, `InsecureSkipVerify`, `CABundle`. Never logs PAT.
- **apiclient** — wraps `net/http`. Builds URLs from `Config.URL + Collection + path`. Sets `Authorization: Basic base64(":<PAT>")` on every request. Configures TLS using `CABundle` (appended to system root pool) and `InsecureSkipVerify`. Hooks the logger via a `RoundTripper` — that transport is the single call site of `LogRequest`, so every outgoing request is logged exactly once and no other module logs HTTP traffic. Classifies HTTP outcomes into the stable error categories defined in the brief (`auth`, `not_found`, `forbidden`, `server`, `config`, `network`).
- **workitem** — domain logic for Work Items (initially Get). Owns the `WorkItem`/`Field` types and the allowlist of fields known to contain HTML (`System.Description`, `Microsoft.VSTS.TCM.ReproSteps`, `Microsoft.VSTS.TCM.SystemInfo`, `Microsoft.VSTS.Common.AcceptanceCriteria`, …). After unmarshalling, tags each field with its `FieldKind`. Returns raw values — does not perform markdown conversion.
- **htmlmd** — thin wrapper over `github.com/JohannesKaufmann/html-to-markdown/v2`. One method: `Convert(html string) (string, error)`. v1 uses library defaults only; the wrapper is the extension point where TFS-specific rules will be registered once real HTML samples have been collected (see `2026-05-20-tasks-html-quirks.md`).
- **tfserr** — typed error `Error{Category, Message, HTTPStatus, Cause}` with the stable category set from the brief. Provides `Print(err, w)` writing `Error [category]: message (HTTP status)` and `ExitCode(err)` mapping to a non-zero process exit code (v1: 1 for every category; the API leaves room for per-category exit codes later without breaking the contract).
- **log** — `Logger` interface with two implementations: `noop` (default) and `stderr` (selected by `--verbose` / `TFSCLI_VERBOSE=1`). Methods: `LogRequest(method, url, status, dur)` and `Warn(msg)`. The logger interface accepts only these four request fields — PAT and the `Authorization` header are never passed in, so they cannot leak through the logger by construction.

### Key dependency interfaces

These are the boundaries between modules. Final signatures may evolve during Step 4.

```go
type APIClient interface {
    Get(ctx context.Context, path string, query url.Values) ([]byte, error)
    Post(ctx context.Context, path string, body any) ([]byte, error)
}

type Logger interface {
    LogRequest(method, url string, status int, dur time.Duration)
    Warn(msg string)
}

type Converter interface {
    Convert(html string) (string, error)
}
```

Domain types for Work Items:

```go
type FieldKind string
const (
    FieldPlain    FieldKind = "plain"
    FieldHTML     FieldKind = "html"
    FieldIdentity FieldKind = "identity"   // {displayName, uniqueName}
    FieldDateTime FieldKind = "datetime"
)

type Field struct {
    Name  string
    Kind  FieldKind
    Value any           // string for Plain/HTML/DateTime, struct for Identity
}

type WorkItem struct {
    ID     int
    Rev    int
    Fields []Field
}
```

### Data flow: `tfscli workitem get -p MyProject 12345 --fields System.Title,System.Description`

1. cobra parses the command and flags.
2. `config.Load(flagSet)` reads the config file, overlays env vars, overlays bound flags; validates that `URL`, `Collection`, `PAT` are present and that `Project` is supplied either by `-p`, env, or config default.
3. cli builds a `Logger` (noop or stderr depending on `--verbose`) and an `apiclient.APIClient`. The TLS config is derived from `CABundle` and `InsecureSkipVerify`. If verification is disabled, the logger emits one `[warn] TLS verification disabled` line.
4. cli calls `workitem.Get(ctx, client, project, id, fields)`.
5. `workitem.Get` constructs the path `/{Collection}/{Project}/_apis/wit/workitems/{id}?fields=…&api-version={APIVersion}` and calls `client.Get`.
6. `apiclient.Get` builds the request, sets the `Authorization` header, and executes it through its `RoundTripper`, which calls `LogRequest` once with method, URL, status, and duration. On non-2xx the response is classified into a `*tfserr.Error`; otherwise the response body is returned.
7. `workitem.Get` unmarshals the JSON, walks `fields`, sets each `Field.Kind` using the HTML allowlist plus known identity/datetime field types, and returns `*WorkItem`.
8. The cli printer iterates `wi.Fields`. HTML fields go through `htmlmd.Convert`. Identity and datetime fields are formatted by small dedicated helpers. Plain fields print as is. Output is written to `stdout`.
9. On error at any step: `tfserr.Print(err, os.Stderr)`; `os.Exit(tfserr.ExitCode(err))`.

### Security checklist

**Trust boundary.** Untrusted input enters the program at three places: (a) HTTP responses from the TFS server, (b) the config file (notably the PAT it stores), (c) CLI args and environment variables. The domain layer (`workitem`) and printers treat parsed Go structures as already-validated; validation happens at the boundary.

**Validation.**
- CLI args: cobra type checking; positive-integer check on work-item id.
- Env vars and config file: required fields enforced in `config.validate()` after the merge. Malformed JSON in the config file fails fast at load with category `config`.
- TFS response: `encoding/json` validates shape into typed intermediate structs. HTML field values are passed only to `htmlmd` — never executed, never written to disk verbatim except as markdown in stdout, never used to build shell or filesystem paths.

**What goes into logs.**
- `--verbose` line: method, full URL, status code, duration. Nothing else.
- The `Authorization` header and PAT never reach the logger — the logger interface accepts only those four scalar fields.
- TLS-disabled warning printed once at startup when `InsecureSkipVerify=true`.
- Errors to stderr: standard `Error [category]: message (HTTP status)` form. The `message` may carry sanitised text from the TFS error JSON (its own `message` field), but never headers and never PAT.

**Secret propagation between modules.**
- PAT enters at `config.Load` and is held only in `Config`.
- Passed to `apiclient.New(cfg, logger)` and stored there. Used in exactly one place: setting `Authorization` on outgoing requests.
- `workitem`, `htmlmd`, `tfserr`, and the cli printer never receive the PAT.

**Defaults are safe.**
- TLS verification: on. `InsecureSkipVerify` must be set explicitly in the config file.
- Verbose: off. Stderr stays empty on success.
- `CABundle`: empty. Falls back to the system root certificate store.

### Notes for future evolution

- **New API domain** (repos, builds, wiql): add a new domain module and register a cobra subcommand. No edits to `apiclient`, `config`, or `tfserr`.
- **`--json` output**: introduce a `Renderer` interface in the cli layer; add a `JSONRenderer` that consumes `*WorkItem` directly. The markdown printer becomes the other implementation. Local refactor confined to the cli layer.
- **TFS-specific HTML quirks** (mentions, attachment links, Word paste leftovers): add rules in `htmlmd` plus golden tests on real samples. Tracked in `2026-05-20-tasks-html-quirks.md`.

---

## Project Structure

Go module path: `github.com/dpleshakov/tfscli`.

### Layout principle

Standard Go CLI layout: a single binary entry point under `cmd/`, all implementation under `internal/`. Package boundaries are exactly the module boundaries from the Architecture section — one Go package per module (cli, config, apiclient, workitem, htmlmd, tfserr, log) — so the dependency rules stated there are visible in import lists and enforced by the compiler rather than by convention. Everything lives under `internal/` because tfscli is a CLI tool, not a library: a zero public API surface keeps full freedom to refactor between releases.

### Top-level directories

| Directory | Purpose |
|---|---|
| `cmd/` | Binary entry points, one subdirectory per binary. v1 has a single binary, `tfscli`; its `main` only wires modules together and delegates to the cli package. |
| `internal/` | All implementation packages, one per architecture module. Tests sit next to the code (`_test.go`); fixtures live in per-package `testdata/` directories. |
| `docs/` | Design documents (`project-brief.md`, `architecture.md`), tasks files, tech-debt register. Not part of the shipped binary. |

Deliberately absent: `pkg/` (nothing is exported), `vendor/` (dependencies resolve through the module proxy), a separate `test/` tree (Go convention keeps tests beside the code they test).
