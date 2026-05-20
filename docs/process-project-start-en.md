# Project Start

This process runs once — at the beginning of a new project. It produces the foundational documents that all subsequent work depends on. Once complete, these documents go into the repository and are not revisited unless a new feature changes architectural decisions.

---

## Overview

| Step | Conversation | Output |
|------|-------------|--------|
| 1 | Idea, audience, constraints | `project-brief.md` |
| 2 | Technology selection | "Tech Stack" section in `architecture.md` |
| 3 | Architecture design | "Architecture" section in `architecture.md` |
| 4 | Project structure | "Project Structure" section in `architecture.md` + real repository skeleton |

Each step is a separate conversation with AI. Do not combine them — the output of each conversation is the input for the next.

---

## Step 1 — Project Brief

**Input:** your idea

**What to do:**
In conversation with AI, articulate what the product is, who it is for, what problem it solves, and what is explicitly out of scope. AI asks clarifying questions and helps identify weak spots and constraints.

**Output:** `project-brief.md`
- Problem description
- Proposed solution
- Target audience
- Key success metrics
- Constraints (platform, budget, timeline)
- Explicit non-goals — what this product will not do

`project-brief.md` is a historical document. It captures the original intent and does not change after creation.

---

## Step 2 — Tech Stack

**Input:** `project-brief.md`

**What to do:**
Discuss technology options with AI taking into account the requirements, your skills, and constraints. Justify the choice of each tool. Consider alternatives and risks.

**Output:** "Tech Stack" section in `architecture.md`
- Selected technologies with rationale
- Considered alternatives and reasons for rejection
- Known risks

This is the first section of `architecture.md`. The file is created here but not yet complete.

---

## Step 3 — Architecture

**Input:** `project-brief.md` + `architecture.md` (Tech Stack section)

**What to do:**
Design the high-level architecture — modules, their responsibilities, and interactions. Define key dependency interfaces between modules. Describe data flows between components.

**Do not include** API contracts, database schema, or other technical reference details here — those belong in `technical-reference.md` and will be created during feature development.

**Output:** "Architecture" section in `architecture.md`
- Diagrams (Mermaid or ASCII)
- Module descriptions and responsibilities
- Key dependency interfaces between modules
- Data flows between components

**Security checklist:**
- Where is the trust boundary? (what comes from outside, what is generated internally)
- Where are user inputs and external API data validated?
- What goes into logs — are there tokens or sensitive data?
- What data passes between modules — is there unnecessary propagation of secrets?

`architecture.md` changes only when a new feature modifies modules or their interactions. This is rare.

---

## Step 4 — Project Structure

**Input:** `architecture.md`

**What to do:**
Based on the architecture, define the directory and file structure of the project. Then create the actual skeleton in the repository — directories, empty files, base configs.

**Output:**
- "Project Structure" section in `architecture.md` — the principle of code organisation and top-level directories with rationale. No individual files, no subdirectories beyond one level deep. This section changes only when a new feature introduces a new top-level directory.
- Base configs: linter, formatter, `.gitignore`, example config with credentials placeholder, `CHANGELOG.md` (see `process-changelog-guide-en.md` for format)

After this step, `architecture.md` is complete. The repository has a working skeleton and development can begin.

**Security reminder:** add config files with credentials to `.gitignore` on day one — never commit secrets.
