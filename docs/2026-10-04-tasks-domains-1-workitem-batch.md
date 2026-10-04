# 2026-10-04-tasks-domains-1-workitem-batch.md

**Status:** Active

## Context

First of five planned API domains, numbered in the order they are to be taken.
Batch get is part of the MVP scope in `docs/project-brief.md` but is not
implemented (see the note in `internal/workitem/workitem.go`). It is a
prerequisite for WIQL, which returns work item IDs only.

The API offers two operations, and both are exposed, named by the rule in
"Follow TFS API structure" in `docs/project-brief.md`:

- `workitem list --ids …` for "Work Items - List",
  `GET _apis/wit/workitems?ids=…`, documented since 4.1 (TFS 2018).
- `workitem get-batch --ids …` for "Get Work Items Batch",
  `POST _apis/wit/workitemsbatch`, since 5.0 (Azure DevOps Server 2019). It
  exists for requests whose query string would exceed the server's limit
  (IIS defaults to 2048 bytes).

Both take `ids` (at most 200), `fields`, `asOf`, `$expand`, and
`errorPolicy`, passed through as flags; tfscli adds no behaviour of its own,
so the 200-ID limit, conflicting parameters, and missing IDs are reported by
the server. As in `workitem get`, the IDs are checked locally to be positive
integers before a request is made.

`$expand` is not exposed in this file. What it adds — relations and links —
is outside `fields`, which is all the printer renders, so the flag would be
accepted and its result silently dropped. It arrives together with rendering
relations, for `workitem get` as well.

Output in markdown:

- Each work item is printed exactly as `workitem get` prints it, one after
  another, separated by a blank line, with no enclosing heading or separator:
  the `# Work item N (rev R)` heading already delimits them, and a work item
  looks the same whichever command printed it. A `---` separator is not used,
  because the HTML converter produces the same line from `<hr>`.
- Work items are printed in the order the server returns them, which need not
  be the order of `--ids`.
- With `--error-policy omit` the server puts `null` in place of a work item
  that does not exist or that the PAT cannot read, and does not say which.
  Since the order of the response is not guaranteed, the missing IDs are
  found as the requested IDs less the returned ones, and each is printed after
  the returned work items as a heading line with no body:
  `# Work item 298 (not returned: it does not exist, or the PAT has no access to it)`.
  The note sits in the parentheses of the heading, where `rev` already marks
  metadata, so that it cannot be read as work item content. The exit code is
  0, since omitting was requested. Without the flag the server's default,
  `fail`, applies, and a missing work item is reported through the normal
  error path.

---


### TASK-01 `workitem-list`
**Description:** Add `workitem.List` to `internal/workitem`: it requests
`GET {project}/_apis/wit/workitems` with `ids` and, when given, `fields`,
`asOf`, and `errorPolicy`, and parses the `value` array of the response,
reusing the parsing of a single work item for each element. The result holds
the returned work items in response order and the missing IDs: the requested
IDs, without duplicates and in request order, less the IDs of the returned
work items.
**Definition of done:** Unit tests cover the query sent for each combination
of parameters, a response with every work item returned, a response with
`null` entries yielding the missing IDs, a response in an order different from
the request, and a malformed response reported in the `server` category;
`make check` passes.
**Status:** Done

### TASK-02 `apiclient-post`
**Description:** Add `Post` to `internal/apiclient` for a request with a JSON
body, sharing with `Get` the URL building, the `api-version` handling, the
authentication, and the classification of responses and transport errors, and
add it to the `APIClient` interface of `internal/workitem`, updating the note
that says it is absent.
**Definition of done:** Unit tests cover the method, the `Content-Type`
header, the body sent, `api-version` present and absent, and the error
classification on a non-2xx response; `make check` passes.
**Status:** Pending

### TASK-03 `workitem-get-batch`
**Description:** Add `workitem.GetBatch` to `internal/workitem`: it requests
`POST {project}/_apis/wit/workitemsbatch` with a body carrying `ids` and,
when given, `fields`, `asOf`, and `errorPolicy`, and returns the same result
as `workitem.List`, sharing its response parsing.
**Definition of done:** Unit tests cover the body sent for each combination of
parameters, omitted parameters being absent from the body rather than empty,
and the same response cases as TASK-01; `make check` passes.
**Status:** Pending

### TASK-04 `print-work-items`
**Description:** Add a printer to `internal/cli` for the result of
`workitem.List` and `workitem.GetBatch`: each returned work item through
`printWorkItem`, separated by a blank line, followed by one heading line per
missing ID in the form recorded in Context. As with `printWorkItem`, the
output is assembled in memory before it is written.
**Definition of done:** Unit tests cover several work items, missing IDs
after the returned ones, a result with only missing IDs, and output identical
to `printWorkItem` for a single work item; `make check` passes.
**Status:** Pending

### TASK-05 `cli-workitem-list`
**Description:** Add `tfscli workitem list` with `-p`, `--ids` (required,
comma-separated, each checked like the ID of `workitem get`), `--fields`,
`--as-of`, and `--error-policy`, the last two passed through without local
validation. Help text and examples follow `workitem get`. A missing `--ids`
is a `config` error that names the flag and gives an example.
**Definition of done:** Command tests against a test server cover the request
sent with each flag, a flag that is not given not being sent, a missing or
invalid `--ids`, a missing project, and the printed output; `make check`
passes.
**Status:** Pending

### TASK-06 `cli-workitem-get-batch`
**Description:** Add `tfscli workitem get-batch` with the same flags and
checks as `workitem list`, calling `workitem.GetBatch`. The help text says
that the command needs Azure DevOps Server 2019 or later and exists for
requests too long for `workitem list`.
**Definition of done:** Command tests cover the same cases as TASK-05 for the
POST request; `make check` passes.
**Status:** Pending

### TASK-07 `docs`
**Description:** Describe both commands in `README.md` and
`skills/tfscli/SKILL.md`, add an entry to `CHANGELOG.md` under
`## [Unreleased]`, and update `docs/architecture.md` where it describes the
API client and the work item module as GET-only or single-item.
**Definition of done:** The documents describe the commands as implemented;
`make check` passes.
**Status:** Pending
