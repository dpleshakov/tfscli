# 2026-08-12-tasks-release-button.md

**Status:** Archived

## Context

The release procedure is five manual steps whose order is load-bearing: rename the
`CHANGELOG.md` section by hand in a strict format, commit before tagging, tag, push, and
only then does anything automated happen. Three of the four entries under "When it goes
wrong" in `CONTRIBUTING.md` exist solely because those steps are manual, and the release
workflow never runs `make check`, so a tagged commit is verified by nobody.

This work replaces the tag push with a single entry point: **Actions → Release → Run
workflow → the version number**. The workflow verifies, rewrites the changelog, commits,
tags, pushes, and builds the draft release. The failure modes that come from doing those
things in the wrong order stop being possible rather than being documented.

Decisions taken before the breakdown:

- The `push: tags` trigger is removed, not kept alongside the new one. Two entry points
  into one process is the kind of thing that has to be re-derived later.
- Git writes happen in the workflow, under `GITHUB_TOKEN`. Such a push does not trigger
  other workflows, so the release commit will not start a redundant CI run.
- The release stays a draft. Nothing is public until a human reads the notes and presses
  Publish, which is also where a bad changelog rewrite gets caught.
- Both workflows keep the same shape: install the external tool, then run one `make`
  target. The goreleaser flags move into the `Makefile` so they are written once.
- The changelog rewrite is verified inside the job by running the existing
  `tools/release-notes.go` against the result, before anything is pushed. The extracted
  notes are printed to the job log.

---

### TASK-01 `release-section-tool`
**Description:** Add `tools/release-section.go` (`//go:build ignore`, like its siblings):
`go run tools/release-section.go <version> [changelog] [date]`. It validates the version
against `X.Y.Z` with an optional prerelease suffix, refuses to run if a section for that
version already exists or if `[Unreleased]` is missing or has no entries, renames
`[Unreleased]` to `[X.Y.Z] — YYYY-MM-DD` (em dash, date defaulting to today), drops the
subsections that stayed empty, and inserts a fresh `[Unreleased]` scaffold above it,
separated by `---`. The layout it produces is the one in the `changelog` skill, including
the `---` between every pair of sections.
**Definition of done:** Running the tool against a copy of `CHANGELOG.md` in a scratch
directory produces exactly the expected file for both cases — the first release (no
previous sections) and a subsequent one (previous sections and separators already
present) — and each of the three refusals exits non-zero with a readable message.
**Status:** Done

### TASK-02 `release-notes-separator`
**Description:** In `tools/release-notes.go`, strip a trailing `---` from the extracted
section. From the second release on, a section body runs up to the next `## [` heading and
therefore ends with the separator that belongs to the boundary, which would render as a
stray horizontal rule directly above the footer.
**Definition of done:** Extracting a section that is followed by another one yields the
entries without the trailing separator; extracting the last section is unchanged.
**Status:** In progress

### TASK-03 `makefile-release-publish`
**Description:** Add a `release-publish` target that runs goreleaser for real, and factor
the `--release-notes`/`--release-footer` flags into one variable shared with the existing
snapshot `release` target, so they stop being written twice. Drop the redundant
`release-notes` prerequisite from `release`: the goreleaser before-hook already generates
the file. Update the `.PHONY` list and the header comment.
**Definition of done:** `make release` still builds a snapshot into `dist/` with the
notes and footer in the release body, `make release-publish` differs from it only by the
absence of `--snapshot`, and the two flags appear once in the file.
**Status:** Done

### TASK-04 `release-workflow`
**Description:** Rewrite `.github/workflows/release.yml`: trigger on `workflow_dispatch`
with a required `version` input instead of `push: tags`, guard it with a `concurrency`
group, and run — install Go and the linter, `make check`, `go run
tools/release-section.go`, `make release-notes` as verification with the result printed to
the log, configure the bot identity, commit, tag, push the branch and the tag, install
goreleaser with `install-only`, `make release-publish`. The input is passed to every step
through an `env:` entry, never interpolated into a shell line.
**Definition of done:** The workflow file has one trigger, one job, no `${{ }}` inside a
`run:` body, and every failure before the push step leaves the repository untouched.
`actionlint` (or a YAML parse) reports no error.
**Status:** Done

### TASK-05 `document-the-button`
**Description:** Rewrite the "Releasing" section of `CONTRIBUTING.md` around the new
single entry point: where the version lives, the procedure (run the workflow, publish the
draft), and the failure modes that remain. Add `make release-publish` to the quick
reference table and note that it is what the workflow runs. Update the Makefile target
list in `CLAUDE.md`.
**Definition of done:** The "Releasing" section describes no step that the workflow now
performs, the three eliminated failure modes are gone, and no document still tells the
reader to tag by hand. The section keeps a paragraph describing what the job does — the
reader performs none of it, but a release process nobody can read is the problem this work
set out to solve.
**Status:** Done
