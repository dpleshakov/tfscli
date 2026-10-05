# 2026-10-05-tasks-build-simplification.md

**Status:** Archived

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

Decisions taken for `make clean`:

- **`git clean` replaces `tools/rm.go`.** Once the release targets leave the Makefile,
  the only local artifacts are `tfscli`, `tfscli.exe`, and `coverage.out`.
  `git clean -fX -- <paths>` removes them on every platform: `-X` limits it to ignored
  files and the pathspec to exactly these paths, so `config.json`, IDE settings, and
  untracked work are never touched. Git is already required everywhere, so the rule that
  no recipe assumes a Unix shell still holds. The target is kept rather than dropped, so
  that nobody falls back to an unrestricted `git clean -fdX`.

Decisions taken for the module files check:

- **`go mod tidy -diff` replaces `go mod tidy` followed by `git diff --exit-code`.** The
  final line of `check` existed only because `lint` rewrote `go.mod` and `go.sum`.
  `-diff` (Go 1.23 and later) prints the required changes and fails without writing
  anything, so the check lives in one target, a red run leaves the working tree as it
  was, and `make lint` alone catches the drift. Fixing it becomes a deliberate
  `go mod tidy` by hand. The intent from `a043988` — module files that drift from the
  imports fail the check — is unchanged.

Considered and left as is:

- **Tests and vet run twice in `check`.** `build` runs `go vet` and `go test ./...`,
  then `lint` runs `govet` and `test` runs the same tests with coverage — about four
  seconds of duplication in CI. Removing it would not remove a file or a mechanism and
  would make `check` stop being a superset of `build`, so `check` keeps depending on
  `build`.
- **The `push: main` trigger in `ci.yml`.** While `main` has no server-side
  protection, the push run is the only check of `main` after a change reaches it by any
  path, so the trigger stays. Its removal belongs to TD-07 in `docs/tech-debt.md`, once
  a ruleset requires branches to be up to date.

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
**Status:** Done

### TASK-02 `release-notes-in-workflow`
**Description:** Simplify `tools/release-notes.go` to extracting the section body from
its heading up to the next `## [` heading, written to an output file or, without one,
to standard output, without the footer and without separator
stripping; an absent or empty section stays an error. Move the text of
`docs/release-footer.md` into `release.footer` of `.goreleaser.yaml`, delete the file,
and delete the `before` hook. Remove the `release`, `release-publish`, and
`release-notes` targets and the `VERSION` variable from the `Makefile`. In
`.github/workflows/release.yml`, run `go run tools/release-notes.go "$VERSION"
CHANGELOG.md "$RUNNER_TEMP/release-notes.md"` and print the file in place of
`make release-notes`, and replace the install-only goreleaser step and
`make release-publish` with one `goreleaser/goreleaser-action` step passing
`release --clean --release-notes` with that file. In `.gitignore`, remove
`/docs/release-notes.md` and the unused `/bin/`, and correct the comment on
`config.json`, which no longer holds credentials — they live in `auth.json`. Update `CONTRIBUTING.md` (quick reference, setup, releasing) and
`CLAUDE.md` (the list of Makefile targets and the release description) to match,
including a one-line note on running `goreleaser release --snapshot --clean` directly
to preview the archives.
**Definition of done:** `make check` passes. No file outside `docs/archive/` refers to
`docs/release-footer.md`, `make release-notes`, `make release`, or
`make release-publish`. Running `go run tools/release-notes.go 0.0.6 CHANGELOG.md
<temp file>` writes exactly the entries of `[0.0.6]`, and a nonexistent version fails.
**Status:** Done

### TASK-03 `clean-without-rm-tool`
**Description:** Make the `clean` target run `git clean -fX -- tfscli tfscli.exe
coverage.out` and delete `tools/rm.go`. Update every description of `clean` and of the
`tools/` helpers to match: the comments in the `Makefile`, the quick reference and the
"No recipe may assume a Unix shell" rule in `CONTRIBUTING.md`, and the build section of
`CLAUDE.md`, which no longer name file removal among the jobs of `tools/`.
**Definition of done:** `make check` passes. With `tfscli.exe`, `coverage.out`, and a
`config.json` present in the repository root, `make clean` removes the first two and
leaves `config.json`; run again with nothing to remove, it succeeds. No file outside
`docs/archive/` refers to `tools/rm.go`.
**Status:** Done

### TASK-04 `tidy-diff`
**Description:** In the `Makefile`, make `lint` run `go mod tidy -diff` instead of
`go mod tidy`, and remove `git diff --exit-code go.mod go.sum` from `check` together
with the comments that explain it. Update the descriptions of `lint` and `check` in
`CONTRIBUTING.md` (quick reference) and anywhere else they mention that `lint` runs
`go mod tidy` or that `check` compares the module files afterwards.
**Definition of done:** `make check` passes. With a requirement removed from `go.mod`
by hand, `make lint` fails, prints the diff, and leaves `go.mod` and `go.sum` exactly as
they were. No file outside `docs/archive/` describes `check` as running `git diff` on
the module files.
**Status:** Done
