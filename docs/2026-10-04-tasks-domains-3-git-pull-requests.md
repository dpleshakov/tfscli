# 2026-10-04-tasks-domains-3-git-pull-requests.md

**Status:** Active

## Context

Third of five planned API domains, numbered in the order they are to be taken.
An agent sees the code through the local `git`, but pull requests, review
comments, and linked work items exist only on the server; the "address the
review comments" scenario is impossible without them. The domain is of no use
on TFVC projects. Pull request threads appeared only in TFS 2017; under
"Server compatibility is preserved" in `docs/project-brief.md` they should be
optional, failing with a clear error on an older server while everything else
keeps working.

---

### TASK-01 `git-pull-requests`
**Description:** Read pull requests, their comment threads, and linked work
items. To be broken down into atomic tasks before work starts.
**Definition of done:** The task is broken down into atomic tasks in this file.
**Status:** Pending
