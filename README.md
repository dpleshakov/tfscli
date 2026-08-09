# tfscli

A stateless CLI for read-only access to on-premises TFS / Azure DevOps Server, designed for AI coding agents and equally usable by humans. Single binary, PAT authentication, markdown output.

Every invocation hits the server: there is no daemon, no background process, and no on-disk cache.

## Status

Pre-release, and **not yet run against a live TFS instance**. One command exists — `workitem get`. It is covered by tests that drive the full command tree against a stub HTTP server, so the request shape, the output format, and the error contract behave as documented; what has not been exercised is a real server with real work items. Expect the first live runs to surface discrepancies, particularly in rich-text rendering.

Available now:

- `workitem get` — one work item as markdown, whole or narrowed by `--fields`
- Configuration through a file, environment variables, and flags
- `--verbose` request logging and the full error contract

Not present in v1: write operations (out of scope) and JSON output (`--json`, planned — markdown is lossy, so writes depend on it landing first). See `docs/project-brief.md` for scope and `docs/architecture.md` for design decisions.

### Known limitations

- **HTML tables collapse.** The markdown converter runs on library defaults, which have no table support: a `<table>` becomes its cell text run together, without separators. A work item whose Description holds a table renders unreadably.
- **Rich-text noise is unhandled.** @-mentions, attachment links, Word- and Outlook-pasted markup, and work-item references are converted literally, with whatever wrapper markup TFS stored. Fixing this needs samples from a real instance and is tracked in `docs/2026-05-20-tasks-html-quirks.md`.

## Installation

Build from source with Go 1.26 or newer:

```
go build -o tfscli ./cmd/tfscli
```

Cross-compilation follows the usual Go pattern:

```
GOOS=windows GOARCH=amd64 go build -o tfscli.exe ./cmd/tfscli
```

## Configuration

Configuration comes from four sources. Later sources override earlier ones:

1. Built-in defaults
2. The config file `~/.tfscli/config.json`
3. Environment variables
4. Command-line flags

Copy `config.example.json` to `~/.tfscli/config.json` and fill in the values:

```json
{
  "url": "https://tfs.company.com:8080/tfs",
  "collection": "DefaultCollection",
  "pat": "YOUR_PAT_HERE",
  "project": "MyProject",
  "apiVersion": "7.2"
}
```

The PAT is stored in plaintext — a conscious v1 trade-off. On shared and CI machines, omit `pat` from the file and pass it through `TFSCLI_PAT` instead.

| Setting | Config key | Environment variable | Flag | Default | Required |
|---|---|---|---|---|---|
| Server URL | `url` | `TFSCLI_URL` | `--url` | — | yes |
| Collection | `collection` | `TFSCLI_COLLECTION` | `--collection` | — | yes |
| Personal access token | `pat` | `TFSCLI_PAT` | `--pat` | — | yes |
| Team project | `project` | `TFSCLI_PROJECT` | `-p`, `--project` | — | per command |
| REST API version | `apiVersion` | `TFSCLI_API_VERSION` | `--api-version` | `7.2` | no |
| Request logging | — | `TFSCLI_VERBOSE=1` | `--verbose` | off | no |

Every flag in the table is global except `-p` / `--project`, which belongs to the commands that need a project.

The config file is not required: environment variables and flags alone are enough to run. Its absence is reported only when a required setting is in fact missing.

### TLS

Two settings are accepted in the config file only, so that relaxing TLS is always a written-down decision:

| Config key | Effect |
|---|---|
| `caBundle` | Path to a PEM bundle appended to the system root pool. Use this for an internal CA. |
| `insecureSkipVerify` | Disables certificate verification entirely. Prints a warning under `--verbose`. |

### Authentication

PAT only. SSPI, NTLM, and interactive login are out of scope. The token is sent as HTTP Basic authentication with an empty user name, which is what the TFS REST API expects.

## Usage

```
tfscli <resource> <action> [flags] [arguments]
```

### `workitem get`

Print one work item as markdown:

```
tfscli workitem get -p MyProject 12345
```

Request a subset of fields. They are printed in the order they were asked for:

```
tfscli workitem get -p MyProject 12345 --fields System.Title,System.State,System.Description
```

Without `--fields`, every field of the work item is printed in the order the server returned it.

`-p` is required unless `project` is set in the config file or `TFSCLI_PROJECT` is exported.

### Output

Short values are printed as `Name: value` lines; prose and multi-line values become sections. HTML fields (`System.Description`, `Microsoft.VSTS.TCM.ReproSteps`, `Microsoft.VSTS.TCM.SystemInfo`, `Microsoft.VSTS.Common.AcceptanceCriteria`) are converted to markdown. Identity fields print as `Display Name <unique.name>`, and timestamps are normalised to UTC with whole seconds, so the same work item renders identically on any machine.

```markdown
# Work item 12345 (rev 7)

System.WorkItemType: Bug
System.AssignedTo: Anna Ivanova <COMPANY\a.ivanova>
System.CreatedDate: 2026-06-14T09:12:33Z

## System.Description

A **partial** refund sends no email.
```

Field names are the raw TFS reference names — the same strings `--fields` accepts.

### Request logging

`--verbose` writes one line per request to stderr — method, URL, status, duration — leaving stdout clean for the markdown. Headers and bodies are never logged, so the PAT cannot leak through it. A failed round trip is logged with status `0`.

```
$ tfscli workitem get -p MyProject 12345 --verbose
GET https://tfs.company.com:8080/tfs/DefaultCollection/MyProject/_apis/wit/workitems/12345?api-version=7.2 200 86.4512ms
```

## Errors

Errors go to stderr in a stable format, and the exit code is non-zero — 1 for every category in v1:

```
Error [category]: message (HTTP status)
```

The `(HTTP status)` part is omitted for errors that did not come from an HTTP response. Where TFS supplies its own message it is passed through, collapsed to one line; otherwise a generic message for the category is used. Categories are a contract: existing ones are never removed or renamed, though new ones may be added.

| Category | Meaning |
|---|---|
| `auth` | The PAT is invalid or expired (HTTP 401). |
| `forbidden` | Authenticated, but access was denied (HTTP 403). |
| `not_found` | The work item, project, or collection does not exist (HTTP 404). |
| `server` | TFS failed or returned an unparseable response (HTTP 5xx). |
| `config` | Missing or invalid configuration, a bad argument, or a request TFS rejected. |
| `network` | The server could not be reached, or the request timed out or was cancelled. |

Examples:

```
$ tfscli workitem get -p MyProject 12345
Error [config]: config file not found at C:\Users\you\.tfscli\config.json

$ tfscli workitem get 12345
Error [config]: project is not set (pass -p, set TFSCLI_PROJECT, or add "project" to the config file)

$ tfscli workitem get -p MyProject abc
Error [config]: work item id "abc" is not a positive integer

$ tfscli workitem get -p MyProject 12345
Error [network]: cannot reach https://tfs.company.com:8080
```

## Development

```
go build ./...
go test ./...
go vet ./...
gofmt -l .
```

External dependencies are deliberately limited to `cobra` and `html-to-markdown/v2`; HTTP, JSON, and config parsing use the standard library.
