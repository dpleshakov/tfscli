# Contributing to tfscli

The developer handbook for this repository. `README.md` documents the tool for its users.

## Quick reference

| Target | What it does | When to run it |
|---|---|---|
| `make build` | `go vet`, `go test`, then builds the binary into the repository root | While working on the code |
| `make lint` | `go mod tidy` and `golangci-lint run` | Before committing |
| `make test` | Measures coverage of `internal/...` and fails below 85% | While working on tests |
| `make check` | `build`, `lint`, `test`, then fails if the `go mod tidy` inside `lint` changed `go.mod` or `go.sum` | Before pushing — CI runs exactly this |
| `make release-notes VERSION=X.Y.Z` | Extracts that changelog section into the gitignored `docs/release-notes.md` | To preview release notes |
| `make release` | Builds a local snapshot release into `dist/`, publishing nothing | To see what a release would contain |
| `make clean` | Removes the binary, `coverage.out`, `docs/release-notes.md`, and `dist/` | Any time |

`make check` is the whole gate: `.github/workflows/ci.yml` installs the linter, runs
`make check`, and does nothing else, so a green check locally is a green CI by
construction.

A single test:

```
go test -run TestName ./internal/config
```

Cross-compilation:

```
GOOS=windows GOARCH=amd64 go build ./cmd/tfscli
```

## Setup

- **Go 1.26 or newer** — matches the `go` directive in `go.mod` and the version both
  workflows install.
- **[`golangci-lint`](https://golangci-lint.run) v2.12 or newer**, for `make lint` —
  earlier v2 releases bundle staticcheck 0.7.0, which panics while analysing the Go 1.26
  standard library (`buildir: interface conversion`). CI pins v2.12 for the same reason.
- **[`goreleaser`](https://goreleaser.com) v2**, for `make release`.

Every other target needs Go alone.

## Code rules

- **Lint.** `.golangci.yml` lists the enabled linters, their settings, and the reason for
  each. Suppress per site, with a reason: `//nolint:<linter> // <reason>`. Turning a linter
  off repository-wide is a change to `.golangci.yml` and needs its reason recorded there.
- **Coverage.** `make test` measures `internal/...` and fails below 85%
  (`tools/check-coverage.go`). `cmd/tfscli` is left out on purpose: it is a handful of
  lines delegating to `internal/cli`.
- **Doc comments on exported identifiers**, methods on unexported receivers included —
  such methods are the bulk of this codebase.
- **No recipe may assume a Unix shell**, since development happens on Windows and CI runs
  on Linux. File removal and the checks that would otherwise need one live in Go programs
  under `tools/`, tagged `//go:build ignore` so that `go build ./...`, `go vet ./...`, and
  `go test ./...` do not see them.

## Before changing behaviour

Read `docs/project-brief.md` before any user-facing change, and `docs/architecture.md`
before adding or swapping a dependency.

The brief is the source of truth for the constraints below, which are repeated here to
catch a violation without opening it. A feature request does not override them on its own;
if a change would break one, raise it before implementing.

- PAT authentication only; no SSPI, NTLM, or interactive login.
- Read-only in v1; write operations additionally depend on JSON output landing first.
- Markdown is the default output format.
- The command hierarchy and parameter names mirror the TFS REST API; a deviation needs an
  explicit usability justification, and no query syntax or batch endpoint is invented.
- Stateless: no daemon, no background process, no on-disk state beyond the config file.
- The error categories `auth`, `not_found`, `forbidden`, `server`, `config`, and `network`
  are a contract. New ones may be added; existing ones are never renamed or removed.
- Configuration precedence is flag, environment variable, config file, built-in default.

**Dependencies.** Two direct dependencies — `cobra` and `html-to-markdown/v2` — are the
intended total. HTTP is `net/http`; JSON and configuration parsing are `encoding/json`. A
third dependency is a decision recorded in `docs/architecture.md`, not a `go get`.

## Process

Work is planned and tracked through the process skills `tasks`, `changelog`, and
`tech-debt`, which are authoritative wherever they are available. Where they are not, ask
how to proceed rather than reconstructing the process from the summary below.

- **Tasks files.** A unit of work gets `docs/YYYY-MM-DD-tasks-<slug>.md`, with
  `**Status:** Active` in the header and an ordered list of atomic tasks, each carrying a
  description, a definition of done, and a status of `Pending`, `In progress`, `Done`, or
  `Skipped — <reason>`. The file travels in the same commit as the work it describes; a
  status-only commit is never correct. When the last task is resolved, the header becomes
  `Archived` and the file moves to `docs/archive/` in that same commit.
  `docs/2026-05-20-tasks-html-quirks.md` predates this format and keeps its own — task
  types and emoji statuses — when updated.
- **Changelog.** `CHANGELOG.md` is written for the user of the tool: features, observable
  bug fixes, changed configuration keys and flags, behaviour changes, and removals, but not
  refactoring, tests, documentation, or invisible dependency updates. Sections are exactly
  `Added`, `Fixed`, `Changed`, and `Removed`; an entry is one sentence starting with the
  subject of the change rather than a verb. Unreleased work accumulates under
  `[Unreleased]`.
- **Tech debt.** A consciously deferred compromise is recorded in `docs/tech-debt.md`
  rather than left in a comment, and is closed through the normal tasks-file workflow.

## Commits and branches

`main` is the only branch. CI runs `make check` on pushes to it and on pull requests
against it. Commit messages follow what the history already does:

- A subject in the imperative, with no type prefix or scope — `Implement workitem package
  (TASK-06)`, `Add build, lint, and release automation`.
- A body in prose explaining why the change looks the way it does, including the decisions
  a reader would otherwise have to reverse-engineer from the diff. Bullets are for
  genuinely enumerable parts, not for restating the file list.
- `Closes docs/archive/<tasks-file>.md` on the commit that finishes a tasks file.

## Releasing

### Where the version lives

In two places, neither of them a source file:

- **the git tag** `vX.Y.Z` — goreleaser takes `.Version` as the tag without the leading
  `v`;
- **the section heading** `## [X.Y.Z] — YYYY-MM-DD` in `CHANGELOG.md`, from which
  `tools/release-notes.go` extracts the release notes.

The two have to agree, and nothing else carries a version number. `version` and `commit`
in `cmd/tfscli/main.go` are targets for `-ldflags -X` and are never edited by hand; a
binary built without them reports `dev (unknown)`.

### Steps

1. Everything going into the release is on `main` and `make check` is green — the release
   workflow does not run it.
2. In `CHANGELOG.md`, rename `## [Unreleased]` to `## [X.Y.Z] — YYYY-MM-DD` (em dash),
   delete the subsections that stayed empty, and add a fresh `## [Unreleased]` above it
   with the four empty subsections, separated from the released section by `---`.
3. Optional: `make release-notes VERSION=X.Y.Z` writes the extracted section to the
   gitignored `docs/release-notes.md`; `make release` goes further and builds the complete
   set of archives into `dist/`, publishing nothing.
4. **Commit the changelog change before tagging** — the workflow reads `CHANGELOG.md` as
   of the tagged commit.
5. Tag and push:

   ```
   git tag vX.Y.Z
   git push origin main --tags
   ```

   The tag push is the only trigger. `.github/workflows/release.yml` listens on
   `push: tags: ['v*']` and has no manual entry point.
6. The workflow runs goreleaser against the tagged commit, whose before-hook regenerates
   the release notes from that commit's `CHANGELOG.md`. The result is six archives — linux,
   windows, darwin × amd64, arm64 — plus `checksums.txt`, attached to a release titled
   `tfscli vX.Y.Z` whose body is the changelog section followed by `docs/release-footer.md`.
   What goes into each archive is `.goreleaser.yaml`.
7. **The release is created as a draft** (`draft: true` in `.goreleaser.yaml`). Nothing is
   public until a human opens the releases page, reads the notes, confirms the seven assets
   are there, and presses Publish.

### When it goes wrong

- **The tag has no matching changelog section** — the most common failure, and a loud one.
  `tools/release-notes.go` exits with an error, so the before-hook fails and the job dies
  before the first binary is built. A section that exists but is empty fails the same way.
- **The changelog rename was not committed before tagging.** The tag points at a commit
  whose changelog still reads `[Unreleased]`, which is the previous failure again.
- **A commit that does not build got tagged.** The release job never runs `make check`;
  the correctness of the tagged commit is on whoever tags it.
- **Undoing a tag.** Delete the draft release on GitHub, then:

  ```
  git push origin --delete vX.Y.Z
  git tag -d vX.Y.Z
  ```

  Fix the problem, commit, and tag again. A release that has already been published is not
  withdrawn this way — issue the next patch version instead.
