# Backlog

Accepted work that has not started yet, in priority order.

### `api-version-negotiation`
**Goal:** Every command works on any server without the user setting an API
version: `wit work-items get-batch` and `wit wiql query-by-wiql` no longer fail
with "No api-version was supplied" on servers that require a version for POST.
**Context:** Microsoft documents `api-version` as required on every request; in
practice GET requests work without it, while POST requests are refused by some
servers. The official SDKs (.NET, Go, Node, Python) discover versions with
`OPTIONS {url}/{collection}/_apis`, which lists every resource with its
`minVersion`, `maxVersion`, `releasedVersion`, and `resourceVersion`. Decisions
already made:
- When no version is configured, every request except the verification in
  `auth login` is preceded by `OPTIONS {url}/{collection}/_apis` and carries the
  `releasedVersion` of its resource — the version the server chooses for a
  request without one, so the output does not change. A resource without a
  released version (`0.0`) gets `{maxVersion}-preview.{resourceVersion}`.
- The response is not cached: an on-disk cache is excluded by the product
  constraints, and an in-process cache gains nothing in a CLI. Each invocation
  makes one extra request.
- A configured version (config file, `TFSCLI_API_VERSION`, `--api-version`) is
  sent as is, without `OPTIONS`.
- Negotiation never fails a command on its own. A 401 or a transport error on
  `OPTIONS` is reported at once in its usual category, since the main request
  would fail the same way. Any other failure — 400, 403, 404, 405, 5xx, an
  unparseable response, the resource missing from the list — falls back to
  sending the request without a version, as before, so that no server that works
  today stops working.
- Under `--verbose` the `OPTIONS` request is logged like any other, and a
  fallback adds one line saying that the version was not negotiated and why.
  Nothing is printed without `--verbose`.
- When a request sent without a version after a fallback is refused for lack of
  one, the `config` error says that negotiation failed and why, names
  `apiVersion` with the config file path, `TFSCLI_API_VERSION`, and
  `--api-version`, and says that the value must not exceed the server's version.
  This supersedes TASK-08 in `docs/2026-10-04-tasks-first-run-errors.md`, which
  moves into this work when it starts.
- No new error category.
- The constraint "requests carry `api-version` only when one is configured" in
  `docs/project-brief.md` and `CLAUDE.md`, and the "REST API version" section of
  `README.md`, change together with this work.

Open questions:
- Whether `OPTIONS` answers in this form on TFS 2015 and 2017 (unverified).
- How a request is matched to its entry in the list: by location id, as the
  SDKs do, or by area and resource name.
**Added:** 2026-10-06, in conversation on API version negotiation

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
