---
name: tech-debt
description: Record and track consciously deferred problems in tech-debt.md — what qualifies as tech debt, the TD entry format, and closing items as fixed or dropped. Use when deferring a known issue or documenting a deliberate compromise.
argument-hint: "[problem to record | TD-NN to close]"
---

Maintenance is not a separate development loop — it is a single living document
(`tech-debt.md`) and a simple rule for how to handle known problems. The skill makes
no assumptions about the project beyond the presence of a repository.

## Project version first

If the current repository contains `.claude/skills/tech-debt/SKILL.md` and that file is
not the one you are reading, read it and follow it instead of this file.

Where the project's own documentation (for example `CONTRIBUTING.md` or `CLAUDE.md`)
sets a different convention, the project's convention wins over this skill.

## tech-debt.md

`tech-debt.md` exists for the entire life of the project. It records known problems
that have been consciously deferred — not forgotten, but deliberately not fixed yet.

**Location.** If `tech-debt.md` already exists in the repository, use that file and
never create a second one. If it does not exist when the first item is recorded,
create it in the directory where the project keeps its documentation, or in `docs/`
(creating it) if the project has no such directory.

**What goes here:**
- A known issue with an explicit decision to defer it, with reasoning
- A compromise made consciously during a code review
- Something that requires design or architectural thinking before implementation

**What does not go here:**
- A bug — fix it now or plan it as regular work
- A small cosmetic fix — either fix it immediately or ignore it

**Practical test:** if during a code review you would write "this is an intentional
tradeoff, document it" — it goes in `tech-debt.md`. If you would write "fix this" —
fix it.

## Format

```markdown
# Tech debt

## Active

### TD-03 `short name`
**Problem:** what is wrong
**Why deferred:** reasoning
**Trigger:** what would make us fix this (new feature, performance issue, etc.)
**Added:** YYYY-MM-DD, in <where the decision was made>

---

## Closed

### TD-01 `short name`
**Closed:** YYYY-MM-DD — fixed in <commit, pull request, or plan of the work>

### TD-02 `short name`
**Closed:** YYYY-MM-DD — dropped: <why no change is needed>
```

Numbers (`TD-NN`) are assigned sequentially and never reused. The **Added** line
names where the decision was made — a review, a plan of work, or a conversation.

## How it gets populated

During a review, if a non-critical problem is found and the decision is made to defer
it — it is added to `tech-debt.md` in the same commit as the review fixes.

## Closing an item

An item is closed in one of two ways:

- **Fixed.** When the item is ready to be fixed, it becomes a regular piece of work,
  planned and carried out the way the project plans any change. When the work is done,
  the item is closed with `fixed in` and a reference to where the fix can be found.
- **Dropped.** When a review of the item shows that no change is needed — the problem
  does not occur in practice, or it turns out to be a decision rather than a deferred
  problem — the item is closed with `dropped:` and the reason.

Either way, the item moves to the Closed section, keeping its number and short name,
and its body is replaced with the **Closed** line. The original reasoning remains in
the repository history.

`tech-debt.md` is never archived — it accumulates a Closed section over time.
