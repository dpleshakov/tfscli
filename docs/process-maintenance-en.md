# Maintenance

Maintenance is not a separate development loop — it is a single living document and a simple rule for how to handle known problems.

---

## tech-debt.md

`tech-debt.md` exists for the entire life of the project. It records known problems that have been consciously deferred — not forgotten, but deliberately not fixed yet.

**What goes here:**
- A known issue with an explicit decision to defer it, with reasoning
- A compromise made consciously during a review task
- Something that requires design or architectural thinking before implementation

**What does not go here:**
- A bug — fix it now or create a feature task for it
- A small cosmetic fix — either fix it immediately or ignore it

**Practical test:** if during a code review you would write "this is an intentional tradeoff, document it" — it goes in `tech-debt.md`. If you would write "fix this" — fix it.

### Format

```markdown
## tech-debt.md

### Active

#### TD-01 `short name`
**Problem:** what is wrong
**Why deferred:** reasoning
**Trigger:** what would make us fix this (new feature, performance issue, etc.)
**Added:** YYYY-MM-DD, in review of YYYY-MM-DD-tasks-{feature}.md

---

### Closed

#### TD-01 `short name`
**Fixed:** YYYY-MM-DD
```

### How it gets populated

During every review task, if a non-critical problem is found and the decision is made to defer it — it is added to `tech-debt.md` in the same commit as the review fixes.

### Acting on tech debt

When a tech debt item is ready to be fixed, it becomes a regular feature: create a `YYYY-MM-DD-tasks-{feature}.md` and run the feature process. Move the item to the Closed section when done.

`tech-debt.md` is never archived — it accumulates a Closed section over time.

---

## Principles for working with AI

These are not process steps — they are habits that make every conversation more effective.

**Context is everything.** AI does not remember past sessions. Documents from previous phases are your shared memory. Pass only the relevant files at the start of each conversation — the active tasks file and the source files needed for the specific task.

**Small tasks beat large ones.** One task = one conversation. "Build the entire backend" produces worse results than "implement the `POST /users` endpoint with validation and database write."

**Never accept code blindly.** Understand what AI generates — otherwise you will be helpless when the first bug appears.

**Commit often.** After each working task — commit. This makes it easy to roll back if something breaks.

**Tests are written with code, not after.** Every task includes tests as part of the definition of done. Ask AI to write tests in the same conversation as the code.

**Security is built in, not added at the end.** Think about it at every stage: what is stored, what is transmitted, what is logged. Specific rules: secrets never in git (`.gitignore` for configs with credentials from day one), input data validated at the system boundary, errors must not expose internal details externally.

**Give AI only the needed context.** Pass only the files relevant to the current task — not the entire project. Extra context does not help; it causes AI to account for unrelated details and produce more diffuse solutions. Rule: if a file is not needed to complete the specific task — do not pass it.
