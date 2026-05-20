# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Process: read the matching document first — mandatory

The project follows a deliberate, strict workflow. The authoritative process descriptions live in `docs/`. **Before doing the work, open and read the document that matches the current task.** Skipping this step breaks the process and corrupts the repository's working model — tasks files get the wrong shape, the changelog accumulates entries that don't belong, tech debt goes uncaptured, and review tasks lose their checklist. Do not rely on this CLAUDE.md as a substitute; it intentionally does not duplicate the process content.

Decide which file to read by the current task:

| If the current task is… | Read |
|---|---|
| Setting up a brand-new project: brief, tech stack, architecture, or initial repo skeleton | `docs/process-project-start-en.md` |
| Anything inside a feature: task breakdown, implementing a Regular/Smoke/Review/Docs task, archiving a tasks file | `docs/process-feature-en.md` |
| Recording a deliberate compromise, updating `tech-debt.md`, or working in a way that touches tech-debt items | `docs/process-maintenance-en.md` |
| Adding or editing an entry in `CHANGELOG.md`, or preparing a release | `docs/process-changelog-guide-en.md` |

If more than one row applies, read all of them. If none clearly applies, ask the user which process the task falls under before proceeding.

## Project state

This is a **pre-code** project. The repository currently contains only design and process documents — no `go.mod`, no source files, no build scripts. If you're being asked to implement something, the first step is almost certainly to scaffold the Go module per the structure defined in `architecture.md` (the "Project Structure" section, once it exists, or by following the architecture decisions already documented).

Three documents drive everything:

- `project-brief.md` — product scope, constraints, non-goals, error/config/command contracts. **Read this before proposing any user-facing change.**
- `architecture.md` — tech stack with rationale and rejected alternatives. **Read this before adding a dependency or swapping a library.**
- `docs/process-*.md` — the development workflow (see below).

## Non-negotiable product constraints

These come from `project-brief.md` and override casual feature requests. If a change would violate one of these, surface it before implementing.

- **PAT auth only.** No SSPI, NTLM, or interactive login. PAT lives in `~/.tfscli/config.json` (plaintext, conscious v1 trade-off) and can be overridden by `TFSCLI_PAT`.
- **Read-only in v1.** Write operations are explicitly out of scope. They additionally depend on JSON output landing first, because markdown is lossy.
- **Markdown is the default output.** HTML fields (Description, ReproSteps, etc.) are converted via `github.com/JohannesKaufmann/html-to-markdown/v2`. JSON output is a planned `--json` flag, not present in v1.
- **Mirror TFS REST API structure.** Command hierarchy, parameter names, and behaviour follow the API. Any deviation needs explicit usability justification — don't invent new abstractions (no custom query syntax, no synthetic batch endpoints, no caching).
- **Stateless.** No daemon, no background process, no on-disk cache. Every call hits the server.
- **Stable error contract.** Error categories (`auth`, `not_found`, `forbidden`, `server`, `config`, `network`) are a contract: existing categories are never removed or renamed. New ones may be added. Format: `Error [category]: message (HTTP status)` to stderr, non-zero exit.
- **Config precedence:** CLI flag > env var (`TFSCLI_URL`, `TFSCLI_COLLECTION`, `TFSCLI_PAT`, `TFSCLI_PROJECT`, `TFSCLI_API_VERSION`) > config file > built-in default.
- **Target is on-prem Azure DevOps Server (REST API 7.2 default, configurable).** Cloud Azure DevOps Services may work incidentally but is not tested or supported.
- **Minimal dependencies.** Total external deps are intentionally two: `cobra` and `html-to-markdown/v2`. HTTP via `net/http`, JSON via `encoding/json`, config via `encoding/json`. Don't pull in `viper`, `resty`, retry libraries, or YAML/TOML parsers without a strong reason and explicit discussion.

## Command shape

```
tfscli <resource> <action> [flags] [arguments]
```

MVP commands (from the brief):

```
tfscli workitem get -p MyProject 12345
tfscli workitem get -p MyProject 12345 --fields System.Title,System.State,System.Description
```

`-p` (project) is required unless a default project is set in config or `TFSCLI_PROJECT`.

## Build / test commands

No build or test commands exist yet — the Go module hasn't been initialized. Once it is, the conventional commands will be `go build ./...`, `go test ./...`, `go test -run TestName ./path/to/pkg` for a single test, and `GOOS=windows GOARCH=amd64 go build` for cross-compilation. Don't document these in this file until they actually work in the repo.

## Tooling note

Primary search tool is `rg` (ripgrep) — already in PATH. Prefer it over `Select-String`/`findstr`. See the user's global instructions for full guidance.
