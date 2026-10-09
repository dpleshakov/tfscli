# Project Brief — tfscli

This document records the purpose, audience, scope, constraints, non-goals, and design principles of tfscli. It changes only when one of these decisions changes. The behaviour of individual features is documented in `README.md`, not here.

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
- **Extensible by design.** Architecture supports adding new API domains (Repos, Builds, Work Item Comments, etc.) and new API versions without restructuring.

## Target Audience

Developers who use AI coding agents with on-premises TFS / Azure DevOps Server. The tool is also usable by humans, but AI agent ergonomics take priority in design decisions (output format, error messages, predictability).

## Key Success Metrics

An AI agent can call `tfscli`, read work item content, and use it meaningfully in its workflow — for example, to understand a task description, check acceptance criteria, or reference related work items.

## Constraints

- **Platform:** primary target is Windows (most on-prem TFS environments). Cross-compilation to Linux/macOS is a bonus enabled by Go, not a priority.
- **API coverage in MVP:** Work Items → Get Work Item (single ID); Work Items - List and Get Work Items Batch (multiple IDs; the latter, a POST, needs Azure DevOps Server 2019 or later). Fields selectable via `--fields` parameter; all fields returned by default. Wiql → Query By Wiql (find work items with a WIQL query, available since TFS 2015); it returns work item IDs and links only, which are then read with Work Items - List.
- **Target API version:** negotiated with the server, not configured. Microsoft documents `api-version` as required on every request, and some servers refuse a POST request without it, such as Get Work Items Batch or Query By Wiql. Without a configured version, each request is therefore preceded by `OPTIONS {url}/{collection}/_apis`, which lists every resource of the server with its versions, as the official SDKs do, and carries the released version of its resource: the version the server chooses for a request without one, so the same configuration works with older TFS installations and with the latest Azure DevOps Server alike. The answer is not kept between calls, since tfscli is stateless. When the negotiation fails for any reason other than a refused token or an unreachable server, the request is sent without a version, so that no server tfscli worked with stops working; if the server then refuses it for the missing version, the `config` error names the settings that supply one. A version can be pinned in the config file, the environment, or per call, and is then sent unchanged, without negotiation.
- **Authentication:** PAT only. No SSPI or NTLM. A PAT is issued for one collection, so the collection is part of the credential. The server URL, the collection, and the PAT are stored together and always come from the same source, so the token is never sent to any other server or collection: either `$XDG_DATA_HOME/tfscli/auth.json` (default `~/.local/share/tfscli/auth.json`, on every OS), written only by `tfscli auth login`, or the `TFSCLI_AUTH` environment variable holding the same JSON, which takes precedence and serves CI. `auth.json` holds exactly one credential, is written with mode `0600` (directory `0700`), and stores the token in plaintext. No flag, separate environment variable, or config key supplies the URL, the collection, or the token. See "Authentication" below.
- **No OS keychain.** Against code running as the same user a keychain adds no protection; other users are excluded by file permissions, and stolen disks by disk encryption. A keychain is reconsidered only if an organisation requires secrets to be kept in the system store, and would then be a second storage backend behind the same `auth login`.
- **Output:** Markdown by default. JSON as optional flag in future versions.
- **Distribution:** open source on GitHub, pre-built binaries in releases.

## Non-Goals

- **Write operations.** v1 is strictly read-only. Write operations are a future consideration, gated by PAT scopes. JSON output (`--json`) is a prerequisite for write support, since markdown conversion is lossy and not round-trippable to HTML. When write operations are added, they are gated on the command line as follows, so that an AI agent host can require confirmation for every change with one permission rule:
  - Every operation that changes server state requires the flag `--allow-changes`. An irreversible operation requires both `--allow-changes` and `--allow-irreversible`: a rule on `--allow-changes` then covers every change, and a rule on `--allow-irreversible` additionally excludes the irreversible ones. Without a required flag the command fails with a `config` error that names the flag.
  - The classification follows the meaning of the operation, not its HTTP method: Get Work Items Batch and Query By Wiql are POST requests and read only, while queueing a build changes server state. An operation is irreversible when its effect cannot be undone by another operation of the REST API: deleting a work item to the recycle bin is reversible, since it can be restored, and destroying it is not.
  - The gate flags are flags only, with no short form. No environment variable or config key supplies them, because a setting outside the command line is invisible to permission rules that match the command text; they are outside the precedence chain in "Configuration".
  - The gate is checked before any request is sent, the `OPTIONS` request that negotiates the API version included.
  - The gate covers server state only. Local commands that change local state, such as `auth login`, live under `auth` and are interactive.
  - The gate flags are not parameters of the REST API and so deviate from "Follow TFS API structure". The deviation is justified by safety: whether a command changes server state cannot be told from its name, since action names come from the API and form an open set, while some POST operations only read.
- **MCP server mode.** tfscli is a CLI utility, not an MCP server. If MCP integration is needed, it can be wrapped externally.
- **Cloud Azure DevOps Services support.** Target is on-premises TFS / Azure DevOps Server. Cloud may work incidentally but is not tested or guaranteed.
- **Custom query syntax.** No invented query language. WIQL queries are passed directly to the API.
- **GUI or interactive mode.** CLI only, designed for scripting and agent consumption. The one exception is `tfscli auth login`, which prompts for the credential once per machine so that the token never appears as a command argument.
- **Caching.** No local caching of API responses. Every call hits the server.

## Design Principles

- **Follow TFS API structure.** Command hierarchy mirrors API domains. Parameter names match API parameter names where possible. If TFS API has a batch endpoint — tfscli exposes batch. If it doesn't — tfscli doesn't invent one. Where the API offers several operations for the same purpose, such as a GET and a POST variant, tfscli exposes each of them rather than choosing one, and passes their parameters through rather than replacing them with behaviour of its own. Commands and their arguments are named by a fixed rule, without exceptions:
  - A command is `tfscli <area> <resource> <action>`. The area and the resource are the segments of the operation's page path in the REST API reference, which the reference already gives in kebab-case: `.../rest/api/azure/devops/wit/work-items/get-work-item` is `wit work-items get`, `.../wit/wiql/query-by-wiql` is `wit wiql query-by-wiql`. The area is kept even where the resource name alone would be unambiguous, because the API does not keep resource names unique across areas: Items exists in both `git` and `tfvc`. The rule names REST API operations only: `tfscli auth login` is a local command of tfscli, not an operation of the API, and lies outside it.
  - The action is the name of the operation in the REST API reference with the resource name removed, in kebab-case: "Get Work Item" is `wit work-items get`, "List" is `wit work-items list`, "Get Work Items Batch" is `wit work-items get-batch`. The resource name is removed only where it names what the operation acts on; where it names something else, it stays: in "Query By Wiql" of the resource Wiql it names the input of the query, as "Id" does in "Query By Id", so the actions are `query-by-wiql` and `query-by-id`.
  - A path parameter that identifies the resource is a positional argument: `wit work-items get 12345` for `_apis/wit/workitems/{id}`. The project, the team, and the collection, which scope the request rather than identify the resource, are not: the collection comes from the credential, the project from `-p` or its defaults, and the team from `--team`.
  - A query or body parameter is a flag with the API name in kebab-case and without a leading `$`: `ids` is `--ids`, `errorPolicy` is `--error-policy`, `$expand` is `--expand`. A flag that is not given is not sent, so the server's default applies.
- **The command line can be governed by permission rules.** AI agent hosts, such as Claude Code and opencode, decide whether an agent may run a command by matching its text against allow, ask, and deny rules. tfscli keeps that text predictable:
  - The command path opens the command line, and flags follow its last word: `tfscli wit work-items get --verbose 12345`, not `tfscli --verbose wit work-items get 12345`. A flag placed earlier is refused with a `config` error that shows the corrected command. The rule has no exceptions: `--help` follows the path like any other flag, and the built-in `help` and `completion` commands are commands like any other. Only an invocation without a command path, such as `tfscli --help` or `tfscli --version`, has no path for a flag to precede; `--version` belongs to the root alone.
  - Each command has exactly one form: no command aliases, no abbreviated command or flag names, no prefix matching of command names.
  - A command that has subcommands — an area, a resource, or a local group such as `auth` — only groups them: it takes no arguments, defines no flags of its own, and does nothing but print its help. Only a command without subcommands runs. A group that also acts on its own, as `git stash` and `git remote` do, is excluded: a rule written for the group would also cover every command below it, and the command path could no longer be told from the group's own arguments.
  - The names of local commands (`auth`, `licenses`, `help`, `completion`) never coincide with an area of the REST API reference.
  - Whether a command changes server state is a property of the command, never of its flags: a reading command does not gain a side effect through a flag. There is no generic pass-through command that sends an arbitrary request; should one be added, it falls under the gate flags described in "Non-Goals".
- **Predictable error output.** Errors go to stderr with a machine-readable category and human-readable message. Exit code is non-zero. Format: `Error [category]: message (HTTP status)`. Error categories are a stable contract: existing categories are never removed or renamed; new categories may be added in future versions.
- **Minimal configuration.** `auth login` stores the server URL, the collection, and the PAT. An optional config file stores an optional default project, the API version, and TLS settings. Environment variables and flags override config values. No other configuration needed.
- **Clean output for AI consumption.** HTML in work item fields is converted to markdown. Metadata noise (URLs, internal IDs, revision details) is minimized in default output.
- **Works out of the box; errors lead the way.** tfscli assumes that the user has read no documentation and prepared nothing beforehand. Required setup is limited to what tfscli cannot determine unambiguously on its own, and no setting is required merely because tfscli lacks a working default. tfscli does not guess and does not make choices on the user's behalf: where a value has more than one plausible answer, it asks the user to supply it. Since tfscli is not interactive, apart from `auth login`, error messages are the means of guidance: every error a first-time user can encounter states the concrete next step — the command to run, or the flag, variable, or value to set — and, where only a person can take that step, as with the interactive `auth login`, says so. An error that leaves the user without a next step is a defect.
- **Server compatibility is preserved.** On-premises installations are upgraded rarely, so the range of TFS / Azure DevOps Server versions tfscli works with should not shrink. A change should not stop tfscli from working with a server it worked with before, whether through the REST API version it requires or through any other server-side dependency. Where a new feature needs a newer server, it is preferably optional: on an older server it fails with a clear error and everything else keeps working. Dropping support for older servers is not prohibited, but it is a deliberate decision: it is made only when the benefit justifies the servers lost, after explicit discussion, and is recorded together with its justification and the alternatives rejected.

## Authentication

`tfscli auth login` takes no flags. It refuses to run when stdin is not a terminal, asks for the server URL, then for the collection, then for the PAT with echo turned off, verifies the three with a request to `{url}/{collection}/_apis/connectionData`, and writes `auth.json` only if the request succeeds. The request is made at collection level because a PAT is issued for a collection: a server may refuse it at server level while accepting it within the collection. Verification failures are reported in the error categories below; a 404 names the address the URL and the collection make up and asks to check both. The URL is normalised before it is stored: lower-case scheme and host, no trailing slash. When the last segment of its path names the collection, as in a URL copied from the browser, that segment is removed and the user is told so. TLS settings come from the config file when it exists. The verification request carries no `api-version`, whatever the config file, `TFSCLI_API_VERSION`, or `--api-version` say: `connectionData` has no released version, and the request reads no data whose shape a version would fix. `--api-version` given to `auth login` is refused with a `config` error.

```json
{
  "url": "https://tfs.example.com:8080/tfs",
  "collection": "DefaultCollection",
  "pat": "..."
}
```

`TFSCLI_AUTH` carries the same JSON for environments without an interactive terminal. When it is set, `auth.json` is not read.

## Configuration

File: `$XDG_CONFIG_HOME/tfscli/config.json`, defaulting to `~/.config/tfscli/config.json` when `XDG_CONFIG_HOME` is unset. The XDG Base Directory rules apply on every OS, Windows included. The file is optional.

```json
{
  "project": "MyProject"
}
```

All values overridable via environment variables (`TFSCLI_PROJECT`, `TFSCLI_API_VERSION`) and command-line flags (`-p`, `--api-version`). The API version is set with `apiVersion` in the config file, and has no default: without it the version is negotiated with the server (see "Target API version"). The TLS settings `caBundle` and `insecureSkipVerify` are accepted in the config file only.

Priority: CLI flag > environment variable > config file > built-in default.

Two kinds of input are outside this chain: the credential (see "Authentication") and the gate flags of write operations (see "Non-Goals"), which are given on the command line only.

## Command Format

```
tfscli <area> <resource> <action> [flags] [arguments]
```

The area, the resource, and the action are named by the rule in "Follow TFS API structure". Flags follow the action, as "The command line can be governed by permission rules" requires. `tfscli auth login` is a local command of tfscli and lies outside this format.

Commands:

```
tfscli wit work-items get -p MyProject 12345
tfscli wit work-items get -p MyProject 12345 --fields System.Title,System.State,System.Description
tfscli wit work-items list -p MyProject --ids 297,299,300
tfscli wit work-items get-batch -p MyProject --ids 297,299,300
tfscli wit wiql query-by-wiql -p MyProject --query "SELECT [System.Id] FROM WorkItems WHERE [System.State] = 'Active'"
tfscli auth login
```

Project flag `-p` is required unless default project is set in config or `TFSCLI_PROJECT`, except for `tfscli wit wiql query-by-wiql`, where the project is optional: without one the query runs at collection level, and `--team` then cannot be used.

## Error Format

Categories (`auth`, `not_found`, `forbidden`, `server`, `config`, `network`) are a stable contract — see Design Principles.

```
Error [auth]: the server did not accept the PAT (it may be invalid, expired, or revoked; if it is valid, IIS Basic Authentication may be enabled on the server, which only an administrator can turn off) (HTTP 401)
Error [not_found]: work item 99999 not found in project MyProject (HTTP 404)
Error [forbidden]: no access to project MyProject (HTTP 403)
Error [server]: TFS returned HTTP 500
Error [config]: not logged in: no credential at ~/.local/share/tfscli/auth.json (run "tfscli auth login", or set TFSCLI_AUTH)
Error [network]: cannot reach https://tfs.example.com:8080
```

## Key Dependencies

- `github.com/JohannesKaufmann/html-to-markdown/v2` — HTML to markdown conversion for work item fields
- `golang.org/x/term` — hidden PAT input and the terminal check in `tfscli auth login`
