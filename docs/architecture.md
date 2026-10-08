# Architecture — tfscli

This document records the technology stack, the architecture, and the project structure of tfscli, with their rationale and the alternatives rejected. It changes when the stack, the modules, their interactions, or the top-level structure change. The behaviour of individual features is documented in `README.md`, not here.

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
- Go module ecosystem is less mature than npm/NuGet for some corporate tooling. Mitigated by minimal dependency count (3 external packages total).

---

### CLI Framework: cobra (`github.com/spf13/cobra`)

**Rationale:**
- De-facto standard for Go CLI tools. Used by kubectl, gh (GitHub CLI), docker CLI, hugo.
- Native support for nested subcommands — fits the `tfscli <area> <resource> <action>` pattern.
- Auto-generated help and usage output. AI agents and humans can call `tfscli --help`, `tfscli wit --help`, `tfscli wit work-items get --help`.
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

### Hidden terminal input: `golang.org/x/term`

**Rationale:**
- `tfscli auth login` reads the personal access token from the terminal with echo turned off, so the token never appears on screen, in scrollback, or in a screen recording. Turning echo off requires per-OS terminal calls (termios on Unix, console modes on Windows), which the standard library does not expose.
- Maintained by the Go team under the `golang.org/x` umbrella, with the same review process as the standard library. The Go toolchain vendors `x/term` itself.
- Brings one indirect dependency, `golang.org/x/sys`, from the same source.
- Also provides `IsTerminal`, which `auth login` uses to refuse to run when stdin is not an interactive terminal.

**Considered alternatives:**

| Alternative | Why rejected |
|---|---|
| Echoed input | Leaves the token visible on screen and in terminal scrollback. |
| Reading the token from stdin only (`echo $PAT \| tfscli auth login`) | Requires the token to be kept in yet another file or variable to pipe it in, which defeats the purpose of a separate credential store. |
| Hand-written per-OS syscall code | Duplicates `x/term` with less testing; the syscall packages it would need are `x/sys` anyway. |

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
- Config file is `$XDG_CONFIG_HOME/tfscli/config.json` (default `~/.config/tfscli/config.json`, on every OS) — an optional flat JSON object with 5 fields. Standard library `encoding/json` reads and writes it in a few lines.
- The credential file `$XDG_DATA_HOME/tfscli/auth.json` (default `~/.local/share/tfscli/auth.json`, on every OS) is a JSON object with two fields, `url` and `pat`, handled the same way.
- Config merging (file → env → flags) is handled by cobra's flag binding + a small custom resolver. No need for viper.

**Considered alternatives:**

| Alternative | Why rejected |
|---|---|
| `github.com/spf13/viper` | Full-featured config library (YAML, TOML, env, remote config). Massive dependency tree for a 5-field config. Adds complexity without proportional value. |
| TOML / YAML config format | JSON is sufficient for a flat config. Every language and tool reads JSON natively. No benefit from TOML/YAML for this use case. |

---

### License notices: `github.com/google/go-licenses/v2` (build tool)

**Rationale:**
- The binary links the Go standard library and the third-party modules statically, and their licenses (MIT, BSD-3-Clause, Apache-2.0) require the license text to accompany every binary copy. The texts are embedded in the binary and printed by `tfscli licenses`, so they travel with the binary even when it is taken out of the archive. GitHub CLI does the same (`gh licenses`).
- go-licenses finds the license file of each dependency, including one kept at the root of a repository above the module, classifies it, and saves it per package path. It runs at build time only: it is never linked into the binary and does not enter `go.mod`, so it does not count against the dependency budget above.
- The dependencies differ by platform (`inconshreveable/mousetrap` is linked on Windows only), so the set is generated per target, by a build pre-hook in `.goreleaser.yaml` with the target's `GOOS` and `GOARCH`, and each binary embeds the set of its own platform. The hook runs without a shell. go-licenses is installed by a `before` hook for the platform goreleaser runs on; `go run` in the build hook would build it for the target platform instead.
- go-licenses skips the standard library, so its license is a committed copy, `internal/licenses/go.LICENSE`, which a test keeps equal to the toolchain's.
- tfscli's own license is embedded too, as `internal/licenses/tfscli.LICENSE`, which a test keeps equal to `LICENSE`. The project owes no notice to itself, but whoever passes the binary on owes one, and finds it in the binary. It is a committed copy rather than part of the generated set, so that every build carries it and it is not listed among the third-party modules; the generated set excludes tfscli with `--ignore`.
- `make check` runs `go-licenses check` with the allowed licenses for each release platform, through `tools/check-licenses.go`, so a dependency under another license fails the pull request that brings it, whichever platform links it. The program installs go-licenses for the platform it runs on before setting each target's `GOOS` and `GOARCH`, for the same reason as the `before` hook.
- The release workflow builds a snapshot before it pushes anything and checks that every platform's set was generated and that the linux/amd64 binary prints it, so that a hook that left a binary without its licenses stops the release before the tag.

**Considered alternatives:**

| Alternative | Why rejected |
|---|---|
| License files in the release archive | Satisfies the licenses only while the binary stays beside them; a binary copied out of the archive carries no notice. |
| One set for every platform | go-licenses cannot merge sets, so a union needs a merging program; a per-platform set needs none, and each binary carries exactly what it links. |
| A set committed to the repository | A committed copy has to be regenerated and checked for drift on every dependency change; generating it at release time cannot drift. |
| A hand-written collector over `go list -deps` and the module cache | Misses license files outside the module directory and cannot classify a license, which `go-licenses check` needs. |

---

### Summary

| Component | Choice | External dependency? |
|---|---|---|
| Language | Go | — |
| CLI framework | cobra | Yes (`github.com/spf13/cobra`) |
| HTML → Markdown | html-to-markdown v2 | Yes (`github.com/JohannesKaufmann/html-to-markdown/v2`) |
| Hidden terminal input | x/term | Yes (`golang.org/x/term`) |
| HTTP client | `net/http` | No (stdlib) |
| JSON parsing | `encoding/json` | No (stdlib) |
| Config format | JSON via `encoding/json` | No (stdlib) |

Total external dependencies: **3** (cobra, html-to-markdown, x/term). Minimal dependency footprint for a CLI tool. go-licenses, which generates the license notices, is a build tool and not among them.

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
        │  net/http + PAT  │  TLS  │  (version: config,  │
        │  + TLS + logger  │       │   else OPTIONS)     │
        │                  │       └─────────────────────┘
        └────────┬─────────┘
                 │ raw JSON / *tfserr.Error
                 ▼
        ┌──────────────────┐
        │  workitem, wiql  │   json.Unmarshal; workitem also tags FieldKind
        │  (domain)        │   (its HTML field allowlist lives there)
        └────────┬─────────┘
                 │ *WorkItem / *Batch / *wiql.Result
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

- **cli** — cobra commands, nested as area, resource, and action after the REST API reference (`wit` → `work-items` → `get`); persistent flags (`--verbose`, config overrides); orchestrates the chain config → apiclient → domain → printer → exit. Owns the markdown printer functions (one per resource and shape, such as a single work item and several) plus small per-kind scalar formatters for identity and datetime values used by those printers. No `Renderer` interface in v1 — printers are plain functions. When `--json` is added later, the interface and a second renderer can be introduced here without touching the domain layer. Enforces the command line shape that "The command line can be governed by permission rules" in `project-brief.md` requires. Cobra accepts flags at any position, so `run` checks the arguments before executing the tree: it locates the target command with `root.Find`, and a flag token placed before the last word of the command path fails with a `config` error that shows the corrected command. The check is explicit rather than a root `PersistentPreRunE`, because cobra runs only the nearest such hook and a child command defining its own would silently drop it. `--help`, `-h`, and `--version` before the end of the path, a flag the target command does not define, and the `--` terminator skip the check, leaving them to cobra's own handling. A test walks the whole command tree and fails on any command alias or on cobra's prefix matching, so that each command keeps exactly one form.
- **config** — resolves the credential (`URL`, `Collection`, and `PAT`; a PAT is issued for one collection) from `TFSCLI_AUTH` or `$XDG_DATA_HOME/tfscli/auth.json`, never from a mix of sources; loads the optional `$XDG_CONFIG_HOME/tfscli/config.json`, applies environment overrides (`TFSCLI_*`), applies cobra flag overrides; validates that all required fields are set for the command being run. Writes `auth.json` for `auth login` (mode `0600`, directory `0700`). Holds `URL`, `Collection`, `PAT`, `Project`, `APIVersion`, `InsecureSkipVerify`, `CABundle`. Never logs PAT.
- **apiclient** — wraps `net/http`. Builds URLs from `Config.URL + Collection + path` and adds `api-version`. A configured `APIVersion` is sent as it is. Otherwise the version is negotiated per request, as the official SDKs do: `OPTIONS {URL}/{Collection}/_apis` lists every resource location with its versions, the caller names its location by id, and the request carries the location's released version, or `{maxVersion}-preview.{resourceVersion}` when it has none — the version the server chooses for a request without one. The response is not cached. A 401 or a transport failure on `OPTIONS` fails the request; any other failure sends it without a version and notes why through `Logger.Info`, and a refusal of that request for the missing version gets the settings to use in its message. A call without a location, the login check, neither negotiates nor sends a version. Sets `Authorization: Basic base64(":<PAT>")` on every request. Configures TLS using `CABundle` (appended to system root pool) and `InsecureSkipVerify`. Hooks the logger via a `RoundTripper` — that transport is the single call site of `LogRequest`, so every outgoing request is logged exactly once and no other module logs HTTP traffic. Classifies HTTP outcomes into the stable error categories defined in the brief (`auth`, `not_found`, `forbidden`, `server`, `config`, `network`).
- **workitem** — domain logic for Work Items: Get Work Item, Work Items - List, and Get Work Items Batch. Owns the `WorkItem`/`Field` types, the `BatchRequest`/`Batch` types for reading several work items, and the allowlist of fields known to contain HTML (`System.Description`, `Microsoft.VSTS.TCM.ReproSteps`, `Microsoft.VSTS.TCM.SystemInfo`, `Microsoft.VSTS.Common.AcceptanceCriteria`, …). After unmarshalling, tags each field with its `FieldKind`. For several work items, keeps the server's order and reports the requested ids the server did not return. Returns raw values — does not perform markdown conversion.
- **wiql** — domain logic for Wiql: Query By Wiql. Owns the `Request` and `Result` types. Builds the path at collection level, under a project, or under a project and a team; passes the query text through unchanged and sends `$top` and `timePrecision` only when they are given. Reads the query type, `asOf`, the reference names of the columns, and either the work item ids of a flat query or the relations of a link query, keeping the server's order. Does not read the work items themselves: that is a separate call through `workitem`, made by the caller.
- **htmlmd** — thin wrapper over `github.com/JohannesKaufmann/html-to-markdown/v2`. One method: `Convert(html string) (string, error)`. v1 uses library defaults only; the wrapper is the extension point where TFS-specific rules will be registered once real HTML samples have been collected (see the `html-quirks` entry in `docs/backlog.md`).
- **tfserr** — typed error `Error{Category, Message, HTTPStatus, Cause}` with the stable category set from the brief. Provides `Print(err, w)` writing `Error [category]: message (HTTP status)` and `ExitCode(err)` mapping to a non-zero process exit code (v1: 1 for every category; the API leaves room for per-category exit codes later without breaking the contract).
- **log** — `Logger` interface with two implementations: `noop` (default) and `stderr` (selected by `--verbose` / `TFSCLI_VERBOSE=1`). Methods: `LogRequest(method, url, status, dur)`, `Warn(msg)` for a condition worth reporting, and `Info(msg)` for what tfscli did on its own, such as sending a request without a version after a failed negotiation. The logger interface accepts only these four request fields — PAT and the `Authorization` header are never passed in, so they cannot leak through the logger by construction.
- **licenses** — the license texts of the code linked into the binary, for `tfscli licenses`. Always embeds `tfscli.LICENSE`, a copy of the project's `LICENSE`, and `go.LICENSE`, the license of the Go standard library; on each release platform also embeds `embed/<os>-<arch>/third-party/`, which the release build fills with go-licenses and git ignores. A committed `PLACEHOLDER` beside it keeps `go:embed` compiling in every other build, whose output then says that the third-party texts are only in the release builds. Depends on nothing else in the module.

### Key dependency interfaces

These are the boundaries between modules. Final signatures may evolve during Step 4.

```go
// location is the id under which the server lists the resource of the
// request in its response to OPTIONS on _apis; empty for no negotiation.
type APIClient interface {
    Get(ctx context.Context, location, path string, query url.Values) ([]byte, error)
    Post(ctx context.Context, location, path string, query url.Values, body any) ([]byte, error)
}

type Logger interface {
    LogRequest(method, url string, status int, dur time.Duration)
    Warn(msg string)
    Info(msg string)
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

type BatchRequest struct {
    IDs         []int
    Fields      []string
    AsOf        string   // passed through; empty is not sent
    ErrorPolicy string   // passed through; empty is not sent
}

type Batch struct {
    WorkItems []*WorkItem   // in the order the server returned them
    Missing   []int         // requested ids the server did not return
}
```

Domain types for Wiql:

```go
type Request struct {
    Project       string   // empty: the query runs at collection level
    Team          string   // used only together with Project
    Query         string   // passed through unchanged
    Top           *int     // nil is not sent
    TimePrecision *bool    // nil is not sent
}

type Relation struct {
    Source    int
    HasSource bool          // false for a root of a link result
    Target    int
    Rel       string
}

type Result struct {
    QueryType string        // flat, tree, oneHop
    AsOf      string
    Columns   []string      // reference names
    Link      bool          // a link query, even when Relations is empty
    WorkItems []int         // flat query, in the server's order
    Relations []Relation    // link query, in the server's order
}
```

### Data flow: `tfscli wit work-items get -p MyProject 12345 --fields System.Title,System.Description`

1. cobra parses the command and flags.
2. `config.Load` takes `URL`, `Collection`, and `PAT` from `TFSCLI_AUTH` or `auth.json`, which must carry all three; reads the config file if present, overlays env vars, overlays bound flags. The command then checks that `Project` is supplied either by `-p`, env, or config default.
3. cli builds a `Logger` (noop or stderr depending on `--verbose`) and an `apiclient.APIClient`. The TLS config is derived from `CABundle` and `InsecureSkipVerify`. If verification is disabled, the logger emits one `[warn] TLS verification disabled` line.
4. cli calls `workitem.Get(ctx, client, project, id, fields)`.
5. `workitem.Get` constructs the path `{Project}/_apis/wit/workitems/{id}` with the `fields` query parameter and calls `client.Get` with the location id of Work Items, `72c7ddf8-2cdc-4f60-90cd-ab71c14a399b`.
6. Without a configured version, `apiclient.Get` first sends `OPTIONS {Collection}/_apis` and takes the version listed for that location; with one, it uses the configured version. It then builds the request with `api-version`, sets the `Authorization` header, and executes it through its `RoundTripper`, which calls `LogRequest` once with method, URL, status, and duration. On non-2xx the response is classified into a `*tfserr.Error`; otherwise the response body is returned.
7. `workitem.Get` unmarshals the JSON, walks `fields`, sets each `Field.Kind` using the HTML allowlist plus known identity/datetime field types, and returns `*WorkItem`.
8. The cli printer iterates `wi.Fields`. HTML fields go through `htmlmd.Convert`. Identity and datetime fields are formatted by small dedicated helpers. Plain fields print as is. Output is written to `stdout`.
9. On error at any step: `tfserr.Print(err, os.Stderr)`; `os.Exit(tfserr.ExitCode(err))`.

`wit work-items list` and `wit work-items get-batch` follow the same flow with a `BatchRequest`. `workitem.List` sends it as the query of `GET {Project}/_apis/wit/workitems`, `workitem.GetBatch` as the JSON body of `POST {Project}/_apis/wit/workitemsbatch` through `client.Post`; parameters that are not set are not sent. Both parse the `value` array of the response with the single-item parser, skip the `null` entries that `errorPolicy=omit` leaves, and return a `*Batch`. The cli printer writes each work item as `wit work-items get` does and then one heading per missing id.

### Data flow: `tfscli wit wiql query-by-wiql -p MyProject --team Web --query "<WIQL>"`

1. cobra parses the command and flags. A missing or blank `--query` and a query given as an argument are rejected locally with category `config`, before any request.
2. `config.Load` resolves the configuration as for `wit work-items get`. The project is optional for this command; `--team` without a project from any source is rejected locally with category `config`.
3. cli builds the `Logger` and the `apiclient.APIClient` as above and calls `wiql.QueryByWiql(ctx, client, req)`, setting `Top` and `TimePrecision` only for flags that were given.
4. `wiql.QueryByWiql` builds the path `{Project}/{Team}/_apis/wit/wiql`, leaving out the segments that are empty, and calls `client.Post` with the body `{"query": ...}` and the optional `$top` and `timePrecision` in the query string. A syntax error in the query comes back as HTTP 400 with the server's message, which `apiclient` reports in the `config` category.
5. `wiql.QueryByWiql` unmarshals the response into a `*wiql.Result`; a response of the wrong shape is reported in the `server` category.
6. The cli printer writes a heading with the query type and `asOf` (in UTC, keeping the server's sub-second precision so that it can be passed to `--as-of`), the columns, and either the ids or one line per relation, with `none` for an empty list. The work items themselves are not read; the caller passes the ids to `wit work-items list --ids`.

### Security checklist

**Trust boundary.** Untrusted input enters the program at four places: (a) HTTP responses from the TFS server, (b) the config file, (c) the credential in `auth.json` or `TFSCLI_AUTH`, (d) CLI args, environment variables, and the answers typed into `auth login`. The domain layer (`workitem`, `wiql`) and printers treat parsed Go structures as already-validated; validation happens at the boundary.

**Validation.**
- CLI args: cobra type checking; positive-integer check on every work-item id, from the argument of `wit work-items get` and from `--ids`.
- Env vars, config file, and credential: required fields enforced in `config.Load` after the merge. Malformed JSON in the config file, `auth.json`, or `TFSCLI_AUTH` fails fast at load with category `config`; the credential URL must be `http` or `https` with a host.
- `auth login` answers: a server URL carrying a user name or password is refused with category `config`, without repeating the URL. tfscli authenticates only with the PAT, and what the user meant by the userinfo cannot be known, so it is neither stored nor silently dropped. A URL already stored with userinfo in `auth.json` or `TFSCLI_AUTH` is still accepted, so that a working credential keeps working.
- TFS response: `encoding/json` validates shape into typed intermediate structs. HTML field values are passed only to `htmlmd` — never executed, never written to disk verbatim except as markdown in stdout, never used to build shell or filesystem paths.

**What goes into logs.**
- `--verbose` line: method, full URL, status code, duration. Nothing else.
- The `Authorization` header and PAT never reach the logger — the logger interface accepts only those four scalar fields.
- TLS-disabled warning printed once at startup when `InsecureSkipVerify=true`.
- Errors to stderr: standard `Error [category]: message (HTTP status)` form. The `message` may carry sanitised text from the TFS error JSON (its own `message` field), but never headers and never PAT.

**Secret propagation between modules.**
- PAT enters at `config.Load` (from `auth.json` or `TFSCLI_AUTH`) or at the hidden prompt of `auth login`, and is held only in `Config` and `config.Auth`. It is never accepted as a flag or a command argument.
- The PAT is always paired with the URL stored next to it; no flag, environment variable, or config key can redirect it to another server.
- Passed to `apiclient.New(cfg, logger)` and stored there. Used in exactly one place: setting `Authorization` on outgoing requests.
- `workitem`, `wiql`, `htmlmd`, `tfserr`, and the cli printer never receive the PAT.

**Defaults are safe.**
- TLS verification: on. `InsecureSkipVerify` must be set explicitly in the config file.
- TLS version: 1.2 or later. `MinVersion` is set to TLS 1.2 when `CABundle` or `InsecureSkipVerify` is configured (`newTLSConfig` in `apiclient`), and the Go client default is the same otherwise. This restricts server compatibility, as "Server compatibility is preserved" in `project-brief.md` requires to be recorded: a server that offers only TLS 1.0 or 1.1, such as an older TFS on a Windows Server without TLS 1.2 enabled, cannot be reached. Accepting TLS 1.0 and 1.1 was rejected: both are deprecated and insecure, the PAT travels in every request, and no server limited to them is known among the users. The decision is revisited if such a server is reported.
- Verbose: off. Stderr stays empty on success.
- `CABundle`: empty. Falls back to the system root certificate store.

### Notes for future evolution

- **New API domain** (repos, builds): add a new domain module and register its resource command under the area command, adding the area command (e.g. `git`) if it does not exist yet. No edits to `apiclient`, `config`, or `tfserr`.
- **`--json` output**: introduce a `Renderer` interface in the cli layer; add a `JSONRenderer` that consumes `*WorkItem` directly. The markdown printer becomes the other implementation. Local refactor confined to the cli layer.
- **TFS-specific HTML quirks** (mentions, attachment links, Word paste leftovers): add rules in `htmlmd` plus golden tests on real samples. Recorded as the `html-quirks` entry in `docs/backlog.md`.
- **Write operations**: a command that changes server state declares it in its cobra `Annotations` — reversible or irreversible, per the criterion in `project-brief.md` — and registers the gate flags through one helper in the cli layer that defines `--allow-changes`, and `--allow-irreversible` for an irreversible command, and refuses a missing one with a `config` error before any request, the `OPTIONS` negotiation included. The command tree test is extended to require that every annotated command defines its gate flags and that no unannotated command defines them.

---

## Project Structure

Go module path: `github.com/dpleshakov/tfscli`.

### Layout principle

Standard Go CLI layout: a single binary entry point under `cmd/`, all implementation under `internal/`. Package boundaries are exactly the module boundaries from the Architecture section — one Go package per module (cli, config, apiclient, workitem, wiql, htmlmd, tfserr, log, licenses) — so the dependency rules stated there are visible in import lists and enforced by the compiler rather than by convention. Everything lives under `internal/` because tfscli is a CLI tool, not a library: a zero public API surface keeps full freedom to refactor between releases.

### Top-level directories

| Directory | Purpose |
|---|---|
| `cmd/` | Binary entry points, one subdirectory per binary. v1 has a single binary, `tfscli`; its `main` only wires modules together and delegates to the cli package. |
| `internal/` | All implementation packages, one per architecture module. Tests sit next to the code (`_test.go`); fixtures live in per-package `testdata/` directories. |
| `docs/` | Design documents (`project-brief.md`, `architecture.md`), tasks files, backlog, tech-debt register. Not part of the shipped binary. |
| `skills/` | The agent skill shipped beside the binary, `skills/tfscli/SKILL.md`, in the Agent Skills format. Copied into the archive by goreleaser and installed by copying the directory into an agent's skills path. |
| `.claude/` | Claude Code project settings and the process skills in `.claude/skills/` (tasks, backlog, changelog, tech-debt). Not shipped, and unrelated to `skills/`, which holds the skill for users of tfscli. |

Deliberately absent: `pkg/` (nothing is exported), `vendor/` (dependencies resolve through the module proxy), a separate `test/` tree (Go convention keeps tests beside the code they test).
