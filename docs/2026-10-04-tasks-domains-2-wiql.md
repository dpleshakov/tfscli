# 2026-10-04-tasks-domains-2-wiql.md

**Status:** Active

## Context

Second of five planned API domains, numbered in the order they are to be taken.
Today an agent can read a work item only when it is given the ID. WIQL lets it
find work items itself: its active tasks, the children of a feature, bugs in an
area. This serves the success metric in `docs/project-brief.md` ("reference
related work items"). The brief already fixes that the query is passed to the
API as is, with no custom syntax. Available since TFS 2015.

Command names follow the scheme `tfscli <area> <resource> <action>` recorded in
"Follow TFS API structure" in `docs/project-brief.md`.

The resource Wiql (area `wit`) has three operations, named by the rule in
"Follow TFS API structure" in `docs/project-brief.md`:

| Operation | Request | Command |
|---|---|---|
| Query By Wiql | `POST {project}/{team}/_apis/wit/wiql` | `tfscli wit wiql query-by-wiql --query "<WIQL>"` |
| Query By Id | `GET {project}/{team}/_apis/wit/wiql/{id}` | `tfscli wit wiql query-by-id <id>` |
| Get | `HEAD {project}/{team}/_apis/wit/wiql/{id}` | `tfscli wit wiql get <id>` |

This file covers `query-by-wiql` only. `query-by-id` and `get` take the GUID of
a saved query, which can only be obtained through the Queries API; they are
added together with it. For `get` the following is already decided: the
response to HEAD has no body, so on success the command prints one line to
stdout, `# WIQL query <id> (exists)`, stating only what the server reported,
and exits with 0. It needs a `Head` method in `internal/apiclient`, and its
404, which carries no server message, is reported as "query <id> not found, or
the PAT has no access to it (check the query id and the project and team it
belongs to)".

Parameters, mapped by the same rule:

- `project` (path, optional) is `-p` with its usual defaults; without a
  project from any source the request is made at collection level.
- `team` (path, optional) is `--team`. In the path the team follows the
  project, so `--team` without a project is a local `config` error that names
  every source of the project, as the existing missing-project error does:
  `-p`, `TFSCLI_PROJECT`, and `"project"` in the config file.
- `query` (body) is `--query`, required.
- `$top` is `--top` and `timePrecision` is `--time-precision`; neither is sent
  unless given.

The command prints what the server returned and does not read the work items
themselves: WIQL returns IDs only, and reading them is a separate call to
`tfscli wit work-items list --ids ...`. The dependency on the batch get is
therefore logical, not in the code. `list` and `get-batch` accept at most 200
IDs, while a query without `$top` can return far more, so the IDs are read in
portions of at most 200, or the query is limited with `--top` when only the
first results are needed. Output in markdown:

- A heading with the query type and the time of the result:
  `# WIQL query (flat, as of 2026-10-04T10:15:00Z)`.
- `Columns:` followed by the reference names of `columns`, comma-separated, in
  the form `--fields` takes.
- For a flat query, `Work items:` followed by the IDs of `workItems`,
  comma-separated, in the form `--ids` takes.
- For a link query (`workItemRelations`, query type `tree` or `oneHop`),
  `Relations:` followed by one list item per relation in response order: the
  target ID alone when the relation has no source, otherwise
  `source -> target (rel)`.
- When the list is empty, the line reads `Work items: none` or
  `Relations: none`, so that an empty result is not mistaken for truncated
  output.
- The `url` of each element is omitted, as metadata noise.

Example of a link query:

```
# WIQL query (tree, as of 2026-10-04T10:15:00Z)

Columns: System.Id,System.Title
Relations:
- 297
- 297 -> 299 (System.LinkTypes.Hierarchy-Forward)
- 297 -> 300 (System.LinkTypes.Hierarchy-Forward)
```

A syntax error in the query is answered by the server with HTTP 400 and a
message, which the API client already reports in the `config` category with
the server's text.

---

### TASK-01 `wiql-query-by-wiql`
**Description:** Add the domain package `internal/wiql` with `QueryByWiql`: it
requests `POST {project}/{team}/_apis/wit/wiql` with the body
`{"query": ...}` and adds `$top` and `timePrecision` to the query only when
they are given. The path is built at collection level without a project, with
the project, or with the project and the team. The response is parsed into a
result holding the query type, `asOf`, the reference names of `columns`, the
IDs of `workItems`, and the relations of `workItemRelations`, whose source may
be absent. A malformed response is reported in the `server` category, as in
`internal/workitem`.
**Definition of done:** Unit tests cover the path in all three forms,
including a team name with a space, the body and the query for each
combination of parameters, a flat response, a link response with a root that
has no source, empty lists, and a malformed response; `make check` passes.
**Status:** Pending

### TASK-02 `print-wiql`
**Description:** Add a printer to `internal/cli` for the result of
`wiql.QueryByWiql` in the format recorded in Context: the heading, `Columns:`,
then `Work items:` or `Relations:`, with `none` for an empty list. As with
`printWorkItem`, the output is assembled in memory before it is written.
**Definition of done:** Unit tests cover a flat result, a link result, both
empty cases, and the example in Context byte for byte; `make check` passes.
**Status:** Pending

### TASK-03 `cli-wit-wiql-query-by-wiql`
**Description:** Add `tfscli wit wiql query-by-wiql` under the area command
`wit` with `-p`, `--team`, `--query`, `--top`, and `--time-precision`. The
project is optional for this command. The following are local `config`
errors: a missing `--query`, which names the flag and gives an example; the
query given as a positional argument, which says to pass it with `--query`, as
the batch commands do for `--ids`; `--team` without a project, which names
`-p`, `TFSCLI_PROJECT`, and `"project"` in the config file. Help text and examples follow
`tfscli wit work-items get`.
**Definition of done:** Command tests against a test server cover the request
sent with each flag, a flag that is not given not being sent, a request
without a project, the three local errors, a 400 response with a server
message reported in the `config` category, and the printed output;
`make check` passes.
**Status:** Pending

### TASK-04 `docs`
**Description:** Describe the command in `README.md` and
`skills/tfscli/SKILL.md`, the latter showing how the IDs found are read with
`tfscli wit work-items list --ids`, in portions of at most 200, and when to
limit the query with `--top`. Add an entry under `### Added` in
`## [Unreleased]` of `CHANGELOG.md`. Add the `wiql` module, its result type,
and its data flow to `docs/architecture.md`, and the command to "Command
Format" and the API coverage in `docs/project-brief.md`. The sentence saying
that `-p` is required unless a default project is set, in "Command Format" of
`docs/project-brief.md` and in "Command shape" of `CLAUDE.md`, is changed to
state that the project is optional for `tfscli wit wiql query-by-wiql`.
**Definition of done:** The documents describe the command as implemented;
`make check` passes.
**Status:** Pending
