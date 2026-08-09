# 2026-07-02-tasks-workitem-get.md

**Status:** Active

## Context

Implements the MVP command `tfscli workitem get` end to end, following the module
structure, interfaces, and data flow defined in `architecture.md`. Tasks are ordered
bottom-up by dependency: leaf modules first, CLI wiring and live verification last.

Decisions recorded for this scope:

- **`Post` is out of scope.** The `APIClient` interface in `architecture.md` lists
  `Get` and `Post`; only `Get` is implemented here. `Post` is added when the first
  POST endpoint (batch get, WIQL) arrives.
- **`tfscli config init` is out of scope.** The `config` error example in
  `project-brief.md` references it; the error message is changed to
  `config file not found at ~/.tfscli/config.json` instead, and the brief example is
  updated in TASK-03. The `config init` command waits for its own tasks file.
- **HTML quirks are out of scope.** `htmlmd` uses library defaults only. TASK-08
  (first live TFS connection) is the trigger for `2026-05-20-tasks-html-quirks.md`.
- **A missing config file is not an error by itself.** The brief recommends
  `TFSCLI_PAT` via the environment on shared and CI machines, so environment
  variables and flags alone must be enough to run. `config file not found at
  <path>` is reported only when a required field is in fact missing and the file
  is absent — then it is the most useful thing to say.
- **`config.Load` does not validate `Project`.** It is required per command, not
  per program; `Config.RequireProject()` is called by the commands that need it
  (TASK-07). `Load` takes the config path and a struct of explicitly-set flag
  values, so the package stays free of a cobra dependency.

---

### TASK-01 `tfserr`
**Description:** Implement the `tfserr` package: typed error `Error{Category, Message, HTTPStatus, Cause}` with the six stable categories from the brief (`auth`, `not_found`, `forbidden`, `server`, `config`, `network`); `Print(err, w)` writing `Error [category]: message (HTTP status)` (status part omitted when there is no HTTP status); `ExitCode(err)` returning a non-zero code (v1: 1 for every category).
**Definition of done:** Unit tests cover the output format for every category, both with and without an HTTP status; `go test ./internal/tfserr` passes.
**Status:** Done

### TASK-02 `log`
**Description:** Implement the `log` package: `Logger` interface with `LogRequest(method, url string, status int, dur time.Duration)` and `Warn(msg string)`; a `noop` implementation (default) and a `stderr` implementation (selected later by `--verbose` / `TFSCLI_VERBOSE=1`).
**Definition of done:** Tests verify the stderr logger writes method, URL, status, and duration — and nothing else; the noop logger writes nothing; `go test ./internal/log` passes.
**Status:** Done

### TASK-03 `config`
**Description:** Implement the `config` package: load `~/.tfscli/config.json`, overlay environment variables (`TFSCLI_URL`, `TFSCLI_COLLECTION`, `TFSCLI_PAT`, `TFSCLI_PROJECT`, `TFSCLI_API_VERSION`), overlay explicitly-set flag values, validate required fields (`URL`, `Collection`, `PAT` always; `Project` must come from flag, env, or config). `Config` also holds `APIVersion` (default `7.2`), `InsecureSkipVerify`, `CABundle`. Malformed JSON and missing required fields fail with category `config`; a missing config file is reported as `config file not found at ~/.tfscli/config.json`. Update the `config` error example in `project-brief.md` to match (drop the `config init` reference).
**Definition of done:** Tests cover the precedence chain flag > env > file > built-in default, and each validation failure; the brief example is updated; `go test ./internal/config` passes.
**Status:** Done

### TASK-04 `apiclient`
**Description:** Implement the `apiclient` package wrapping `net/http`: build URLs from `Config.URL + Collection + path`; set `Authorization: Basic base64(":<PAT>")` on every request; configure TLS from `CABundle` (appended to the system root pool) and `InsecureSkipVerify`; log every request exactly once via a `RoundTripper` that is the single call site of `LogRequest`; classify non-2xx responses and transport failures into `*tfserr.Error` categories (401 → `auth`, 403 → `forbidden`, 404 → `not_found`, 5xx → `server`, connection errors → `network`). Implement `Get(ctx, path, query)` only.
**Definition of done:** Tests against `httptest.Server` verify the Authorization header, the classification of 401/403/404/5xx responses, and that an unreachable server yields category `network`; `go test ./internal/apiclient` passes.
**Status:** Pending

### TASK-05 `htmlmd`
**Description:** Implement the `htmlmd` package: a thin wrapper over `github.com/JohannesKaufmann/html-to-markdown/v2` exposing `Convert(html string) (string, error)`, library defaults only. Add the dependency to `go.mod`.
**Definition of done:** Smoke tests convert representative snippets (bold, list, link) to expected markdown; `go test ./internal/htmlmd` passes.
**Status:** Pending

### TASK-06 `workitem`
**Description:** Implement the `workitem` package: `WorkItem`, `Field`, `FieldKind` types per `architecture.md`; the allowlist of HTML fields (`System.Description`, `Microsoft.VSTS.TCM.ReproSteps`, `Microsoft.VSTS.TCM.SystemInfo`, `Microsoft.VSTS.Common.AcceptanceCriteria`); known identity and datetime fields; `Get(ctx, client, project, id, fields)` building the path `/{project}/_apis/wit/workitems/{id}` with `fields` and `api-version` query parameters, unmarshalling the response, and tagging each field with its `FieldKind`. Returns raw values — no markdown conversion here.
**Definition of done:** Tests with a fake `APIClient` run against a realistic TFS response JSON fixture in `testdata/` and verify field kinds and values; `go test ./internal/workitem` passes.
**Status:** Pending

### TASK-07 `cli-wiring`
**Description:** Implement the `cli` package and wire `cmd/tfscli/main.go`: cobra root command with persistent flags (`--verbose`, config overrides including `--api-version`); `workitem get` subcommand with `-p`/`--project`, `--fields`, and a positive-integer check on the id argument; markdown printer iterating `WorkItem.Fields` — HTML fields through `htmlmd.Convert`, identity and datetime fields through small dedicated formatters, plain fields as is — writing to stdout; on any error `tfserr.Print(err, os.Stderr)` and exit with `tfserr.ExitCode(err)`. `main` only wires modules together and delegates to `cli.Run()`.
**Definition of done:** `go build ./...` produces a working binary; an integration test drives the command against an `httptest.Server` and verifies stdout content, stderr error format, and exit behaviour for success, `not_found`, and `auth` cases; `go test ./...`, `go vet ./...`, and `gofmt -l .` are clean.
**Status:** Pending

### TASK-08 `live-smoke`
**Description:** Run the built binary against a real on-prem TFS / Azure DevOps Server instance: a successful `workitem get`, a run with `--fields`, a nonexistent id (expect `not_found`), and an invalid PAT (expect `auth`). This is the trigger event for `2026-05-20-tasks-html-quirks.md`.
**Definition of done:** A short report of the four runs (command, expected vs. actual outcome) is appended below this task's DoD in this file; any discrepancies are either fixed or recorded as new tasks / tech debt.
**Status:** Pending

### TASK-09 `docs`
**Description:** Write `README.md` (what tfscli is, installation, configuration file and environment variables, `workitem get` usage examples, error format) and add the CHANGELOG entry via the `changelog` skill.
**Definition of done:** `README.md` and `CHANGELOG.md` are updated and consistent with the implemented behaviour.
**Status:** Pending
