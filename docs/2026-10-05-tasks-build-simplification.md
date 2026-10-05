# 2026-10-05-tasks-build-simplification.md

**Status:** Active

## Context

The build and release tooling has accumulated parts that no longer pay for themselves.
This work removes them without weakening any check that guards against a known failure.
The items are discussed one at a time; tasks are added to this file as each item is
settled.

Decisions taken for the release notes:

- **`CHANGELOG.md` follows Keep a Changelog 1.1.0**, as now prescribed by the
  `changelog` skill: a preamble naming the format, version headings of the form
  `## [X.Y.Z] - YYYY-MM-DD`, no `---` between sections, a section heading only when it
  has entries (so an empty `[Unreleased]` is the heading alone), the four sections in
  the order Added, Changed, Removed, Fixed, and no version comparison links. The
  separator stripping and the empty-subsection filtering in `tools/` existed only
  because of the old format.
- **Two tools stay, one per operation.** `tools/release-section.go` renames
  `[Unreleased]`; `tools/release-notes.go` extracts a version section. Extracting from
  the rewritten file keeps the cross-check recorded in
  `docs/archive/2026-08-12-tasks-release-button.md`: a malformed rewrite fails before
  anything is pushed.
- **The footer moves to `release.footer` in `.goreleaser.yaml`.** The v0.0.1 footer
  was lost because the `--release-footer` flag is ignored next to `--release-notes`
  (`5147e2a`). The config field is a different mechanism: goreleaser's
  `internal/pipe/release/body.go` adds it to the body regardless of where the notes
  came from. A snapshot renders no release body, so this is confirmed only by the next
  real release, which the user checks.
- **Release notes leave the Makefile entirely.** The `release`, `release-publish`, and
  `release-notes` targets are not used locally. The rationale for routing goreleaser
  through `make` — flags shared between the snapshot and the publish targets — ended
  with `5147e2a`. The release workflow calls the tools and goreleaser directly, and
  goreleaser stops being a local prerequisite.
- **No third-party actions for the changelog.** The two operations are small and
  already implemented; an external action would add a dependency and its author's
  format assumptions without a benefit.

---

### TASK-01 `changelog-format`
**Description:** Convert `CHANGELOG.md` to the format described in the context: add the
preamble, replace the em dash in version headings with a hyphen, remove the `---`
separators and the empty subsection scaffold in `[Unreleased]`, and reorder the
subsections of every release to Added, Changed, Removed, Fixed without changing any
entry. Rewrite `tools/release-section.go` for the new format: it inserts
`## [X.Y.Z] - YYYY-MM-DD` directly below `## [Unreleased]`, keeps the existing
refusals (malformed version or date, a section for the version already present, no
`[Unreleased]`, no entries in it), and no longer writes a scaffold or a separator or
filters subsections. Update the changelog rules in `CONTRIBUTING.md` (section order,
heading format).
**Definition of done:** `make check` passes. `CHANGELOG.md` differs from the previous
version only in format, not in entries. Running `go run tools/release-section.go 9.9.9
<copy> 2026-01-01` on a copy of the changelog with one entry in `[Unreleased]` yields an
empty `[Unreleased]` followed by `## [9.9.9] - 2026-01-01` with that entry, and leaves
the rest of the file byte for byte; running it on a copy with an empty `[Unreleased]`
or with an existing `[9.9.9]` fails and leaves the copy unchanged.
**Status:** Pending

### TASK-02 `release-notes-in-workflow`
**Description:** Simplify `tools/release-notes.go` to extracting the section body from
its heading up to the next `## [` heading, without the footer and without separator
stripping; an absent or empty section stays an error. Move the text of
`docs/release-footer.md` into `release.footer` of `.goreleaser.yaml`, delete the file,
and delete the `before` hook. Remove the `release`, `release-publish`, and
`release-notes` targets and the `VERSION` variable from the `Makefile`. In
`.github/workflows/release.yml`, run `go run tools/release-notes.go "$VERSION"
CHANGELOG.md "$RUNNER_TEMP/release-notes.md"` and print the file in place of
`make release-notes`, and replace the install-only goreleaser step and
`make release-publish` with one `goreleaser/goreleaser-action` step passing
`release --clean --release-notes` with that file. Remove `/docs/release-notes.md` from
`.gitignore`. Update `CONTRIBUTING.md` (quick reference, setup, releasing) and
`CLAUDE.md` (the list of Makefile targets and the release description) to match,
including a one-line note on running `goreleaser release --snapshot --clean` directly
to preview the archives.
**Definition of done:** `make check` passes. No file outside `docs/archive/` refers to
`docs/release-footer.md`, `make release-notes`, `make release`, or
`make release-publish`. Running `go run tools/release-notes.go 0.0.6 CHANGELOG.md
<temp file>` writes exactly the entries of `[0.0.6]`, and a nonexistent version fails.
**Status:** Pending
