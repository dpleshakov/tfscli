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
the server. Open: how several work items, and the `null` entries that
`errorPolicy=omit` returns, are rendered in markdown.

---

### TASK-01 `workitem-batch-get`
**Description:** Read several work items in one call through the work items
batch endpoint, so that a list of IDs can be read without one request per ID.
To be broken down into atomic tasks before work starts.
**Definition of done:** The task is broken down into atomic tasks in this file.
**Status:** Pending
