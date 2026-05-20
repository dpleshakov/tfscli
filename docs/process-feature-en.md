# Feature Development

This process repeats for every feature — including the initial MVP features. There is no special MVP mode: the first features are developed exactly the same way as all subsequent ones.

---

## Overview

| Step | Conversation | Output |
|------|-------------|--------|
| 1 | Task breakdown | `YYYY-MM-DD-tasks-{feature}.md` |
| 2–N | One conversation per task | Working committed code + updated `technical-reference.md` |

---

## Step 1 — Task Breakdown

**Input:** `architecture.md` + `technical-reference.md` (if it exists)

If `technical-reference.md` does not yet exist, it is created during this conversation. It contains API endpoints, database schema, and any other technical details that need to be stable before tasks can be written.

**What to do:**
In conversation with AI, break the feature into tasks. AI proposes the task structure, you agree or adjust. The goal is a complete, ordered list where every task is atomic — achievable in one AI conversation (roughly 30–100 lines of code).

One conversation may produce more than one tasks file if the feature naturally splits into independent parts. This is preferable to forcing everything into one file.

**Output:** `YYYY-MM-DD-tasks-{feature}.md`

The date prefix keeps files ordered and prevents name collisions. Example: `2026-03-04-tasks-auth.md`.

### Task types

Every tasks file contains four types of tasks. AI places them during breakdown — you confirm their placement.

**Regular tasks** — implementation: code + tests. Each ends with a commit.

**Smoke test tasks** — placed at natural checkpoints before continuing (e.g. after backend, before frontend). Goal: verify the mechanism works end-to-end with minimal effort — logs, a primitive HTML page, a curl call. Not a full test, just a confidence check.

**Review tasks** — placed after each logical block (e.g. after all files in `auth/` are done). Each review task covers:
- Code: security, error handling, readability, obvious performance issues
- Security checklist: input validation, no tokens in logs, errors do not expose internal details, dependency vulnerability check (`npm audit`, `pip audit`, `govulncheck`, or equivalent for your stack)
- Documentation: does `technical-reference.md` reflect what was actually built? Update if not. Does `architecture.md` need updating?

**Docs task** — always the last task in the file. Covers:
- User-facing documentation (README, help, guides)
- `CHANGELOG.md` — only changes visible to the user; no technical details, no refactoring, no infrastructure changes (see `process-changelog-guide-en.md`)

### Tasks file structure

```markdown
## YYYY-MM-DD-tasks-{feature}.md

**Status:** Active

### Contracts
[API endpoints and DB schema planned for this feature — filled during breakdown]

---

### TASK-01 `name`
**Type:** Regular
**Description:** ...
**Definition of done:** working code + tests + committed
**Status:** ⬜ Pending

### TASK-02 `name`
**Type:** Smoke test
**Description:** ...
**Status:** ⬜ Pending

### TASK-03 `name`
**Type:** Review
**Covers:** TASK-01, TASK-02
**Status:** ⬜ Pending

...

### TASK-NN `docs`
**Type:** Docs
**Status:** ⬜ Pending
```

---

## Step 2–N — Task Execution

Each task is a separate conversation with AI.

**Input for every conversation:** the specific task + relevant source files (only what is needed for this task — not the entire project)

### Regular task

1. Give AI the task and relevant context
2. Receive code — understand it, do not accept blindly
3. Write tests in the same conversation
4. Run code and tests, test manually
5. If something is wrong — fix in the same conversation
6. Commit working code with tests; update task status to `✅ Done` in the same commit

### Smoke test task

Build the minimal thing that demonstrates the mechanism works. This is not a deliverable — it is a checkpoint. Log output, a primitive page, a curl call — whatever is fastest. Commit if useful, discard if not.

### Review task

Review the logical block that preceded this task. Work through the checklist:
- Security: input validation, no secrets in logs or responses, dependency audit
- Error handling: no silently ignored errors, no internal details in HTTP responses
- Readability: nothing obviously confusing
- Documentation: update `technical-reference.md` and `architecture.md` if they do not match reality

Commit any fixes and documentation updates.

### Docs task

Write or update user-facing documentation. Update `CHANGELOG.md` with only what the user can see and use — not technical details.

---

## Task status values

| Symbol | Meaning |
|--------|---------|
| ⬜ | Pending |
| 🔄 | In progress |
| ✅ | Done |
| ⏭️ | Skipped — reason noted |

When all tasks are done, update the file header status to `Archived` and move the file to `docs/archive/`.
