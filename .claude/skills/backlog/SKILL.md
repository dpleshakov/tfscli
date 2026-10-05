---
name: backlog
description: Keep accepted work that has not started yet in backlog.md — what belongs there, the entry format, priority order, and how an entry leaves the backlog when the work starts or is dropped. Use when work is accepted for later, or when asked to record, note, or make a task for something that will not start now.
argument-hint: "[work to record | entry to start or drop]"
---

This skill maintains `backlog.md`, a single document that lists work accepted for
implementation but not started yet, and defines how entries are added, ordered, and
taken out. It assumes nothing about the project beyond the presence of a repository.

## Project version first

If the current repository contains `.claude/skills/backlog/SKILL.md` and that file is
not the one you are reading, read it and follow it instead of this file.

Where the project's own documentation (for example `CONTRIBUTING.md` or `CLAUDE.md`)
sets a different convention, the project's convention wins over this skill.

## backlog.md

`backlog.md` records decisions to do a piece of work later, together with what has
already been decided about it. Most of the effort behind an entry is the discussion
that led to it; the entry keeps the outcome of that discussion so that it is not
repeated. An entry is not broken down into tasks: the breakdown is done when the work
starts, when it can draw on the state of the project at that time.

**Location.** If `backlog.md` already exists in the repository, use that file and never
create a second one. If it does not exist when the first entry is recorded, create it
in the directory where the project keeps its documentation, or in `docs/` (creating it)
if the project has no such directory.

**What goes here:**
- Work that has been accepted for implementation and is not starting now
- Work accepted on a condition that has not occurred yet, with that condition recorded
- Work that started and is postponed before it is finished

**What does not go here:**
- Work that starts now — break it down and start it the way the project handles any
  change
- An idea that has not been accepted — discuss it first; record it only once accepted
- A known problem deliberately left unfixed — that is tech debt, recorded where the
  project keeps it

**Practical test:** if the answer to "are we going to do this?" is yes, and to "are we
starting now?" is no, it goes in `backlog.md`.

## Format

```markdown
# Backlog

Accepted work that has not started yet, in priority order.

### `short-name`
**Goal:** what the work achieves for the users of the project
**Context:** decisions, constraints, and open questions already established
**Blocked until:** the event that must occur before the work can start
**Added:** YYYY-MM-DD, in <where the work was accepted>

### `short-name`
**Goal:** ...
**Context:** ...
**Added:** ...
```

**Blocked until** is optional: an entry without it can start at any time. It names a
precondition, not a deadline — the work does not have to start when it occurs.
**Context** may run to several paragraphs or a list; it is the input to planning the
work when it starts.
The **Added** line names where the work was accepted — a review, a plan of work, or a
conversation. Entries have no numbers; the short name identifies an entry and is not
reused while the entry is in the file.

## Order

The order of entries is their priority: the entry nearest the top that is not blocked
is the next to start. A new entry goes where its priority puts it, not at the end by
default; ask the user when the position is not evident.

## Adding and changing an entry

Add an entry when work is accepted for later — in a discussion, a review, or while
planning or doing other work — or when work already started is postponed; then the
entry carries the decisions made so far. Before writing it, show the entry and its
position to the user. The entry goes into the same commit as the change that led to the
decision, or into a commit of its own when there is no such change.

An entry is edited when a later discussion changes what was decided: the goal, the
context, the precondition, or the position. It records the current decision, not the
history of how it was reached.

## Taking an entry out

An entry leaves the backlog in one of two ways:

- **Started.** The work is broken down the way the project breaks down any change, with
  the entry's goal and context as the input. The entry is removed in the same commit
  that creates that breakdown.
- **Dropped.** When the work is no longer wanted, the entry is removed, and the commit
  message states why.

Either way, the entry is deleted rather than marked; its text remains in the repository
history. `backlog.md` stays in place when it becomes empty.
