# 2026-10-04-tasks-domains-2-wiql.md

**Status:** Active

## Context

Second of five planned API domains, numbered in the order they are to be taken.
Today an agent can read a work item only when it is given the ID. WIQL lets it
find work items itself: its active tasks, the children of a feature, bugs in an
area. This serves the success metric in `docs/project-brief.md` ("reference
related work items"). The brief already fixes that the query is passed to the
API as is, with no custom syntax. Available since TFS 2015.

Command names follow the scheme `tfscli <area> <resource> <action>` introduced
by `2026-10-04-tasks-command-areas.md`, which is to be done before this file.

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
the PAT has no access to it".

Parameters, mapped by the same rule:

- `project` (path, optional) is `-p` with its usual defaults; without a
  project from any source the request is made at collection level.
- `team` (path, optional) is `--team`. In the path the team follows the
  project, so `--team` without a project is a local `config` error that names
  `-p` and `TFSCLI_PROJECT`.
- `query` (body) is `--query`, required.
- `$top` is `--top` and `timePrecision` is `--time-precision`; neither is sent
  unless given.

The command prints what the server returned and does not read the work items
themselves: WIQL returns IDs only, and reading them is a separate call to
`tfscli wit work-items list --ids ...`. The dependency on the batch get is
therefore logical, not in the code. Output in markdown:

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

### TASK-01 `wiql-query`
**Description:** Run a WIQL query against the server and return the matching
work items. To be broken down into atomic tasks before work starts.
**Definition of done:** The task is broken down into atomic tasks in this file.
**Status:** Pending
