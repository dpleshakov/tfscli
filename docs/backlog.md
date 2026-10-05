# Backlog

Accepted work that has not started yet, in priority order.

### `git-pull-requests`
**Goal:** An agent can read pull requests, their comment threads, and the work items
linked to them.
**Context:** The API domains in this backlog are ordered by how much each lets an agent
do without being handed an id by a person. An agent sees the code through the local
`git`, but pull requests, review comments, and linked work items exist only on the
server; the "address the review comments" scenario is impossible without them. The
domain is of no use on TFVC projects. Pull request threads appeared only in TFS 2017;
under "Server compatibility is preserved" in `docs/project-brief.md` they should be
optional, failing with a clear error on an older server while everything else keeps
working.
**Added:** 2026-10-04, in conversation on the next API domains

### `workitem-comments`
**Goal:** An agent can read the discussion of a work item.
**Context:** This extends the existing work item domain rather than adding a new one.
The Comments API exists only on recent servers; on older ones the discussion is kept in
the revisions of the `System.History` field.
**Added:** 2026-10-04, in conversation on the next API domains

### `build-status-and-logs`
**Goal:** An agent can read builds, their timeline, and their logs.
**Context:** Scenario: a CI build failed and the agent is to find out why, locating the
failed step through the timeline and reading its log. Less frequent than the domains
before it, since many on-prem teams run CI outside TFS Build. The volume of logs needs
a decision: the whole log or only the failed step.
**Added:** 2026-10-04, in conversation on the next API domains

### `html-quirks`
**Goal:** Rich-text fields such as Description and Repro Steps are printed without the
TFS-specific markup noise they are stored with.
**Context:** The scope is the `htmlmd` package and golden tests around it; no new API
contracts. Decisions already made:
- The rules are derived from real data: 10–20 raw HTML samples of
  `System.Description`, `Microsoft.VSTS.TCM.ReproSteps`,
  `Microsoft.VSTS.TCM.SystemInfo`, and `Microsoft.VSTS.Common.AcceptanceCriteria`,
  covering @-mentions, attachments, Word- and Outlook-pasted content, and links to
  other work items, stored as fixtures in `internal/htmlmd/testdata/` with a
  `README.md` giving the provenance of each.
- Golden tests first record the current output of `htmlmd.Convert` on the samples,
  with the usual `-update` flag, so that each rule added afterwards shows as a golden
  diff.
- The expected noise classes: mentions rendered as `[@name](#)`, attachment links to
  authenticated URLs, Word-paste leftovers (`<o:p>`, `MsoNormal`, `mso-*` attributes),
  and work item references rendered as long URLs. Known before the samples arrive: the
  library's default rule set has no table support, so `<table>` collapses to its
  concatenated cell text; enabling `plugin/table` belongs to this work.
- Out of scope: anything that needs additional TFS API calls, such as fetching
  attachment file names; that is a separate feature.
- The limitation is stated in "Known limitations" in both `README.md` and
  `skills/tfscli/SKILL.md`, which must stop claiming it together.
**Trigger:** A decision to collect rich-text samples from a live TFS instance. A live
connection is no longer the blocker: release 0.0.5 ran `auth login` and reading one
work item against a live server.
**Added:** 2026-05-20, in the conversation that produced the Architecture section of
`docs/architecture.md`
