# 2026-10-04-tasks-domains-2-wiql.md

**Status:** Active

## Context

Second of five planned API domains, numbered in the order they are to be taken.
Today an agent can read a work item only when it is given the ID. WIQL lets it
find work items itself: its active tasks, the children of a feature, bugs in an
area. This serves the success metric in `docs/project-brief.md` ("reference
related work items"). The brief already fixes that the query is passed to the
API as is, with no custom syntax. WIQL returns IDs only and depends on the batch
get from `2026-10-04-tasks-domains-1-workitem-batch.md`. Available since
TFS 2015.

---

### TASK-01 `wiql-query`
**Description:** Run a WIQL query against the server and return the matching
work items. To be broken down into atomic tasks before work starts.
**Definition of done:** The task is broken down into atomic tasks in this file.
**Status:** Pending
