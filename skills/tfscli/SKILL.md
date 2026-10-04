---
name: tfscli
description: Read work items from an on-premises TFS / Azure DevOps Server with the tfscli command-line tool. Use when a request refers to a TFS or Azure DevOps Server work item — a bug, task, user story, or PBI named by its numeric id — to an on-prem team project or collection, or asks for a work item's title, state, assignee, description, repro steps, or acceptance criteria. Covers authentication, configuration and its precedence, the workitem get command, the markdown output format, the error categories and the action each one calls for, and the known limitations of the HTML-to-markdown conversion.
---

# tfscli

`tfscli` is a single-binary, read-only client for the TFS / Azure DevOps Server
REST API. It is stateless: there is no daemon and no cache, and every
invocation performs one HTTP request against the server.

The request it issues for a work item is:

```
GET <url>/<collection>/<project>/_apis/wit/workitems/<id>
```

with `api-version=<version>` added to the query only when a version is
configured.

Authentication is a personal access token sent as HTTP Basic with an empty user
name. Output is markdown on stdout; errors are one line on stderr.

## When to use it

- A work item is named by its numeric id and its content is needed — the
  description of a task, the state of a bug, the acceptance criteria of a story.
- The server is an on-premises TFS or Azure DevOps Server instance.

## When not to use it

Apart from `auth login`, which is the user's to run, the tool covers one
command. Do not attempt anything below; none of it exists,
and inventing a flag or a subcommand produces an error, not a result.

- **No writes.** Nothing creates, updates, or comments on a work item. If the
  user asks for a change, report that the tool is read-only.
- **No search and no queries.** There is no WIQL, no filter by title, state, or
  assignee, and no way to list work items. A work item is reached by id only.
- **No batch.** One id per invocation.
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

The server URL and the PAT are stored together, and the token is only ever
sent to the URL stored with it. They come from one of two sources:

- `$XDG_DATA_HOME/tfscli/auth.json` (`~/.local/share/tfscli/auth.json` when
  `XDG_DATA_HOME` is not set, on every OS), written by `tfscli auth login`.
- `TFSCLI_AUTH`, holding the same JSON, `{"url": "…", "pat": "…"}`. When it is
  set, `auth.json` is not read.

There is no flag, separate environment variable, or config key for the URL or
the token. The server is therefore fixed by the user's login; do not try to
point tfscli at another one.

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
| Collection | `collection` | `TFSCLI_COLLECTION` | `--collection` | — | yes |
| Team project | `project` | `TFSCLI_PROJECT` | `-p`, `--project` | — | per command |
| REST API version | `apiVersion` | `TFSCLI_API_VERSION` | `--api-version` | none | no |
| Request logging | — | `TFSCLI_VERBOSE=1` | `--verbose` | off | no |

Two further keys are accepted in the config file only, with no environment
variable and no flag: `caBundle` (path to a PEM bundle appended to the system
root pool, for an internal CA) and `insecureSkipVerify` (disables certificate
verification entirely).

Without a configured API version no `api-version` is sent, and the server
answers at the version it chooses. A configured version is sent unchanged. Do
not set one unless the user asks for it.

The config file is optional. Environment variables and flags alone are enough,
so `--collection` and `-p` can carry everything a call needs.

## The command

```
tfscli workitem get -p <project> <id>
```

`<id>` is a positive integer. Anything else is rejected locally, without a
request.

| Flag | Effect |
|---|---|
| `-p`, `--project` | Team project. Required unless `project` is in the config file or `TFSCLI_PROJECT` is set. |
| `--fields` | Comma-separated TFS reference names. Without it, every field is printed. |
| `--verbose` | One line per request to stderr: method, URL, status, duration. Never headers or bodies. |

Narrow the request whenever the needed fields are known — a full work item can
be large, and its tokens are paid for on every call:

```
tfscli workitem get -p MyProject 12345 --fields System.Title,System.State,System.Description
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
| `not_found` | HTTP 404 — no such work item, project, or collection. | Check the id and the project spelling against what the user gave. Do not scan ids looking for a match. |
| `server` | HTTP 5xx, or a response that could not be parsed. | One retry is reasonable. If it repeats, report the server as unavailable. |
| `config` | A missing or invalid setting, a missing or malformed credential, a bad argument, or a request TFS rejected — a 4xx other than 401, 403, and 404, such as an unknown field name or an unsupported API version. A refused API version names the setting it came from — `--api-version`, `TFSCLI_API_VERSION`, or `apiVersion` in the config file — and the next step. | Fix the invocation if the fault is in it. If the API version was refused and it came from `--api-version` you passed, drop the flag and retry; if it came from the environment or the config file, report the message to the user rather than changing the setting. If a setting is missing, report what is missing; do not write the config file. If the user is not logged in, tell them to run `tfscli auth login`. |
| `network` | The server could not be reached, or the request timed out or was canceled. | Do not repeat the call in a loop. Report the server as unreachable and let the user check the URL and their connection. |

Representative messages:

```
Error [config]: not logged in: no credential at C:\Users\you\.local\share\tfscli\auth.json (run "tfscli auth login", or set TFSCLI_AUTH)
Error [config]: project is not set (pass -p, set TFSCLI_PROJECT, or add "project" to the config file)
Error [config]: work item id "abc" is not a positive integer
Error [config]: The requested REST API version of 7.2 is out of range for this server. The latest REST API version this server supports is 7.1. (api-version "7.2" is set by TFSCLI_API_VERSION; remove it to let the server choose the version, or set one the server supports) (HTTP 400)
Error [auth]: the server did not accept the PAT (it may be invalid, expired, or revoked; if it is valid, IIS Basic Authentication may be enabled on the server, which only an administrator can turn off) (HTTP 401)
Error [network]: cannot reach https://tfs.company.com:8080
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

Every invocation is an HTTP round trip to the server; nothing is cached. Call
the tool once per work item, reuse output already in the conversation instead of
fetching the same work item again, and narrow the request with `--fields` when
the needed fields are known. Do not iterate over a range of ids.
