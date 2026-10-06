# Backlog

Accepted work that has not started yet, in priority order.

### `workitem-relations`
**Goal:** An agent reading a work item sees the work items linked to it — parent,
children, related, duplicates — without composing a WIQL link query.
**Context:** "Key Success Metrics" in `docs/project-brief.md` includes referencing
related work items. Relations are not fields: the API returns them only with
`$expand`, and the printer renders fields only. Decisions already made:
- `--expand` is exposed on `wit work-items get`, `list`, and `get-batch` at once,
  together with rendering relations and not before it: a flag whose result the
  printer drops would be accepted and silently ignored (recorded in
  `docs/archive/2026-10-04-tasks-domains-1-workitem-batch.md`).
- The flag follows the naming rule in "Follow TFS API structure": `$expand` is
  `--expand`, and its value is passed to the server unchecked.

Open questions:
- How a relation is printed: the link type reference name, the target work item id
  taken from its URL, and the link attributes such as the comment; how relations
  that point to something other than a work item (hyperlinks, attachments, commits)
  are printed.
- The server is expected to refuse `$expand` combined with `fields` (unverified);
  whether tfscli says anything beyond passing the server's error through.
- What the other values of `$expand` (`fields`, `links`, `all`) change in the
  output, if anything.
- The server versions on which `$expand` is available, under "Server compatibility
  is preserved".
**Added:** 2026-10-06, in conversation on the MVP coverage

### `git-pull-requests`
**Goal:** An agent can read pull requests, their comment threads, and the work items
linked to them.
**Context:** The API domains are ordered by how much each lets an agent do without being
handed an id by a person. An agent sees the code through the local `git`, but pull
requests, review comments, and linked work items exist only on the server; the "address
the review comments" scenario is impossible without them. The domain is of no use on
TFVC projects. Pull request threads appeared only in TFS 2017; under "Server
compatibility is preserved" in `docs/project-brief.md` they should be optional, failing
with a clear error on an older server while everything else keeps working.
**Added:** 2026-10-04, in conversation on the next API domains

### `workitem-comments`
**Goal:** An agent can read the discussion of a work item.
**Context:** The API domains are ordered by how much each lets an agent do without being
handed an id by a person. This extends the existing work item domain rather than adding
a new one. The Comments API exists only on recent servers; on older ones the discussion
is kept in the revisions of the `System.History` field.
**Added:** 2026-10-04, in conversation on the next API domains

### `build-status-and-logs`
**Goal:** An agent can read builds, their timeline, and their logs.
**Context:** The API domains are ordered by how much each lets an agent do without being
handed an id by a person. Scenario: a CI build failed and the agent is to find out why,
locating the failed step through the timeline and reading its log. Less frequent than
the domains before it, since many on-prem teams run CI outside TFS Build. The volume of
logs needs a decision: the whole log or only the failed step.
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
**Blocked until:** A decision to collect rich-text samples from a live TFS instance.
**Added:** 2026-05-20, in the conversation that produced the Architecture section of
`docs/architecture.md`
