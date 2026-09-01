# 2026-09-01-tasks-agent-skill.md

**Status:** Active

## Context

tfscli is an AI-first tool, but the release archive ships only the binary and
human documentation (`LICENSE`, `README.md`, `CHANGELOG.md`,
`config.example.json`). An agent that receives the archive learns nothing about
the tool unless a human explains it, and has to discover behaviour through
`--help` — which does not cover what matters most: what to do with each error
category, where the boundaries are (read-only, one command), which output
defects are known, and why the tool must not be called in a loop.

The work adds one artifact: `skills/tfscli/SKILL.md`, in the Agent Skills
format. opencode has first-party skill support and searches
`~/.claude/skills/<name>/SKILL.md` and `.claude/skills/<name>/SKILL.md` among
its discovery paths, so a single file covers both Claude Code and opencode with
no duplication.

Decisions taken during planning, recorded so they are not re-litigated:

- One artifact only. No `AGENTS.md` fragment, no Claude Code plugin, no MCP
  wrapper (a stated non-goal of the brief).
- No opencode `.ts` custom tool: it is single-vendor, mirrors the CLI surface in
  a second source of truth, and needs the `@opencode-ai/plugin` npm dependency.
  Revisit when `--json` lands and a wrapper could return structured data.
- Frontmatter carries `name` and `description` only. opencode recognises `name`,
  `description`, `license`, `compatibility`, and `metadata`; everything else is
  ignored.
- The content is self-contained in one file: Claude Code can load sibling files
  from a skill directory, the opencode documentation does not say it can.
- Staleness is handled by a release-checklist item, not by a verifying test.

The discovery paths documented here come from the Claude Code and opencode
documentation, not from a run against either agent. TASK-06 confirms them;
until it closes, `README.md` and `CHANGELOG.md` state as fact something that
has only been read. That is an accepted risk of shipping the skill before an
agent has loaded it.

---

### TASK-01 `skill-file`
**Description:** Write `skills/tfscli/SKILL.md` — frontmatter (`name: tfscli`, a
`description` carrying the trigger words: TFS, Azure DevOps Server, work item,
task or bug id, on-premises) and nine sections: when to use it and when not to;
readiness check; configuration with precedence; the `workitem get` command;
output format with an example; error categories with the action each one calls
for; known limitations; PAT handling; the cost of a call.
**Definition of done:** The file exists with all nine sections, and every flag,
error category, and environment variable it mentions is verified against
`internal/cli`, `internal/tfserr`, and `internal/config`.
**Status:** Done

### TASK-02 `release-archive`
**Description:** Add `skills` to `archives[0].files` in `.goreleaser.yaml`,
beside `LICENSE`, `README.md`, `CHANGELOG.md`, and `config.example.json`.
**Definition of done:** `make release` builds a local snapshot and an unpacked
archive from `dist/` contains `skills/tfscli/SKILL.md` at that path.
**Status:** Done

### TASK-03 `readme`
**Description:** Add an "install for an AI agent" section to `README.md` and
update the list of files each archive holds.
**Definition of done:** The section describes copying the directory into
`~/.claude/skills/`, lists the other opencode discovery paths, and notes that
opencode gates skills through `permission.skill` in `opencode.json` and can
disable them with `tools.skill: false`. The archive contents list mentions
`skills/`.
**Status:** Done

### TASK-04 `release-checklist`
**Description:** Add an item to the release procedure in `CONTRIBUTING.md`:
reconcile `skills/tfscli/SKILL.md` with the current command tree, flags,
`tfserr` categories, and `config` environment variables.
**Definition of done:** The item is in the checklist and names what to check.
**Status:** Done

### TASK-05 `changelog`
**Description:** Record the change in `CHANGELOG.md` through the `changelog`
skill, section `Added`.
**Definition of done:** An entry in the right section, in the format the skill
prescribes.
**Status:** Done

### TASK-06 `live-check`
**Description:** Verify the skill is discovered from `~/.claude/skills/tfscli/`
by both Claude Code and opencode.
**Definition of done:** In both agents the skill is advertised and opens on a
request about a TFS work item.
**Status:** Pending
