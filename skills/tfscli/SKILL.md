---
name: tfscli
description: Read work items from an on-premises TFS / Azure DevOps Server with the tfscli command-line tool. Use when a request refers to a TFS or Azure DevOps Server work item — a bug, task, user story, or PBI named by its numeric id — to an on-prem team project or collection, or asks for a work item's title, state, assignee, description, repro steps, or acceptance criteria, or asks to find work items by a condition such as state, assignee, area, or parent. Covers authentication, configuration and its precedence, the wit work-items get, list, and get-batch commands, the wit wiql query-by-wiql command, the markdown output format, the error categories and the action each one calls for, and the known limitations of the HTML-to-markdown conversion.
---

# tfscli

`tfscli` is a single-binary, read-only client for the TFS / Azure DevOps Server
REST API. It is stateless: there is no daemon and no cache, and every
invocation performs its requests against the server anew.

The requests it issues are:

```
GET  <url>/<collection>/<project>/_apis/wit/workitems/<id>         wit work-items get
GET  <url>/<collection>/<project>/_apis/wit/workitems?ids=<ids>    wit work-items list
POST <url>/<collection>/<project>/_apis/wit/workitemsbatch         wit work-items get-batch
POST <url>/<collection>[/<project>[/<team>]]/_apis/wit/wiql         wit wiql query-by-wiql
```

with `api-version=<version>` added to the query. Unless a version is
configured, each of them is preceded by `OPTIONS <url>/<collection>/_apis`,
which tells tfscli the version the server supports for the resource.

Authentication is a personal access token sent as HTTP Basic with an empty user
name. Output is markdown on stdout; errors are one line on stderr.

## When to use it

- A work item is named by its numeric id and its content is needed — the
  description of a task, the state of a bug, the acceptance criteria of a story.
- Several work items are named by id — the related items of a task, the ids a
  user listed — and are needed together.
- Work items have to be found rather than read: the active tasks of a user, the
  children of a feature, the bugs in an area. WIQL finds their ids.
- The server is an on-premises TFS or Azure DevOps Server instance.

## When not to use it

Apart from `auth login`, which is the user's to run, and `licenses`, which
prints the license texts of the code built into the binary and reads nothing
from the server, the tool covers four commands: `wit work-items get`,
`wit work-items list`, `wit work-items get-batch`, and
`wit wiql query-by-wiql`. Do not attempt anything below; none of it exists,
and inventing a flag or a subcommand produces an error, not a result.

- **No writes.** Nothing creates, updates, or comments on a work item. If the
  user asks for a change, report that the tool is read-only.
- **No search beyond WIQL.** The only way to find work items is a WIQL query
  passed to `wit wiql query-by-wiql`; there is no filter flag, no full-text
  search, and no saved (shared) query by name or id. `wit work-items list`
  reads the ids it is given; it does not list the work items of a project.
- **No other resources.** No repositories, builds, pipelines, pull requests,
  test plans, or wiki.
- **No JSON output.** There is no `--json` flag. The output is markdown, and it
  is the only shape available.
- **Cloud Azure DevOps Services is not supported.** It may work incidentally; it
  is not tested.

## Check that the tool is ready

```
tfscli --version
```

It prints `tfscli <version> (<commit>)`. A binary built from source rather than
released reports `tfscli dev (unknown)`, which is normal.

Configuration is not verified until a command runs. Do not create or edit
the config file on the user's behalf: a missing configuration is something to
report, not to guess at.

## Authentication

A PAT is issued for one collection, so the server URL, the collection, and the
PAT are stored together, and the token is only ever sent to the URL and the
collection stored with it. They come from one of two sources:

- `$XDG_DATA_HOME/tfscli/auth.json` (`~/.local/share/tfscli/auth.json` when
  `XDG_DATA_HOME` is not set, on every OS), written by `tfscli auth login`.
- `TFSCLI_AUTH`, holding the same JSON,
  `{"url": "…", "collection": "…", "pat": "…"}`. When it is set, `auth.json`
  is not read.

There is no flag, separate environment variable, or config key for the URL, the
collection, or the token. The server and the collection are therefore fixed by
the user's login; do not try to point tfscli at another one. If the user needs
a different collection, they have to log in again with a token issued for it.

`tfscli auth login` is for the user to run in their own terminal: it asks for
the token with echo turned off and refuses to run without an interactive
terminal. Do not run it, and do not write `auth.json` or set `TFSCLI_AUTH` on
the user's behalf. When a command reports `not logged in`, tell the user to
run `tfscli auth login`. The login check is sent without an API version; the
API version settings below do not apply to it.

## Configuration

The remaining settings come from four sources, each overriding the ones above
it: built-in defaults, the config file `$XDG_CONFIG_HOME/tfscli/config.json`
(`~/.config/tfscli/config.json` when `XDG_CONFIG_HOME` is not set, on every
OS), environment variables, command-line flags.

| Setting | Config key | Environment variable | Flag | Default | Required |
|---|---|---|---|---|---|
| Team project | `project` | `TFSCLI_PROJECT` | `-p`, `--project` | — | per command |
| REST API version | `apiVersion` | `TFSCLI_API_VERSION` | `--api-version` | negotiated with the server | no |
| Request logging | — | `TFSCLI_VERBOSE=1` | `--verbose` | off | no |

Two further keys are accepted in the config file only, with no environment
variable and no flag: `caBundle` (path to a PEM bundle appended to the system
root pool, for an internal CA) and `insecureSkipVerify` (disables certificate
verification entirely).

Without a configured API version, tfscli negotiates one with the server for
each request; nothing has to be set. A configured version is sent unchanged.
Do not set one unless the user asks for it.

When the server does not answer the negotiation, the request is sent without
a version. `wit work-items get-batch` and `wit wiql query-by-wiql` send a POST
request, which a server may then refuse: the `config` error says that no
`api-version` was supplied and that the version could not be negotiated. Do not
pick a version yourself, and do not take the `1.0` from the server's example.
Report the error and let the user set the version their server supports, with
`--api-version`, `TFSCLI_API_VERSION`, or `apiVersion` in the config file. For
`get-batch`, `wit work-items list` reads the same work items with a GET, which
servers accept without a version.

The config file is optional. Environment variables and flags alone are enough,
so `-p` can carry everything a call needs beyond the credential.

## The commands

Write the full command name first and every flag after it:
`tfscli wit work-items get --verbose -p <project> <id>`. A flag before the last
word of the command, as in `tfscli --verbose wit work-items get ...`, is refused
with a `config` error that shows the corrected command; the user's permission
rules match the command by its name, so keep that order rather than relying on
the error.

### One work item

```
tfscli wit work-items get -p <project> <id>
```

`<id>` is a positive integer. Anything else is rejected locally, without a
request.

| Flag | Effect |
|---|---|
| `-p`, `--project` | Team project. Required unless `project` is in the config file or `TFSCLI_PROJECT` is set. |
| `--fields` | Comma-separated TFS reference names. Without it, every field is printed. |
| `--verbose` | One line per request to stderr, the `OPTIONS` request that negotiates the API version included: method, URL, status, duration. Never headers or bodies. A line starting with `[info]` says what tfscli did on its own, such as sending a request without a version. |

Narrow the request whenever the needed fields are known — a full work item can
be large, and its tokens are paid for on every call:

```
tfscli wit work-items get -p MyProject 12345 --fields System.Title,System.State,System.Description
```

Requested fields are printed in the order they were asked for. Without
`--fields`, fields keep the order the server returned them in. A field name the
server does not recognise makes TFS reject the whole request — see the `config`
category below.

Field names are raw TFS reference names, and the output and `--fields` use the
same strings: `System.Title`, `System.State`, `System.WorkItemType`,
`System.AssignedTo`, `System.Description`, `Microsoft.VSTS.TCM.ReproSteps`,
`Microsoft.VSTS.Common.AcceptanceCriteria`, and so on. Custom fields carry the
project's own prefix.

### Several work items

```
tfscli wit work-items list -p <project> --ids <id>,<id>,...
tfscli wit work-items get-batch -p <project> --ids <id>,<id>,...
```

Both read up to 200 work items in one request and take the same flags. They
are the two operations the REST API offers for this: `list` sends the ids in
the URL, `get-batch` in the body of a POST. Use `list`. Use `get-batch` when
`list` fails because the URL is too long — many ids with a long `--fields`.
The web server in front of TFS refuses such a request without a TFS message of
its own, typically with HTTP 404 from IIS request filtering, or with 414.
`get-batch` needs Azure DevOps Server 2019 or later. On an older server it
fails, typically with `not_found` and a message that says so; `list` is then
the only choice.

The ids go in `--ids`, not as arguments; every id is checked locally, as for
`wit work-items get`.

| Flag | Effect |
|---|---|
| `-p`, `--project` | Team project, as for `wit work-items get`. |
| `--ids` | Comma-separated work item ids. Required. |
| `--fields` | Comma-separated TFS reference names, as for `wit work-items get`. |
| `--as-of` | Read the work items as they were at this UTC time, e.g. `2026-06-14T09:00:00Z`. |
| `--error-policy` | `fail` (the server's default) or `omit`. With `fail`, one id that does not exist or cannot be read fails the whole request with `not_found`. With `omit`, the other work items are returned. |

A flag that is not given is not sent; `--as-of` and `--error-policy` are
passed to the server unchecked.

Pass `--error-policy omit` when the ids come from somewhere that may hold
stale or inaccessible ones, such as the links of another work item: one
missing id then does not cost the rest.

### Finding work items with WIQL

```
tfscli wit wiql query-by-wiql -p <project> --query "<WIQL>"
```

The query is WIQL, passed to the server as written; there is no other query
syntax. It returns ids, not work items: read them afterwards with
`wit work-items list --ids`, which accepts at most 200 ids per call, so split a
longer list into portions of at most 200. When only the first results are
needed — the latest bugs, a sample — limit the query with `--top` rather than
reading everything it returns. Ask in `SELECT` only for `[System.Id]` unless
the column list matters: the fields of the work items are read by
`wit work-items list --fields`, not by the query.

| Flag | Effect |
|---|---|
| `--query` | The WIQL query. Required, and given as a flag, not as an argument; quote it as one shell word. |
| `-p`, `--project` | Team project. Optional here: without a project from any source, the query runs across the collection. |
| `--team` | Team of the project. Macros that depend on a team, such as `@CurrentIteration`, need it. It needs a project. |
| `--top` | Return at most this many results. |
| `--time-precision` | Compare dates with the time of day, not only the date. |

A flag that is not given is not sent.

Typical queries:

```
tfscli wit wiql query-by-wiql -p MyProject --query "SELECT [System.Id] FROM WorkItems WHERE [System.AssignedTo] = @Me AND [System.State] = 'Active'"
tfscli wit wiql query-by-wiql -p MyProject --query "SELECT [System.Id] FROM WorkItemLinks WHERE [Source].[System.Id] = 297 AND [System.Links.LinkType] = 'System.LinkTypes.Hierarchy-Forward' MODE (MustContain)"
tfscli wit wiql query-by-wiql -p MyProject --top 20 --query "SELECT [System.Id] FROM WorkItems WHERE [System.WorkItemType] = 'Bug' ORDER BY [System.CreatedDate] DESC"
```

A query that selects `FROM WorkItems` is flat and prints the ids; one that
selects `FROM WorkItemLinks` is a link query and prints the links. A syntax
error in the query, or a field it names that does not exist, is reported as a
`config` error carrying the server's message; fix the query rather than
retrying it unchanged.

## Output

Short values print as `Name: value` lines. Prose, and any value spanning more
than one line, becomes a `## Name` section. Four fields are stored as HTML by
TFS and are converted to markdown: `System.Description`,
`Microsoft.VSTS.TCM.ReproSteps`, `Microsoft.VSTS.TCM.SystemInfo`,
`Microsoft.VSTS.Common.AcceptanceCriteria`.

```markdown
# Work item 12345 (rev 7)

System.WorkItemType: Bug
System.AssignedTo: Jane Doe <COMPANY\j.doe>
System.CreatedDate: 2026-06-14T09:12:33Z
Microsoft.VSTS.Scheduling.RemainingWork: 4.5

## System.Description

A **partial** refund sends no email.

## Microsoft.VSTS.TCM.ReproSteps

1. Open the order
2. Refund half of it
```

Identity fields print as `Display Name <unique.name>`. Timestamps are
normalised to UTC with whole seconds, so the same work item renders identically
on any machine.

`wit work-items list` and `wit work-items get-batch` print each work item
exactly as above, one after another, separated by a blank line. The order is
the server's and need not match `--ids`; find a work item by its
`# Work item <id>` heading, not by position. With `--error-policy omit`, every id the server did not return
follows as a heading with no body:

```markdown
# Work item 298 (not returned: it does not exist, or the PAT has no access to it)
```

The server does not say which of the two it is. The text in the parentheses is
tfscli's note, not work item content.

`wit wiql query-by-wiql` prints the type of the query, the time of the result,
and the columns the query selected, then the result. A flat query prints the
ids in the form `--ids` takes, ready to pass on:

```markdown
# WIQL query (flat, as of 2026-10-04T10:15:00Z)

Columns: System.Id,System.Title
Work items: 297,299,300
```

A link query prints one line per link in the server's order: the id alone for
a top-level work item, otherwise `source -> target (link type)`:

```markdown
# WIQL query (tree, as of 2026-10-04T10:15:00Z)

Columns: System.Id,System.Title
Relations:
- 297
- 297 -> 299 (System.LinkTypes.Hierarchy-Forward)
- 297 -> 300 (System.LinkTypes.Hierarchy-Forward)
```

An empty result reads `Work items: none` or `Relations: none`: the query
matched nothing, and the output is not truncated.

The time after `as of` is the moment the server evaluated the query, in UTC
with the precision the server sent. When the work items have to match the
query result exactly — a state or an assignee the query filtered on — pass it
unchanged to `wit work-items list --as-of`.

## Errors

Failures go to stderr in one stable format, and the exit code is non-zero — 1
for every category in this version:

```
Error [category]: message (HTTP status)
```

The `(HTTP status)` part is absent for errors that did not come from an HTTP
response. Where TFS supplies its own message, it is passed through in place of
the generic wording. The categories are a contract: existing ones are never
removed or renamed, though new ones may appear.

| Category | Cause | What to do |
|---|---|---|
| `auth` | HTTP 401 — the server did not accept the PAT: it is invalid, expired, or revoked, or IIS Basic Authentication is enabled on the server. | Do not retry, and do not try another token. Tell the user the PAT needs to be renewed and stored again with `tfscli auth login`, or, if the PAT is known to be valid, that a server administrator has to turn IIS Basic Authentication off. |
| `forbidden` | HTTP 403 — authenticated, but access denied. | Do not retry. The PAT lacks the scope, or the project is closed to this user. Report it. |
| `not_found` | HTTP 404 — no such work item, project, or team. For `wit work-items list` and `wit work-items get-batch`, one missing id is enough. | Check the id, the project, and the team spelling against what the user gave. Do not scan ids looking for a match. For several ids, retry once with `--error-policy omit` to learn which are missing. |
| `server` | HTTP 5xx, or a response that could not be parsed. | One retry is reasonable. If it repeats, report the server as unavailable. |
| `config` | A missing or invalid setting, a missing or malformed credential, a bad argument, or a request TFS rejected — a 4xx other than 401, 403, and 404, such as an unknown field name or an unsupported API version. A refused API version names the setting it came from — `--api-version`, `TFSCLI_API_VERSION`, or `apiVersion` in the config file — and the next step. A request refused for carrying no version, sent so because the version could not be negotiated, names those three settings. | Fix the invocation if the fault is in it. If no version could be negotiated, report the message to the user; do not set a version yourself. If the API version was refused and it came from `--api-version` you passed, drop the flag and retry; if it came from the environment or the config file, report the message to the user rather than changing the setting. If a setting is missing, report what is missing; do not write the config file. If the user is not logged in, tell them to run `tfscli auth login`. |
| `network` | The server could not be reached, or the request timed out or was canceled. | Do not repeat the call in a loop. Report the server as unreachable and let the user check the URL and their connection. |

Representative messages:

```
Error [config]: not logged in: no credential at C:\Users\you\.local\share\tfscli\auth.json (run "tfscli auth login", or set TFSCLI_AUTH)
Error [config]: project is not set (pass -p, set TFSCLI_PROJECT, or add "project" to the config file)
Error [config]: work item id "abc" is not a positive integer
Error [config]: list takes no arguments; pass the work item ids with --ids, e.g. --ids 297,299
Error [config]: --team needs a project, and project is not set (pass -p, set TFSCLI_PROJECT, or add "project" to the config file)
Error [config]: The requested REST API version of 7.2 is out of range for this server. The latest REST API version this server supports is 7.1. (api-version "7.2" is set by TFSCLI_API_VERSION; remove it to let the server choose the version, or set one the server supports) (HTTP 400)
Error [auth]: the server did not accept the PAT (it may be invalid, expired, or revoked; if it is valid, IIS Basic Authentication may be enabled on the server, which only an administrator can turn off) (HTTP 401)
Error [network]: cannot reach https://tfs.example.com:8080
```

When the project is missing, ask the user for it. Do not guess a project name
from the repository, the branch, or the work item id.

## Known limitations

These are defects in the conversion, not in the data. Read the output with them
in mind rather than reporting a work item as empty or corrupt.

- **HTML tables collapse.** The converter runs without table support: a
  `<table>` becomes its cell text run together, with no separators. A
  description built around a table will look like one unbroken paragraph. If the
  content matters, say that the field holds a table that did not survive
  conversion.
- **Rich-text markup is passed through literally.** @-mentions, attachment
  links, work-item references, and markup pasted from Word or Outlook are
  converted as-is, including whatever wrapper markup TFS stored. Unexpected
  noise in a field is usually this, not the author's text.

## The token

The PAT is a credential. Never print it, and never pass a token as a command
argument of any program, which puts it into shell history, the process list,
and any transcript of the session. Never read `auth.json` or print
`TFSCLI_AUTH`, not even to check that a login exists: run a command and read
its error instead. tfscli itself takes the token only from those two sources,
and `--verbose` deliberately logs no headers, so a request log cannot leak it.

## The cost of a call

Every invocation is an HTTP round trip to the server; nothing is cached. Read
several known work items with one `wit work-items list` call rather than one
`wit work-items get` per id, reuse output already in the conversation instead
of fetching the same work item again, and narrow the request with `--fields` when
the needed fields are known. Do not iterate over a range of ids, singly or with
`--ids`.
