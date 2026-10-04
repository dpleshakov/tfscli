# 2026-10-04-tasks-domains-1-workitem-batch.md

**Status:** Active

## Context

First of five planned API domains, numbered in the order they are to be taken.
Batch get is part of the MVP scope in `docs/project-brief.md` but is not
implemented (see the note in `internal/workitem/workitem.go`). It is a
prerequisite for WIQL, which returns work item IDs only. Available since
TFS 2015.

---

### TASK-01 `workitem-batch-get`
**Description:** Read several work items in one call through the work items
batch endpoint, so that a list of IDs can be read without one request per ID.
To be broken down into atomic tasks before work starts.
**Definition of done:** The task is broken down into atomic tasks in this file.
**Status:** Pending
