---
name: tasks
description: Universal task-file workflow — create a structured tasks file for work that starts now and execute it step by step, tracking what is done vs. pending in any project. No project-specific assumptions.
argument-hint: "[feature description to break down | path to a tasks file]"
---

A portable, file-based way to record and track work in any repository. It keeps
"what is done vs. what is left" unambiguous: a single markdown file holds an ordered
list of atomic tasks, each with a plain-text status. The skill requires no particular
tooling or project files and imposes no fixed task categories.

## Project version first

If the current repository contains `.claude/skills/tasks/SKILL.md` and that file is
not the one you are reading, read it and follow it instead of this file.

Where the project's own documentation (for example `CONTRIBUTING.md` or `CLAUDE.md`)
sets a different convention, the project's convention wins over this skill.

## Mode detection

Decide what to do from the user's request and the current context:

- A path to an existing `Active` tasks file is given, or the request names no new work
  and exactly one `Active` file exists in the tasks directory (see "The tasks file") →
  **Execute** mode.
- A feature or request description is given and no matching file exists → **Create**
  mode (breakdown), then offer to start executing. Work that is not to start now does
  not get a tasks file (see "Create mode").
- Ambiguous (e.g. several active files, or unclear whether to create or execute) → ask
  the user one short question before proceeding.

## The tasks file

Use the following format verbatim.

- **Name:** `YYYY-MM-DD-tasks-{slug}.md` — the date prefix orders files and avoids
  name collisions; `{slug}` is a short kebab-case name for the feature/work.
- **Location:** active files live in the *tasks directory*: the directory that already
  holds the project's active tasks files, or the parent of the `archive/` directory
  that holds its archived ones; otherwise the directory where the project keeps its
  documentation; otherwise `docs/`, created if it does not exist. Completed files move
  to the `archive/` subdirectory of the tasks directory.
- **Header status:** `**Status:** Active` while in progress; `**Status:** Archived`
  once every task is resolved — `Done` or `Skipped`.

### Layout

```markdown
# YYYY-MM-DD-tasks-{slug}.md

**Status:** Active

## Context

[Optional, free-form. Shared decisions, constraints, or notes that span several
tasks. Omit the whole section if there is nothing to record.]

---

### TASK-01 `short-name`
**Description:** what to do
**Definition of done:** an observable, checkable outcome
**Status:** Pending

### TASK-02 `short-name`
**Description:** ...
**Definition of done:** ...
**Status:** Pending
```

There are no task types and no cross-references between tasks. Every task has the same
four fields.

### Status values

| Status | Meaning |
|--------|---------|
| `Pending` | Not started. |
| `In progress` | Currently being worked on. |
| `Done` | Definition of done is met. |
| `Skipped — <reason>` | Intentionally not done; reason recorded inline. |

A task is *resolved* when it is `Done` or `Skipped`.

Write the status text verbatim. The only status that carries extra text is
`Skipped`, which appends its reason. Never annotate a `Pending` task — do not mark one
as "next", do not rank candidates, do not add any note to it. The next task to do is
the one already `In progress`, if any, and otherwise the topmost `Pending` one; that is
self-evident and needs no marker.

## Create mode (breakdown)

A tasks file is created when the work starts, not in advance: most of the effort before
that is discussion and decisions, and a breakdown made ahead of time goes stale.

1. Confirm that the work starts now. If it does not — the user wants it recorded,
   planned, or kept for later — do not create a tasks file. Record it where the project
   keeps planned work, if it has such a place; otherwise ask the user where it should
   go. If the work was recorded there earlier, use that record as the input to the
   breakdown and take it out as that place's rules require.
2. Turn the feature/request into a complete, ordered list of **atomic** tasks — each
   one achievable in a single focused session, with a definition of done that can be
   checked.
3. Propose the task structure to the user and let them adjust before writing anything.
4. Write the file to the tasks directory with header `Active` and every task `Pending`,
   numbered `TASK-01`, `TASK-02`, … in order.
5. Offer to start executing the first task.

## Execute mode (tracking discipline)

1. The next task is the one already `In progress`, if any — for example, one left
   unfinished by an interrupted session; otherwise it is the topmost `Pending` one, and
   when you start working on it, set its status to `In progress`. No decision is
   required either way. Do not start or mark the other `Pending` tasks.
2. Do the work for that one task only — do not run ahead into later tasks.
3. When its definition of done is met, set its status to `Done`.
4. **Same-commit rule:** when the task's work is committed, the updated tasks file goes
   into that **same commit** — never a separate status-only commit. This skill follows
   the project's normal commit cadence and does not force commits the user did not ask
   for; it only governs how the file participates when a commit happens.
5. When the last task is resolved: change the header to `Archived` and move the file to
   the `archive/` subdirectory — in the same commit as that task's work.
6. When the work shows that the plan is wrong, change it as described in "Changing
   the plan".

## Changing the plan

The `Pending` tasks are a plan, not a record, and the plan changes when the work shows
it is wrong: a task turns out to be needed, a description contains a mistake, two tasks
contradict each other, or a task is redundant. A task whose outcome is a decision often
determines the tasks that follow it; those are added once the decision is made.

- `Pending` tasks may be added, edited, reordered, or removed, and the Context section
  may be updated. Propose every such change to the user before writing it, as when
  creating the file.
- `Done` tasks are not rewritten: they record what was done, and commits refer to them.
- A task that was a mistake in the plan — a duplicate, a contradiction, a task not
  needed — is removed. A task that is deliberately not done, where the reason is worth
  knowing later, is marked `Skipped — <reason>` instead.
- Task numbers identify tasks and do not express order. A number is never changed or
  reused; a new task takes the next unused number and is placed where it is to be
  done. The order of execution is the order in the file.

## Invariants

- Tasks are atomic; work on one at a time.
- Never leave a finished task unmarked — the file is the source of truth for progress.
- `Done` tasks are never rewritten; `Pending` tasks change when the plan proves wrong.
- The tasks file travels with the work it describes in a single commit.
- Archive the file (header + move to `archive/`) once every task is resolved.
- Require no particular project files or tooling; follow the project's documented
  conventions where they exist.
