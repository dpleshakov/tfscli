# 2026-10-04-tasks-command-areas.md

**Status:** Active

## Context

Commands are currently named `tfscli <resource> <action>`, where the resource
is taken from the REST API and the API area it belongs to is dropped:
`workitem get` for "Get Work Item" in the area `wit`. This holds only while
resource names are unique across areas, which the API does not guarantee: the
resource Items exists in both `git` and `tfvc`, and Comments in `wit` sits next
to Pull Request Thread Comments in `git`. The first such collision would force
either an invented name, which the naming rule forbids, or an area added to
some commands only.

Commands are therefore named `tfscli <area> <resource> <action>`:

- The area and the resource are the segments of the operation's page path in
  the REST API reference, which the reference already gives in kebab-case:
  `.../rest/api/azure/devops/wit/work-items/get-work-item` is
  `tfscli wit work-items get`, `.../wit/wiql/query-by-wiql` is
  `tfscli wit wiql ...`, `.../git/pull-requests/...` is
  `tfscli git pull-requests ...`. The resource is thus `work-items`, not
  `workitem`.
- The action is derived from the operation name as before.
- `tfscli auth login` is not a REST API operation but a local command of
  tfscli, and stays outside the scheme; the brief states this explicitly so
  that the rule keeps having no exceptions.
- There are no users yet, so the old commands `workitem get`, `workitem list`,
  and `workitem get-batch` are removed without aliases.
- The Go package `internal/workitem` keeps its name: package boundaries follow
  the architecture modules, not the command names.

This file is to be done before `2026-10-04-tasks-domains-2-wiql.md`.

---

### TASK-01 `brief-command-rule`
**Description:** Rewrite the "Command Format" section and the naming rule in
"Follow TFS API structure" in `docs/project-brief.md` for the scheme recorded
in Context: the area and the resource from the reference page path, the action
by the existing rule, `auth login` outside the scheme. Update the command
examples there and in `CLAUDE.md`.
**Definition of done:** `docs/project-brief.md` and `CLAUDE.md` describe the
new scheme and use the new command names; `make check` passes.
**Status:** Pending

### TASK-02 `cli-areas`
**Description:** In `internal/cli`, add the area command `wit` with the
resource command `work-items` under it, and move `get`, `list`, and
`get-batch` there. Update the help texts, the examples, and every error
message that names a command, such as the `bareNotFound` message of
`get-batch`.
**Definition of done:** The command tests run against
`tfscli wit work-items ...`; a test checks that `tfscli workitem get` is
reported as an unknown command; `make check` passes.
**Status:** Pending

### TASK-03 `docs`
**Description:** Update `README.md`, `skills/tfscli/SKILL.md`, and
`docs/architecture.md` for the new command names, add an entry under
`### Changed` in `## [Unreleased]` of `CHANGELOG.md`, and replace the old
command names in the active tasks files
`2026-10-04-tasks-first-run-errors.md`, `2026-05-20-tasks-html-quirks.md`,
and `2026-10-04-tasks-domains-2-wiql.md`.
**Definition of done:** No document outside `docs/archive/` names the old
commands; `make check` passes.
**Status:** Pending
