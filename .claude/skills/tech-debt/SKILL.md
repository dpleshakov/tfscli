---
name: tech-debt
description: Record and track consciously deferred problems in tech-debt.md — what qualifies as tech debt, the TD entry format, and closing entries as fixed or dropped. Use when deferring a known issue or documenting a deliberate compromise.
argument-hint: "[problem to record | TD-NN to close]"
---

This skill maintains `tech-debt.md`, a single document that records known problems
deliberately deferred, and defines how entries are added, changed, and closed. It
assumes nothing about the project beyond the presence of a repository.

## Project version first

If the current repository contains `.claude/skills/tech-debt/SKILL.md` and that file is
not the one you are reading, read it and follow it instead of this file.

Where the project's own documentation (for example `CONTRIBUTING.md` or `CLAUDE.md`)
sets a different convention, the project's convention wins over this skill.

## tech-debt.md

`tech-debt.md` exists for the entire life of the project. It records known problems
that have been consciously deferred — not forgotten, but deliberately not fixed yet.

**Location.** If `tech-debt.md` already exists in the repository, use that file and
never create a second one. If it does not exist when the first entry is recorded,
create it in the directory where the project keeps its documentation, or in `docs/`
(creating it) if the project has no such directory.

**What goes here:**
- A known issue with an explicit decision to defer it, with reasoning
- A compromise made consciously during a code review
- A known problem whose fix requires design or architectural thinking first

**What does not go here:**
- A bug with no decision to defer it — fix it now or plan it as regular work
- A small cosmetic fix — either fix it immediately or ignore it
- New work accepted for later that does not fix a problem in what exists — record it
  where the project keeps accepted work that has not started

**Practical test:** if during a code review you would write "this is an intentional
tradeoff, document it" — it goes in `tech-debt.md`. If you would write "fix this" —
fix it.

## Format

```markdown
# Tech debt

## Active

### TD-03 `short-name`
**Problem:** what is wrong
**Why deferred:** reasoning
**Trigger:** the event that makes the fix necessary (a new feature, a performance issue)
**Fix notes:** what is already known about the fix (optional)
**Added:** YYYY-MM-DD, in <where the decision was made>

---

## Closed

### TD-01 `short-name`
**Closed:** YYYY-MM-DD — fixed in <commit, pull request, or plan of the work>

### TD-02 `short-name`
**Closed:** YYYY-MM-DD — dropped: <why no change is needed>
```

Numbers (`TD-NN`) are assigned sequentially and never reused. The **Added** line
names where the decision was made — a review, a plan of work, or a conversation.

**Trigger** names only the event that makes the fix necessary, not what to do when it
occurs. What is already known about the fix — findings, constraints, and the steps
worked out when the decision to defer was made — goes into **Fix notes**, so that it is
not rediscovered later. The field may run to several paragraphs or a list, and is
omitted when nothing is known yet.

## Adding and changing an entry

When a problem is deliberately deferred — during a review, while planning or doing
work, or in a discussion — add an entry under Active with the next free number. Before
writing it, show the entry to the user. The entry goes into the same commit as the
change that led to the decision, such as the review fixes, or into a commit of its own
when there is no such change.

An entry is edited when a later discussion changes what is known: the problem, the
reasoning, the trigger, or the fix notes. It records the current understanding, not
the history of how it was reached.

## Closing an entry

An entry is closed in one of two ways:

- **Fixed.** When the entry is ready to be fixed, it becomes a regular piece of work,
  planned and carried out the way the project plans any change, with the entry's
  **Fix notes** as an input to the planning. The entry is closed in the commit that
  completes the fix, with `fixed in` and a reference to the plan of the work or the
  pull request; a commit is named only when the entry is closed after the fix has been
  committed.
- **Dropped.** When a review of the entry shows that no change is needed — the problem
  does not occur in practice, or it turns out to be a decision rather than a deferred
  problem — the entry is closed with `dropped:` and the reason.

Either way, the entry moves to the Closed section, keeping its number and short name,
and its body is replaced with the **Closed** line. The original reasoning remains in
the repository history.

`tech-debt.md` is never moved or split; closed entries stay in its Closed section.
