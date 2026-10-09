# tfscli

A stateless CLI for read-only access to on-premises TFS / Azure DevOps Server, designed for AI coding agents and equally usable by humans. Single binary, PAT authentication, markdown output.

Every invocation hits the server: there is no daemon, no background process, and no on-disk cache.

## Features

- `wit work-items get` — one work item as markdown, whole or narrowed by `--fields`
- `wit work-items list` and `wit work-items get-batch` — several work items by id in one request
- `wit wiql query-by-wiql` — find work items with a WIQL query
- `auth login` — store the server URL, the collection, and a PAT, checked against the server
- A stable error contract and `--verbose` request logging

Not present in v1: write operations (out of scope) and JSON output (`--json`, planned — markdown is lossy, so writes depend on it landing first). See `docs/project-brief.md` for scope and `docs/architecture.md` for design decisions. Known defects are listed under [Known limitations](#known-limitations).

## Quick start

1. Download the archive for your platform from the [releases page](https://github.com/dpleshakov/tfscli/releases), unpack it, and put `tfscli` (`tfscli.exe` on Windows) in a directory on your `PATH`.
2. In the TFS web interface, issue a personal access token for your collection with these scopes only:
   - **Work Items (Read)**
3. Run `tfscli auth login` in a terminal and enter the server URL, the collection, and the token.
4. Check the setup by reading a work item you know exists:

   ```
   tfscli wit work-items get -p MyProject 12345
   ```

5. Copy the `skills/tfscli` directory from the archive into the skills directory of your AI agent.

Details for each step: [Installation](#installation) covers the archive names, checksums, and building from source; [Authentication](#authentication) covers the token scope, Git Bash on Windows, and CI; [Use with an AI agent](#use-with-an-ai-agent) lists the skills directories of the supported agents, gives the copy commands, and describes an agent without skill support. Further settings, such as a default project, are described under [Configuration](#configuration).

## Installation

### Pre-built binaries

Download an archive from the [releases page](https://github.com/dpleshakov/tfscli/releases), unpack it, and put `tfscli` somewhere on your `PATH`. Each release carries six archives, named `tfscli-<version>-<os>-<arch>`:

| Platform | amd64 | arm64 |
|---|---|---|
| Linux | `linux-amd64.tar.gz` | `linux-arm64.tar.gz` |
| Windows | `windows-amd64.zip` | `windows-arm64.zip` |
| macOS | `darwin-amd64.tar.gz` | `darwin-arm64.tar.gz` |

Every archive unpacks into a directory of the same name, `tfscli-<version>-<os>-<arch>/`, holding the binary, `LICENSE`, and the agent skill in `skills/tfscli/SKILL.md`. SHA-256 sums for all six archives are in `checksums.txt`, attached to the same release.

Verify a download before using it:

```
sha256sum --check --ignore-missing checksums.txt
```

Check what you installed:

```
tfscli --version
```

### From source

Requires Go 1.26 or newer:

```
go build -o tfscli ./cmd/tfscli
```

Cross-compilation follows the usual Go pattern:

```
GOOS=windows GOARCH=amd64 go build -o tfscli.exe ./cmd/tfscli
```

A binary built this way reports its version as `dev (unknown)`: the real values are stamped in by the release build. Its `tfscli licenses` prints the licenses of tfscli and Go alone: the third-party license texts are generated and embedded by the release build as well.

## Authentication

A personal access token is issued for one collection, so the server URL, the collection, and the token are stored together, and the token is only ever sent to the URL and the collection stored with it.

Issue the token in the TFS web interface for the collection you will log in to, with these scopes only, and leave every other scope off:

- **Work Items (Read)**

tfscli only reads data, so no scope beyond reading is needed. A scope limits what the token can do but does not extend what its owner can do: a work item in a project or area the user has no access to is refused with a `forbidden` error whatever the scope.

Store them once per machine:

```
$ tfscli auth login
Server URL (e.g. https://tfs.example.com:8080/tfs): https://tfs.example.com:8080/tfs
Collection (e.g. DefaultCollection): DefaultCollection
Personal access token:
Logged in to https://tfs.example.com:8080/tfs, collection DefaultCollection, as Jane Doe
```

The token is typed with echo turned off. The three values are checked against the server before anything is written: a rejected token, or a URL and collection that lead nowhere, is reported as an error and leaves nothing behind. The command takes no flags, refuses `--api-version`, and sends its check without `api-version` whatever the config file or `TFSCLI_API_VERSION` say.

Particular setups:

- **A URL copied from the browser** often ends with the collection. When its last path segment names the collection entered next, that segment is removed, and the command says which server URL it uses instead.
- **A server behind an internal CA.** `auth login` reads `caBundle` and `insecureSkipVerify` from the config file when there is one; see [TLS](#tls).
- **Git Bash (mintty) on Windows.** stdin is not a console there, and the command needs an interactive terminal. Run it from Windows Terminal, PowerShell, or `cmd`, or prefix it with `winpty`.
- **CI and other environments without a terminal.** Use `TFSCLI_AUTH`, described below.

The credential is written to `$XDG_DATA_HOME/tfscli/auth.json`, or `~/.local/share/tfscli/auth.json` when `XDG_DATA_HOME` is not set — the same location on every OS. The file is created with mode `0600` and its directory with `0700`; on Windows it inherits the permissions of the user profile. It holds exactly one credential: running `auth login` again replaces it. Working with another collection means logging in again with a token issued for it.

Where an interactive login is impossible, put the same JSON into `TFSCLI_AUTH`. When the variable is set, `auth.json` is not read:

```
TFSCLI_AUTH='{"url": "https://tfs.example.com:8080/tfs", "collection": "DefaultCollection", "pat": "…"}'
```

All three fields are required. The URL, the collection, and the token always come from the same source, and no flag, separate environment variable, or config key can supply any of them, so a mistyped or injected server address cannot make tfscli send the token to another host.

PAT is the only authentication method; SSPI and NTLM are out of scope. The token is sent as HTTP Basic authentication with an empty user name, which is what the TFS REST API expects. It is not kept in the OS keychain, which adds no protection against code running as the same user; other users are excluded by the file permissions.

## Use with an AI agent

tfscli is built for AI coding agents first, and every archive carries an agent
skill beside the binary: `skills/tfscli/SKILL.md`, in the Agent Skills format.
It tells an agent when the tool applies, how the output is shaped, what each
error category calls for, and where the boundaries are — none of which `--help`
conveys.

Install it by copying the directory. Create the destination first: copying
into a path that does not exist yet leaves `SKILL.md` one level too high,
where no agent looks for it.

```sh
mkdir -p ~/.claude/skills && cp -r skills/tfscli ~/.claude/skills/
```

```powershell
New-Item -ItemType Directory -Force $HOME\.claude\skills | Out-Null
Copy-Item -Recurse skills\tfscli $HOME\.claude\skills\
```

That one location serves both agents: opencode has first-party skill support and
searches `~/.claude/skills/` among its own paths. Other locations work as well,
if the skill should be scoped to a single project or kept out of the Claude
directory:

| Agent | Project | Global |
|---|---|---|
| Claude Code | `.claude/skills/tfscli/` | `~/.claude/skills/tfscli/` |
| opencode | `.opencode/skills/tfscli/`, `.claude/skills/tfscli/`, `.agents/skills/tfscli/` | `~/.config/opencode/skills/tfscli/`, `~/.claude/skills/tfscli/`, `~/.agents/skills/tfscli/` |

In opencode, loading a skill is subject to `permission.skill` in
`opencode.json` — `allow`, `ask`, or `deny` — and the mechanism can be switched
off altogether with `tools.skill: false`. If the skill is never offered, check
those settings first.

An agent without skill support can be pointed at the same file directly: it is
plain markdown under a short YAML header.

### Permission rules

Both agents decide whether a shell command may run by matching its text against
rules. Every tfscli command starts with its full name, such as
`tfscli wit work-items get` or `tfscli licenses`, followed by flags and
arguments, and each command has exactly one name, so a rule written as a prefix
covers the command whatever its flags. The current commands only read;
`auth login` is interactive and is not for an agent to run.

Commands that change data are not part of tfscli yet. When they are added,
each will require the flag `--allow-changes` and will fall under `tfscli wit *`
like the reading commands, so the examples below already ask before a command
that carries that flag as it is usually written.

Claude Code, in `.claude/settings.json` or `~/.claude/settings.json`; the
`PowerShell` rules apply where the agent runs commands through PowerShell. An
`ask` rule takes precedence over an `allow` rule:

```json
{
  "permissions": {
    "allow": [
      "Bash(tfscli wit *)",
      "Bash(tfscli licenses)",
      "PowerShell(tfscli wit *)",
      "PowerShell(tfscli licenses)"
    ],
    "ask": [
      "Bash(*--allow-changes*)",
      "PowerShell(*--allow-changes*)"
    ],
    "deny": [
      "Bash(tfscli auth *)",
      "PowerShell(tfscli auth *)"
    ]
  }
}
```

opencode, in `opencode.json`. The last matching rule wins, so place these after
any catch-all `"*"` rule, in this order:

```json
{
  "permission": {
    "bash": {
      "tfscli wit *": "allow",
      "tfscli licenses": "allow",
      "*--allow-changes*": "ask",
      "tfscli auth *": "deny"
    }
  }
}
```

A rule matches the command as the agent types it. A call through `tfscli.exe`
or a full path matches none of the `tfscli` rules and gets the agent's default
for unmatched commands: Claude Code asks in its default mode, while opencode
allows it unless a catch-all `"*": "ask"` is set. Likewise, a flag written
with quotes in the middle, such as `--allow-chan''ges`, reaches tfscli as
`--allow-changes` but may not match the rule. The rules therefore govern the
usual form of a call and guard against an agent's mistake; they are not a
security boundary against an agent that sets out to get around them.

## Configuration

Every setting apart from the credential:

| Setting | Config key | Environment variable | Flag | Default | Required |
|---|---|---|---|---|---|
| Team project | `project` | `TFSCLI_PROJECT` | `-p`, `--project` | — | per command |
| REST API version | `apiVersion` | `TFSCLI_API_VERSION` | `--api-version` | negotiated with the server | no |
| Request logging | — | `TFSCLI_VERBOSE=1` | `--verbose` | off | no |
| CA bundle | `caBundle` | — | — | none | no |
| Skip certificate verification | `insecureSkipVerify` | — | — | `false` | no |

Every flag in the table is global except `-p` / `--project`, which belongs to the commands that act within a project: it is required for the `wit work-items` commands and optional for `wit wiql query-by-wiql`.

The settings come from four sources. Later sources override earlier ones:

1. Built-in defaults
2. The config file `$XDG_CONFIG_HOME/tfscli/config.json`, or `~/.config/tfscli/config.json` when `XDG_CONFIG_HOME` is not set. The same location is used on every OS, Windows included.
3. Environment variables
4. Command-line flags

The config file is optional: every setting it holds, except the TLS ones, can be given by an environment variable or a flag instead. To use one, create it at that location; for example:

```json
{
  "project": "MyProject"
}
```

### REST API version

Without a configured version, tfscli asks the server for one. Each command first sends `OPTIONS` to `{url}/{collection}/_apis`, which lists every resource of the server with its versions, and the request then carries the released version of its resource. That is the version the server would choose for a request without one, so no setting is needed on any server release. The extra request shows under `--verbose`.

If the server does not answer `OPTIONS` with that list, the request is sent without a version, and `--verbose` says why. A server may refuse a POST request without a version — `wit work-items get-batch` and `wit wiql query-by-wiql` send one — and the `config` error then names the settings above. A token the server does not accept, or a server that cannot be reached, is reported at once.

A configured version is sent unchanged, without `OPTIONS`. Set one only for a server that refuses requests without a version, or to pin the shape of the response. `auth login` does not use this setting.

The version must not exceed the highest one the server supports:

| Server | Highest API version |
|---|---|
| Azure DevOps Server 2022.1 | 7.1 |
| Azure DevOps Server 2022 | 7.0 |
| Azure DevOps Server 2020 | 6.0 |
| Azure DevOps Server 2019 | 5.0 |
| TFS 2018 Update 2 | 4.1 |

The full mapping is under "API and TFS version mapping" in the [REST API reference](https://learn.microsoft.com/en-us/rest/api/azure/devops/). The value is sent unchanged, so a resource that exists only in preview at that version needs the `-preview` suffix.

### TLS

`caBundle` and `insecureSkipVerify` are accepted in the config file only, so that relaxing TLS is always a written-down decision. `caBundle` is the path to a PEM bundle appended to the system root pool; use it for an internal CA. `insecureSkipVerify` set to `true` disables certificate verification entirely and prints a warning under `--verbose`.

## Usage

```
tfscli <area> <resource> <action> [flags] [arguments]
```

Commands are named after the REST API reference: the area and the resource are the segments of the operation's page path, so Get Work Item, documented under `.../wit/work-items/get-work-item`, is `wit work-items get`. `auth login` and `licenses` are local to tfscli and have no counterpart in the API.

Flags follow the command: `tfscli wit work-items get --verbose -p MyProject 12345`, never `tfscli --verbose wit work-items get ...`. A flag placed before the last word of the command is refused with a `config` error that shows the command with the flag moved, and nothing is sent to the server; a flag that the command does not define is reported as unknown instead. `--help` is no exception: `tfscli wit work-items get --help`, not `tfscli --help wit work-items get`. `--version` belongs to `tfscli` alone, and placed before a command it is refused with a message that names `tfscli --version`. The rule keeps the command at the start of the command line, where agent permission rules look for it (see "Permission rules").

`tfscli --version` prints the version and the commit it was built from; `tfscli --help`, and `--help` on any command, lists the flags. `tfscli licenses` prints the components built into the binary — tfscli itself, the Go standard library, and the third-party modules — followed by the license text of each.

### `wit work-items get`

Print one work item as markdown:

```
tfscli wit work-items get -p MyProject 12345
```

Request a subset of fields. They are printed in the order they were asked for:

```
tfscli wit work-items get -p MyProject 12345 --fields System.Title,System.State,System.Description
```

Without `--fields`, every field of the work item is printed in the order the server returned it.

### `wit work-items list` and `wit work-items get-batch`

Print several work items, at most 200, with one request:

```
tfscli wit work-items list -p MyProject --ids 297,299,300
```

The two commands are the two operations the REST API offers for this, and take the same flags. `wit work-items list` is Work Items - List, a GET that carries the ids in the URL. `wit work-items get-batch` is Get Work Items Batch, a POST that carries them in the request body, so it is not limited by the length of the URL; it needs Azure DevOps Server 2019 or later, and a 404 without a message from the server says so and names `wit work-items list` instead.

| Flag | API parameter | Effect |
|---|---|---|
| `--ids` | `ids` | Comma-separated work item ids. Required. |
| `--fields` | `fields` | Comma-separated field names, as for `wit work-items get`. |
| `--as-of` | `asOf` | Read the work items as they were at this UTC time, e.g. `2026-06-14T09:00:00Z`. |
| `--error-policy` | `errorPolicy` | `fail` or `omit`. With `fail`, the server's default, a work item that does not exist or cannot be read fails the whole request; with `omit`, the others are returned. |

A flag that is not given is not sent, and `--as-of` and `--error-policy` are passed to the server unchecked: what it accepts is the server's to decide.

### `wit wiql query-by-wiql`

Find work items with a WIQL query:

```
tfscli wit wiql query-by-wiql -p MyProject --query "SELECT [System.Id] FROM WorkItems WHERE [System.State] = 'Active'"
```

The query is passed to the server as written; WIQL syntax errors come back from the server as `config` errors with its message. The server returns ids, not work items, so the command prints the ids; read the work items with `wit work-items list --ids`, at most 200 ids per call.

| Flag | API parameter | Effect |
|---|---|---|
| `--query` | `query` | The WIQL query. Required. |
| `-p`, `--project` | `project` | Team project. Optional for this command: without a project from any source, the query runs across the collection. |
| `--team` | `team` | Team of the project, for macros such as `@CurrentIteration`. Needs a project. |
| `--top` | `$top` | Return at most this many results. Without it the server's own limit applies. |
| `--time-precision` | `timePrecision` | Compare dates with the time of day, not only the date. |

A flag that is not given is not sent.

### Output

Short values are printed as `Name: value` lines; prose and multi-line values become sections. HTML fields (`System.Description`, `Microsoft.VSTS.TCM.ReproSteps`, `Microsoft.VSTS.TCM.SystemInfo`, `Microsoft.VSTS.Common.AcceptanceCriteria`) are converted to markdown. Identity fields print as `Display Name <unique.name>`, and timestamps are normalised to UTC with whole seconds, so the same work item renders identically on any machine.

```markdown
# Work item 12345 (rev 7)

System.WorkItemType: Bug
System.AssignedTo: Jane Doe <COMPANY\j.doe>
System.CreatedDate: 2026-06-14T09:12:33Z

## System.Description

A **partial** refund sends no email.
```

Field names are the raw TFS reference names — the same strings `--fields` accepts.

`wit work-items list` and `wit work-items get-batch` print each work item exactly as `wit work-items get` does, one after another, separated by a blank line, in the order the server returned them, which need not be the order of `--ids`. With `--error-policy omit`, each id the server did not return is printed after them as a heading with no body; the server does not say whether the work item does not exist or cannot be read:

```markdown
# Work item 298 (not returned: it does not exist, or the PAT has no access to it)
```

`wit wiql query-by-wiql` prints the type of the query and the time of the result, the columns the query selected as reference names in the form `--fields` takes, and the result. A flat query prints the ids in the form `--ids` takes:

```markdown
# WIQL query (flat, as of 2026-10-04T10:15:00Z)

Columns: System.Id,System.Title
Work items: 297,299,300
```

A tree or one-hop query prints one line per link, in the order the server returned them: the id alone for a top-level work item, otherwise the source, the target, and the link type:

```markdown
# WIQL query (tree, as of 2026-10-04T10:15:00Z)

Columns: System.Id,System.Title
Relations:
- 297
- 297 -> 299 (System.LinkTypes.Hierarchy-Forward)
- 297 -> 300 (System.LinkTypes.Hierarchy-Forward)
```

An empty result reads `Work items: none` or `Relations: none`. The time of the result is printed in UTC with the precision the server sent, unlike the timestamps of work items, so that it can be passed to `--as-of` of `wit work-items list` to read the work items as the query saw them.

### Request logging

`--verbose` writes one line per request to stderr — method, URL, status, duration — leaving stdout clean for the markdown. Headers and bodies are never logged, so the PAT cannot leak through it. A failed round trip is logged with status `0`. The `OPTIONS` request that negotiates the API version is logged like any other:

```
$ tfscli wit work-items get -p MyProject 12345 --verbose
OPTIONS https://tfs.example.com:8080/tfs/DefaultCollection/_apis 200 41.2087ms
GET https://tfs.example.com:8080/tfs/DefaultCollection/MyProject/_apis/wit/workitems/12345?api-version=7.1 200 86.4512ms
```

Lines starting with `[warn]` report a condition worth knowing, such as disabled certificate verification; lines starting with `[info]` report what tfscli did on its own, such as sending a request without a version:

```
[info] api-version not negotiated (OPTIONS returned HTTP 405); sending the request without it
```

## Errors

Errors go to stderr in a stable format, and the exit code is non-zero — 1 for every category in v1:

```
Error [category]: message (HTTP status)
```

The `(HTTP status)` part is omitted for errors that did not come from an HTTP response. Where TFS supplies its own message it is passed through, collapsed to one line; otherwise a generic message for the category is used. Categories are a contract: existing ones are never removed or renamed, though new ones may be added.

| Category | Meaning |
|---|---|
| `auth` | The server did not accept the PAT (HTTP 401): it is invalid, expired, or revoked, or IIS Basic Authentication is enabled on the server. |
| `forbidden` | Authenticated, but access was denied (HTTP 403). |
| `not_found` | The work item, project, or team does not exist (HTTP 404). During `auth login`, the server URL or the collection leads nowhere. |
| `server` | TFS failed or returned an unparseable response (HTTP 5xx). |
| `config` | Missing or invalid configuration, a bad argument, or a request TFS rejected. When TFS refuses a configured API version, the message names the setting it came from; when it refuses a request for carrying no version, after the version could not be negotiated, the message names the settings that supply one. |
| `network` | The server could not be reached, or the request timed out or was canceled. |

Examples:

```
$ tfscli wit work-items get -p MyProject 12345
Error [config]: not logged in: no credential at C:\Users\you\.local\share\tfscli\auth.json (run "tfscli auth login", or set TFSCLI_AUTH)

$ tfscli wit work-items get 12345
Error [config]: project is not set (pass -p, set TFSCLI_PROJECT, or add "project" to the config file)

$ tfscli wit work-items get -p MyProject abc
Error [config]: work item id "abc" is not a positive integer

$ tfscli wit work-items list -p MyProject 297 299
Error [config]: list takes no arguments; pass the work item ids with --ids, e.g. --ids 297,299

$ tfscli wit wiql query-by-wiql --team Web --query "SELECT [System.Id] FROM WorkItems"
Error [config]: --team needs a project, and project is not set (pass -p, set TFSCLI_PROJECT, or add "project" to the config file)

$ tfscli wit work-items get -p MyProject 12345
Error [config]: The requested REST API version of 7.2 is out of range for this server. The latest REST API version this server supports is 7.1. (api-version "7.2" is set by TFSCLI_API_VERSION; remove it to let the server choose the version, or set one the server supports) (HTTP 400)

$ tfscli wit work-items get -p MyProject 12345
Error [network]: cannot reach https://tfs.example.com:8080
```

## Known limitations

- **HTML tables collapse.** A `<table>` in a rich-text field becomes its cell text run together, without separators, so a work item whose Description holds a table renders unreadably.
- **Rich-text noise is unhandled.** @-mentions, attachment links, Word- and Outlook-pasted markup, and work-item references are converted literally, with whatever wrapper markup TFS stored. The fix for this and for tables is recorded in `docs/backlog.md`.
- **Servers with anonymous access are not supported.** On a server that admits anonymous requests, such as Azure DevOps Server with public projects enabled, an invalid PAT may pass the login check and be stored, and the other commands would then read as the anonymous user, seeing less, without an error.

## Development

Everything the project verifies runs through the `Makefile`, and CI runs `make check` verbatim, so a green `make check` locally is the whole gate. External dependencies are deliberately limited to `cobra`, `html-to-markdown/v2`, and `golang.org/x/term`; HTTP, JSON, and config parsing use the standard library.

[CONTRIBUTING.md](CONTRIBUTING.md) is the developer handbook: required tooling, the make targets, the lint and coverage rules, the process the repository follows, and the release procedure.

## License and trademarks

tfscli is released under the MIT License; see [LICENSE](LICENSE). `tfscli licenses` prints the licenses of the components included in the binary.

tfscli is an independent project and is not affiliated with, endorsed by, or sponsored by Microsoft. Team Foundation Server, Azure DevOps, and Azure DevOps Server are trademarks of the Microsoft group of companies.
