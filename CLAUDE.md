# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Process

The project follows a deliberate, strict workflow defined by process skills: `project-start` (brief, tech stack, architecture, repo skeleton), `tasks` (task breakdown and execution), `tech-debt` (deferred compromises), and `changelog` (CHANGELOG.md entries and releases). If these skills are available, they are authoritative — invoke the matching one before doing the work; this CLAUDE.md intentionally does not duplicate their content. If they are not available in the current environment, ask the user how to proceed instead of improvising the process from memory.

Note: tasks files created before the move to skills (e.g. `docs/2026-05-20-tasks-html-quirks.md`) follow an older format with task types and emoji statuses; keep their existing format when updating them.

## Project state

The MVP is implemented: `tfscli workitem get` works end to end, and the module has one package per architecture module (see the "Project Structure" section of `docs/architecture.md`). Release 0.0.1 is tagged; what has not happened yet is a run against a live TFS instance — see the "Status" and "Known limitations" sections of `README.md`. Further work goes through tasks files per the `tasks` skill.

Three documents drive everything:

- `project-brief.md` — product scope, constraints, non-goals, error/config/command contracts. **Read this before proposing any user-facing change.**
- `architecture.md` — tech stack with rationale and rejected alternatives. **Read this before adding a dependency or swapping a library.**
- Process skills — the development workflow (see "Process" above).

## Non-negotiable product constraints

These come from `project-brief.md` and override casual feature requests. If a change would violate one of these, surface it before implementing.

- **PAT auth only.** No SSPI or NTLM. The server URL and PAT are stored together and always come from one source: `$XDG_DATA_HOME/tfscli/auth.json` (default `~/.local/share/tfscli/auth.json`, mode `0600`, written only by the interactive `tfscli auth login`) or `TFSCLI_AUTH` holding the same JSON, which takes precedence. No flag, separate env var, or config key may supply the URL or the token. No OS keychain unless an organisation requires one.
- **Read-only in v1.** Write operations are explicitly out of scope. They additionally depend on JSON output landing first, because markdown is lossy.
- **Markdown is the default output.** HTML fields (Description, ReproSteps, etc.) are converted via `github.com/JohannesKaufmann/html-to-markdown/v2`. JSON output is a planned `--json` flag, not present in v1.
- **Mirror TFS REST API structure.** Command hierarchy, parameter names, and behaviour follow the API. Any deviation needs explicit usability justification — don't invent new abstractions (no custom query syntax, no synthetic batch endpoints, no caching).
- **Stateless.** No daemon, no background process, no on-disk cache. Every call hits the server.
- **Stable error contract.** Error categories (`auth`, `not_found`, `forbidden`, `server`, `config`, `network`) are a contract: existing categories are never removed or renamed. New ones may be added. Format: `Error [category]: message (HTTP status)` to stderr, non-zero exit.
- **Config precedence:** CLI flag > env var (`TFSCLI_COLLECTION`, `TFSCLI_PROJECT`, `TFSCLI_API_VERSION`) > config file (optional) > built-in default. The credential is outside this chain (see above).
- **Target is on-prem Azure DevOps Server (REST API 7.2 default, configurable).** Cloud Azure DevOps Services may work incidentally but is not tested or supported.
- **Minimal dependencies.** Total external deps are intentionally three: `cobra`, `html-to-markdown/v2`, and `golang.org/x/term` (hidden PAT input for `auth login`). HTTP via `net/http`, JSON via `encoding/json`, config via `encoding/json`. Don't pull in `viper`, `resty`, retry libraries, or YAML/TOML parsers without a strong reason and explicit discussion.

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

Everything the project verifies runs through the `Makefile` — `build`, `lint`, `test`, `check`, `release-notes`, `release`, `release-publish`, `clean` — and CI runs `make check` verbatim, so a green `make check` locally is the whole gate. A single test is `go test -run TestName ./path/to/pkg`.

Recipes never assume a Unix shell: file removal, the coverage check, and the changelog handling live in Go programs under `tools/`, tagged `//go:build ignore` so that `go build ./...`, `go vet ./...`, and `go test ./...` do not see them.

A release is made by running the `Release` workflow from the Actions tab with a version number; it verifies, rewrites `CHANGELOG.md`, commits, tags, pushes, and builds a draft release. Nothing about a release is done by hand except pressing Publish.

`CONTRIBUTING.md` has the rest — required tool versions, what each target does, the lint and coverage rules, and the release procedure. **Read it before touching the build, the workflows, or a release.**

## Tooling note

Primary search tool is `rg` (ripgrep) — already in PATH. Prefer it over `Select-String`/`findstr`. See the user's global instructions for full guidance.
