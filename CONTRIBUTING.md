# Contributing to tfscli

The developer handbook for this repository. `README.md` documents the tool for its users.

## Quick reference

| Target | What it does | When to run it |
|---|---|---|
| `make build` | `go vet`, `go test`, then builds the binary into the repository root | While working on the code |
| `make lint` | `go mod tidy` and `golangci-lint run` | Before committing |
| `make test` | Measures coverage of `internal/...` and fails below 85% | While working on tests |
| `make check` | `build`, `lint`, `test`, then fails if the `go mod tidy` inside `lint` changed `go.mod` or `go.sum` | Before pushing — CI runs exactly this |
| `make release-notes VERSION=X.Y.Z` | Writes that changelog section, then the footer, into the gitignored `docs/release-notes.md` | To preview the release body |
| `make release` | Builds a local snapshot release into `dist/`, publishing nothing | To see what a release would contain |
| `make release-publish` | The same archives, uploaded to a draft GitHub release for the current tag | Never by hand — the release workflow runs it |
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
- **[`golangci-lint`](https://golangci-lint.run) v2.13 or newer**, for `make lint`, built
  with a Go no older than the one it runs against — the linter type-checks the standard
  library with the `go/types` compiled into it, so a v2.12 binary built with Go 1.26 fails
  on the Go 1.27 sources of `math/rand/v2` (`method must have no type parameters`), and
  pre-v2.12 releases bundle a staticcheck that panics on the Go 1.26 standard library
  (`buildir: interface conversion`). A release binary is built with the Go current at the
  time it was cut; `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2`
  builds it with the Go you have. CI pins v2.13 against Go 1.26.
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

- PAT authentication only; no SSPI or NTLM. The PAT is entered interactively by
  `tfscli auth login` and never accepted as a command argument.
- Read-only in v1; write operations additionally depend on JSON output landing first.
- Markdown is the default output format.
- The command hierarchy and parameter names mirror the TFS REST API; a deviation needs an
  explicit usability justification, and no query syntax or batch endpoint is invented.
- Stateless: no daemon, no background process, no on-disk state beyond the config file and the credential written by `auth login`.
- The error categories `auth`, `not_found`, `forbidden`, `server`, `config`, and `network`
  are a contract. New ones may be added; existing ones are never renamed or removed.
- Configuration precedence is flag, environment variable, config file, built-in default. The server URL, the collection, and the PAT stay outside it: they come only from `auth.json` or `TFSCLI_AUTH`, together.

**Dependencies.** Three direct dependencies — `cobra`, `html-to-markdown/v2`, and
`golang.org/x/term` — are the intended total. HTTP is `net/http`; JSON and configuration
parsing are `encoding/json`. A fourth dependency is a decision recorded in
`docs/architecture.md`, not a `go get`.

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
  refactoring, tests, documentation, or invisible dependency updates. The format is
  [Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/) without version
  comparison links. Sections are exactly `Added`, `Changed`, `Removed`, and `Fixed`, in
  that order, and a section heading appears only when it has entries; an entry is one
  sentence starting with the subject of the change rather than a verb. Unreleased work
  accumulates under `[Unreleased]`.
- **Tech debt.** A consciously deferred compromise is recorded in `docs/tech-debt.md`
  rather than left in a comment, and is closed through the normal tasks-file workflow.

## Commits and branches

Every change reaches `main` through a pull request, merged only when the `check` job of
`.github/workflows/ci.yml` is green. The `Release` workflow is the sole exception — it
pushes the release commit and the tag to `main` directly. This is a convention rather than
a setting: the repository is private on the free plan, where GitHub does not enforce
rulesets. What protecting `main` would take is recorded as TD-07 in `docs/tech-debt.md`.

Commit messages follow what the history already does:

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
- **the section heading** `## [X.Y.Z] - YYYY-MM-DD` in `CHANGELOG.md`, from which
  `tools/release-notes.go` extracts the release notes.

Both are written by `.github/workflows/release.yml` from the version typed into it, so
they cannot disagree. Nothing else carries a version number: `version` and `commit` in
`cmd/tfscli/main.go` are targets for `-ldflags -X` and are never edited by hand, and a
binary built without them reports `dev (unknown)`.

### Steps

1. Everything going into the release is merged into `main` and recorded under
   `## [Unreleased]` in `CHANGELOG.md`.
2. `skills/tfscli/SKILL.md` still describes the tool as it now is: the command tree and
   the flags in `internal/cli`, the error categories in `internal/tfserr`, and the
   environment variables in `internal/config`. Nothing verifies this, and the file ships
   inside every archive as what an AI agent reads instead of `--help`.
3. **Actions → Release → Run workflow**, from `main`, with the version — `0.1.0`, without
   the leading `v`.
4. When the job is green, open the releases page, read the notes, confirm the seven assets
   are there, and press **Publish**. Nothing is public until then.

There is no other entry point: the workflow runs on `workflow_dispatch` alone, and pushing
a tag by hand does nothing.

What the job does, in order: `make check` on the commit it is about to tag; then
`tools/release-section.go`, which inserts `## [X.Y.Z] - YYYY-MM-DD` directly below
`## [Unreleased]`, so that the unreleased entries become the new version and
`[Unreleased]` is left empty; then
`make release-notes`, which assembles the release body — that section followed by
`docs/release-footer.md` — and prints it to the log; then the commit `Release X.Y.Z`, the
tag, and the push; then goreleaser. The result is six archives — linux, windows, darwin
× amd64, arm64 — plus `checksums.txt`, attached to a draft release titled `tfscli vX.Y.Z`
whose body is that same file, passed as `--release-notes`. What goes into each archive is
`.goreleaser.yaml`; that the release is a draft is `draft: true` there.

The footer is part of the file and not a goreleaser flag on purpose. `--release-footer`
and `--release-header` decorate the changelog goreleaser generates from the commit log,
and `--release-notes` switches that generation off; passed together, the footer is dropped
without a warning. Anything else that belongs at the end of every release page goes into
`docs/release-footer.md`, never into a flag.

To see what a release would contain without making one, `make release` builds the same
archives into `dist/` and publishes nothing. The release body is not part of that preview:
goreleaser skips the changelog step entirely for a snapshot. To see the body, run
`make release-notes VERSION=X.Y.Z` and read `docs/release-notes.md`.

### When it goes wrong

Everything that can be checked is checked before the push, and a failure there leaves the
repository exactly as it was — fix it through a pull request and run the workflow again:

- `make check` is red on the commit being released;
- the version is not of the form `X.Y.Z`, or the changelog already has a section for it;
- `[Unreleased]` has no entries, so there is nothing to release.

After the push the failure to expect is a release body that is wrong rather than
unusable — the wrong entries under `[Unreleased]`, say. The release is a draft, so delete
it on GitHub, delete the tag, and revert the release commit through a pull request:

```
git push origin --delete vX.Y.Z
git tag -d vX.Y.Z
```

The merged revert restores `[Unreleased]`, so the workflow can be run again with the same
version. A release that has already been published is not withdrawn this way — issue the
next patch version instead.
