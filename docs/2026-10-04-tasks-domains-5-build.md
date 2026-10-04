# 2026-10-04-tasks-domains-5-build.md

**Status:** Active

## Context

Fifth of five planned API domains, numbered in the order they are to be taken.
Scenario: a CI build failed and the agent is to find out why, locating the
failed step through the timeline and reading its log. Less frequent than the
domains before it, since many on-prem teams run CI outside TFS Build. The
volume of logs needs a decision: the whole log or only the failed step.

---

### TASK-01 `build-status-and-logs`
**Description:** Read builds, their timeline, and their logs. To be broken down
into atomic tasks before work starts.
**Definition of done:** The task is broken down into atomic tasks in this file.
**Status:** Pending
