# Project Brief — tfscli

## Problem

AI coding agents (Claude Code, opencode, Cursor) have no lightweight way to access on-premises TFS / Azure DevOps Server data. Existing solutions are MCP servers that require Node.js runtime, long startup times, a persistent process, and explicit credentials (PAT or login/password). They don't fit the stateless CLI model that agents already use for OS interaction (`git`, `grep`, `find`).

Developers working with on-prem TFS need a tool that an AI agent can call the same way it calls any other command-line utility — instantly, with no prerequisites, and with predictable output.

## Proposed Solution

`tfscli` — a stateless CLI utility written in Go that provides read-only access to TFS / Azure DevOps Server REST API. Distributed as a single compiled binary (`.exe` for Windows, native binaries for other platforms).

Key properties:

- **Stateless.** No daemon, no background process. Called, returns output, exits.
- **PAT authentication.** No SSPI, no NTLM. PAT provides both simplicity and server-side access control (read-only scopes).
- **Markdown output.** HTML fields (Description, ReproSteps, etc.) are converted to markdown to save tokens and improve readability for AI agents. JSON output planned for future versions as an optional flag.
- **Mirrors TFS API structure.** Command hierarchy, parameters, and behavior follow TFS REST API conventions. Any deviation must be explicitly justified by usability. Users (including AI) should not need to learn a new mental model.
- **Extensible by design.** Architecture supports adding new API domains (WIQL, Repos, Builds, etc.) and new API versions without restructuring.

## Target Audience

Developers who use AI coding agents with on-premises TFS / Azure DevOps Server. The tool is also usable by humans, but AI agent ergonomics take priority in design decisions (output format, error messages, predictability).

## Key Success Metrics

An AI agent can call `tfscli`, read work item content, and use it meaningfully in its workflow — for example, to understand a task description, check acceptance criteria, or reference related work items.

## Constraints

- **Platform:** primary target is Windows (most on-prem TFS environments). Cross-compilation to Linux/macOS is a bonus enabled by Go, not a priority.
- **API coverage in MVP:** Work Items → Get Work Item (single ID). Get Work Items Batch (multiple IDs) if supported by the target API version. Fields selectable via `--fields` parameter; all fields returned by default.
- **Target API version:** Azure DevOps Server (latest, REST API 7.2). API version is configurable (default in config, overridable per call) to support older TFS installations.
- **Authentication:** PAT only. Stored in `~/.tfscli/config.json`, overridable via `TFSCLI_PAT` environment variable. No SSPI, NTLM, or interactive login. Note: PAT is stored in plaintext — this is a conscious trade-off for v1 simplicity. For shared or CI machines, `TFSCLI_PAT` via environment is recommended. OS keychain / DPAPI integration is a candidate for future versions.
- **Output:** Markdown by default. JSON as optional flag in future versions.
- **Distribution:** open source on GitHub, pre-built binaries in releases.

## Non-Goals

- **Write operations.** v1 is strictly read-only. Write operations are a future consideration, gated by PAT scopes. JSON output (`--json`) is a prerequisite for write support, since markdown conversion is lossy and not round-trippable to HTML.
- **MCP server mode.** tfscli is a CLI utility, not an MCP server. If MCP integration is needed, it can be wrapped externally.
- **Cloud Azure DevOps Services support.** Target is on-premises TFS / Azure DevOps Server. Cloud may work incidentally but is not tested or guaranteed.
- **Custom query syntax.** No invented query language. WIQL support (when added) will pass queries directly to the API.
- **GUI or interactive mode.** CLI only, designed for scripting and agent consumption.
- **Caching.** No local caching of API responses. Every call hits the server.

## Design Principles

- **Follow TFS API structure.** Command hierarchy mirrors API domains. Parameter names match API parameter names where possible. If TFS API has a batch endpoint — tfscli exposes batch. If it doesn't — tfscli doesn't invent one.
- **Predictable error output.** Errors go to stderr with a machine-readable category and human-readable message. Exit code is non-zero. Format: `Error [category]: message (HTTP status)`. Error categories are a stable contract: existing categories are never removed or renamed; new categories may be added in future versions.
- **Minimal configuration.** Config file stores server URL, collection, PAT, optional default project and API version. Environment variables override config values. No other configuration needed.
- **Clean output for AI consumption.** HTML in work item fields is converted to markdown. Metadata noise (URLs, internal IDs, revision details) is minimized in default output.

## Configuration

File: `~/.tfscli/config.json`

```json
{
  "url": "https://tfs.company.com:8080/tfs",
  "collection": "DefaultCollection",
  "pat": "...",
  "project": "MyProject",
  "apiVersion": "7.2"
}
```

All values overridable via environment variables (`TFSCLI_URL`, `TFSCLI_COLLECTION`, `TFSCLI_PAT`, `TFSCLI_PROJECT`, `TFSCLI_API_VERSION`) and command-line flags.

Priority: CLI flag > environment variable > config file > built-in default.

## Command Format

```
tfscli <resource> <action> [flags] [arguments]
```

MVP command:

```
tfscli workitem get -p MyProject 12345
tfscli workitem get -p MyProject 12345 --fields System.Title,System.State,System.Description
```

Project flag `-p` is required unless default project is set in config or `TFSCLI_PROJECT`.

## Error Format

Categories (`auth`, `not_found`, `forbidden`, `server`, `config`, `network`) are a stable contract — see Design Principles.

```
Error [auth]: PAT is invalid or expired (HTTP 401)
Error [not_found]: work item 99999 not found in project MyProject (HTTP 404)
Error [forbidden]: no access to project MyProject (HTTP 403)
Error [server]: TFS returned HTTP 500
Error [config]: config file not found at ~/.tfscli/config.json
Error [network]: cannot reach https://tfs.company.com:8080
```

## Key Dependencies

- `github.com/JohannesKaufmann/html-to-markdown/v2` — HTML to markdown conversion for work item fields
