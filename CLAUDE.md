# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Process

The workflow is defined by four process skills kept in the repository under `.claude/skills/`: `tasks` (task breakdown and execution), `backlog` (accepted work not started yet), `changelog` (`CHANGELOG.md` entries and releases), and `tech-debt` (deferred compromises). They are authoritative — invoke the matching one before doing the work; this CLAUDE.md intentionally does not duplicate their content. `CONTRIBUTING.md` records the choices they leave to the project ("Process") and the purpose of each project document ("Project documents").

## Project state

The MVP is implemented and released: `tfscli auth login` stores the credential, `tfscli wit work-items get` reads one work item, `tfscli wit work-items list` and `tfscli wit work-items get-batch` read several work items by id, `tfscli wit wiql query-by-wiql` finds work item ids with a WIQL query, `tfscli licenses` prints the license texts embedded in the binary, and the module has one package per architecture module (see the "Project Structure" section of `docs/architecture.md`). Known defects are listed in the "Known limitations" section of `README.md`. Accepted work that has not started is listed in `docs/backlog.md`; work in progress goes through tasks files per the `tasks` skill.

Read these before the corresponding kind of change (the full list of project documents is in "Project documents" in `CONTRIBUTING.md`):

- `project-brief.md` — purpose, audience, scope, constraints, non-goals, and design principles of the product. **Read this before proposing any user-facing change.** It changes only when a product decision changes; feature behaviour belongs in `README.md`, not here.
- `architecture.md` — technology stack, architecture, and project structure, with rationale and rejected alternatives. **Read this before adding a dependency or swapping a library.**
- Process skills — the development workflow (see "Process" above).

## Non-negotiable product constraints

These come from `project-brief.md` and override casual feature requests. If a change would violate one of these, surface it before implementing.

- **PAT auth only.** No SSPI or NTLM. A PAT is issued for one collection, so the server URL, the collection, and the PAT are stored together and always come from one source: `$XDG_DATA_HOME/tfscli/auth.json` (default `~/.local/share/tfscli/auth.json`, mode `0600`, written only by the interactive `tfscli auth login`) or `TFSCLI_AUTH` holding the same JSON, which takes precedence. No flag, separate env var, or config key may supply the URL, the collection, or the token. No OS keychain unless an organisation requires one.
- **Read-only in v1.** Write operations are explicitly out of scope. They additionally depend on JSON output landing first, because markdown is lossy.
- **Write operations are gated on the command line.** When they are added, every operation that changes server state requires `--allow-changes`, and an irreversible one (its effect cannot be undone by another API operation) requires `--allow-irreversible` as well. Writing is a property of the command, never of a flag of a reading command. The gate flags have no short form, environment variable, or config key, and are checked before any request. See "Write operations" in Non-Goals of `project-brief.md`.
- **The command line stays matchable by agent permission rules.** Flags follow the last word of the command path; each command has one form (no aliases, no abbreviations, no prefix matching); local command names never coincide with REST API areas; no generic pass-through command. See "The command line can be governed by permission rules" in `project-brief.md`.
- **Markdown is the default output.** HTML fields (Description, ReproSteps, etc.) are converted via `github.com/JohannesKaufmann/html-to-markdown/v2`. JSON output is a planned `--json` flag, not present in v1.
- **Mirror TFS REST API structure.** Command hierarchy, parameter names, and behaviour follow the API. Any deviation needs explicit usability justification — don't invent new abstractions (no custom query syntax, no synthetic batch endpoints, no caching).
- **Stateless.** No daemon, no background process, no on-disk cache. Every call hits the server.
- **Stable error contract.** Error categories (`auth`, `not_found`, `forbidden`, `server`, `config`, `network`) are a contract: existing categories are never removed or renamed. New ones may be added. Format: `Error [category]: message (HTTP status)` to stderr, non-zero exit.
- **Config precedence:** CLI flag > env var (`TFSCLI_PROJECT`, `TFSCLI_API_VERSION`) > config file (optional) > built-in default. The credential is outside this chain (see above).
- **Target is on-prem TFS / Azure DevOps Server; the API version is negotiated.** Without a configured version, each request is preceded by `OPTIONS {url}/{collection}/_apis` and carries the version the server lists for its resource; when the negotiation fails, the request goes out without a version, as before. A configured version is sent as is. `auth login` never sends one. Cloud Azure DevOps Services may work incidentally but is not tested or supported.
- **Do not narrow server compatibility.** A change must not stop tfscli from working with a TFS / Azure DevOps Server version it worked with before, unless that is an explicitly discussed and recorded decision (see "Server compatibility is preserved" in `project-brief.md`).
- **Works out of the box; errors lead the way.** Assume the user has read nothing and configured nothing. Do not add a required setting where a working default exists, and do not guess where the answer is ambiguous — ask the user instead. Every error a first-time user can hit names the concrete next step, and says when only a person can take it (see "Works out of the box; errors lead the way" in `project-brief.md`).
- **Minimal dependencies.** Total external deps are intentionally three: `cobra`, `html-to-markdown/v2`, and `golang.org/x/term` (hidden PAT input for `auth login`). HTTP via `net/http`, JSON via `encoding/json`, config via `encoding/json`. go-licenses, which generates the license notices at release time, is a build tool and not a dependency. Don't pull in `viper`, `resty`, retry libraries, or YAML/TOML parsers without a strong reason and explicit discussion.

## Command shape

```
tfscli <area> <resource> <action> [flags] [arguments]
```

The area and the resource are the segments of the operation's page path in the REST API reference (`.../wit/work-items/get-work-item` is `wit work-items get`); the action is the operation name without the resource name. Flags follow the action: a flag placed before the end of the command path, such as `tfscli --verbose wit work-items get 1`, is refused. `tfscli auth login` and `tfscli licenses` are local commands and lie outside this scheme. The full rule is in "Follow TFS API structure" in `project-brief.md`.

Commands (from the brief):

```
tfscli wit work-items get -p MyProject 12345
tfscli wit work-items get -p MyProject 12345 --fields System.Title,System.State,System.Description
tfscli wit work-items list -p MyProject --ids 297,299,300
tfscli wit work-items get-batch -p MyProject --ids 297,299,300
tfscli wit wiql query-by-wiql -p MyProject --query "SELECT [System.Id] FROM WorkItems WHERE [System.State] = 'Active'"
```

`-p` (project) is required unless a default project is set in config or `TFSCLI_PROJECT`, except for `tfscli wit wiql query-by-wiql`, where the project is optional and the query then runs at collection level.

## Build / test commands

Everything the project verifies runs through the `Makefile` — `build`, `lint`, `test`, `licenses`, `check`, `clean` — and CI runs `make check` verbatim, so a green `make check` locally is the whole gate. A single test is `go test -run TestName ./path/to/pkg`.

Recipes never assume a Unix shell: file removal goes through `git clean`, and the coverage check, the license check, and the changelog handling live in Go programs under `tools/`, tagged `//go:build ignore` so that `go build ./...`, `go vet ./...`, and `go test ./...` do not see them.

A release is made by running the `Release` workflow from the Actions tab with a version number; it verifies, rewrites `CHANGELOG.md`, commits, tags, pushes, and builds a draft release with goreleaser, which is run by the workflow and not through the `Makefile`. Nothing about a release is done by hand except pressing Publish.

`CONTRIBUTING.md` has the rest — required tool versions, what each target does, the lint and coverage rules, and the release procedure. **Read it before touching the build, the workflows, or a release.**

## Git workflow

Changes reach `main` only through pull requests, and a merge requires a green CI. The only
direct pushes to `main` are made by the `Release` workflow. This is a convention: GitHub
does not enforce it while the repository is private (TD-07 in `docs/tech-debt.md`).

Before committing, check the current branch. If it is `main`, do not commit: propose a
branch name to the user and create the branch once they agree. Pushing and opening pull
requests are done by the user; commit locally and stop there.

`CONTRIBUTING.md` ("Commits and branches") has the full procedure.

## Tooling note

Primary search tool is `rg` (ripgrep) — already in PATH. Prefer it over `Select-String`/`findstr`. See the user's global instructions for full guidance.
