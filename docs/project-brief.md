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
- **Authentication:** PAT only. No SSPI or NTLM. The server URL and the PAT are stored together and always come from the same source, so the token is never sent to any other server: either `$XDG_DATA_HOME/tfscli/auth.json` (default `~/.local/share/tfscli/auth.json`, on every OS), written only by `tfscli auth login`, or the `TFSCLI_AUTH` environment variable holding the same JSON, which takes precedence and serves CI. `auth.json` holds exactly one server, is written with mode `0600` (directory `0700`), and stores the token in plaintext. No flag, separate environment variable, or config key supplies the URL or the token. See "Authentication" below.
- **No OS keychain.** Against code running as the same user a keychain adds no protection; other users are excluded by file permissions, and stolen disks by disk encryption. A keychain is reconsidered only if an organisation requires secrets to be kept in the system store, and would then be a second storage backend behind the same `auth login`.
- **Output:** Markdown by default. JSON as optional flag in future versions.
- **Distribution:** open source on GitHub, pre-built binaries in releases.

## Non-Goals

- **Write operations.** v1 is strictly read-only. Write operations are a future consideration, gated by PAT scopes. JSON output (`--json`) is a prerequisite for write support, since markdown conversion is lossy and not round-trippable to HTML.
- **MCP server mode.** tfscli is a CLI utility, not an MCP server. If MCP integration is needed, it can be wrapped externally.
- **Cloud Azure DevOps Services support.** Target is on-premises TFS / Azure DevOps Server. Cloud may work incidentally but is not tested or guaranteed.
- **Custom query syntax.** No invented query language. WIQL support (when added) will pass queries directly to the API.
- **GUI or interactive mode.** CLI only, designed for scripting and agent consumption. The one exception is `tfscli auth login`, which prompts for the credential once per machine so that the token never appears as a command argument.
- **Caching.** No local caching of API responses. Every call hits the server.

## Design Principles

- **Follow TFS API structure.** Command hierarchy mirrors API domains. Parameter names match API parameter names where possible. If TFS API has a batch endpoint — tfscli exposes batch. If it doesn't — tfscli doesn't invent one.
- **Predictable error output.** Errors go to stderr with a machine-readable category and human-readable message. Exit code is non-zero. Format: `Error [category]: message (HTTP status)`. Error categories are a stable contract: existing categories are never removed or renamed; new categories may be added in future versions.
- **Minimal configuration.** `auth login` stores the server URL and PAT. An optional config file stores the collection, an optional default project, the API version, and TLS settings. Environment variables and flags override config values. No other configuration needed.
- **Clean output for AI consumption.** HTML in work item fields is converted to markdown. Metadata noise (URLs, internal IDs, revision details) is minimized in default output.

## Authentication

`tfscli auth login` takes no flags. It refuses to run when stdin is not a terminal, asks for the server URL, then for the PAT with echo turned off, verifies the pair with a request to `{url}/_apis/connectionData`, and writes `auth.json` only if the request succeeds. Verification failures are reported in the error categories below. The URL is normalised before it is stored: lower-case scheme and host, no trailing slash. TLS settings and the API version for the verification request come from the config file when it exists.

```json
{
  "url": "https://tfs.company.com:8080/tfs",
  "pat": "..."
}
```

`TFSCLI_AUTH` carries the same JSON for environments without an interactive terminal. When it is set, `auth.json` is not read.

## Configuration

File: `$XDG_CONFIG_HOME/tfscli/config.json`, defaulting to `~/.config/tfscli/config.json` when `XDG_CONFIG_HOME` is unset. The XDG Base Directory rules apply on every OS, Windows included. The file is optional.

```json
{
  "collection": "DefaultCollection",
  "project": "MyProject",
  "apiVersion": "7.2"
}
```

All values overridable via environment variables (`TFSCLI_COLLECTION`, `TFSCLI_PROJECT`, `TFSCLI_API_VERSION`) and command-line flags (`--collection`, `-p`, `--api-version`). The TLS settings `caBundle` and `insecureSkipVerify` are accepted in the config file only.

Priority: CLI flag > environment variable > config file > built-in default.

## Command Format

```
tfscli <resource> <action> [flags] [arguments]
```

MVP command:

```
tfscli workitem get -p MyProject 12345
tfscli workitem get -p MyProject 12345 --fields System.Title,System.State,System.Description
tfscli auth login
```

Project flag `-p` is required unless default project is set in config or `TFSCLI_PROJECT`.

## Error Format

Categories (`auth`, `not_found`, `forbidden`, `server`, `config`, `network`) are a stable contract — see Design Principles.

```
Error [auth]: PAT is invalid or expired (HTTP 401)
Error [not_found]: work item 99999 not found in project MyProject (HTTP 404)
Error [forbidden]: no access to project MyProject (HTTP 403)
Error [server]: TFS returned HTTP 500
Error [config]: not logged in: no credential at ~/.local/share/tfscli/auth.json (run "tfscli auth login", or set TFSCLI_AUTH)
Error [network]: cannot reach https://tfs.company.com:8080
```

## Key Dependencies

- `github.com/JohannesKaufmann/html-to-markdown/v2` — HTML to markdown conversion for work item fields
- `golang.org/x/term` — hidden PAT input and the terminal check in `tfscli auth login`
