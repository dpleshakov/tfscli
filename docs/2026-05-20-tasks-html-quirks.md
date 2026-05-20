## 2026-05-20-tasks-html-quirks.md

**Status:** Active
**Trigger:** First end-to-end connection to a live TFS instance with real work items containing rich-text fields. Until then the work in this file is deliberately deferred — see the "HTML quirks" decision in the conversation that produced the Architecture section of `architecture.md`.

### Contracts

No new API contracts. Scope is limited to the `htmlmd` module and golden tests around it.

### Note on task types

Per `docs/process-feature-en.md`, a tasks file normally contains all four task types (Regular, Smoke, Review, Docs). This file deliberately omits the Smoke type: there is no running code to smoke-test — the feature is fixtures (`testdata/`) plus golden tests plus library-plugin rules. TASK-01 already requires a live-TFS round-trip to harvest the fixtures, which is the only end-to-end interaction the feature can produce; the golden tests in TASK-02 are the verification mechanism instead of a smoke check.

---

### TASK-01 `collect-html-samples`
**Type:** Regular
**Description:** Connect to a live TFS instance and dump the raw HTML of 10–20 work item rich-text fields, with coverage across `System.Description`, `Microsoft.VSTS.TCM.ReproSteps`, `Microsoft.VSTS.TCM.SystemInfo`, and `Microsoft.VSTS.Common.AcceptanceCriteria`. Aim for variety: items with @-mentions, items with attachments, items with Word- or Outlook-pasted content, items linking to other work items. Store the samples as fixtures under `internal/htmlmd/testdata/` (one file per sample, `.html` extension). Add a sibling `README.md` enumerating each file with one-line provenance — work item type and which quirks it exhibits.
**Definition of done:** ≥10 sample HTML files committed under `internal/htmlmd/testdata/`, with a `README.md` enumerating them.
**Status:** ⬜ Pending

### TASK-02 `golden-tests-baseline`
**Type:** Regular
**Description:** For each sample collected in TASK-01, add a golden test that runs the current `htmlmd.Convert` over it and records the output as `<sample>.expected.md` next to the input. Goal here is to capture today's behaviour, not to improve it — golden tests will signal when changes in TASK-03 take effect. Standard `-update` flag pattern to regenerate expectations.
**Definition of done:** all golden tests pass; running with `-update` regenerates expected files.
**Status:** ⬜ Pending

### TASK-03 `identify-and-fix-noise`
**Type:** Regular
**Description:** Inspect the golden outputs from TASK-02. Classify noise: mentions rendering as `[@name](#)`, attachment links pointing to authenticated URLs, Word-paste leftovers (`<o:p>`, `MsoNormal`, mso-* attributes), work-item references rendering as long URLs. For each chosen class, add a rule via the library's plugin API. Update golden expectations in the same commit. Out of scope: anything that requires additional TFS API calls (e.g. fetching attachment filenames from the attachments endpoint) — that becomes a separate feature.
**Definition of done:** the list of noise classes addressed is recorded both in the commit message and in this tasks file (append a short bullet list under this DoD when the task completes); each rule has a corresponding golden diff demonstrating its effect.
**Status:** ⬜ Pending

### TASK-04 `review`
**Type:** Review
**Covers:** TASK-01, TASK-02, TASK-03
**Status:** ⬜ Pending

### TASK-05 `docs`
**Type:** Docs
**Description:** Update `README.md` and `CHANGELOG.md` with a one-line user-facing note that HTML rendering now handles the relevant TFS patterns (mentions, attachments, Word paste, work-item refs — whichever were addressed). No technical detail in `CHANGELOG.md`; see `docs/process-changelog-guide-en.md`.
**Status:** ⬜ Pending
