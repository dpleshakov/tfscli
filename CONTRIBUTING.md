# Contributing to tfscli

The developer handbook for this repository. `README.md` documents the tool for its users.

## Quick reference

| Target | What it does | When to run it |
|---|---|---|
| `make build` | `go vet`, `go test`, then builds the binary into the repository root | While working on the code |
| `make lint` | `go mod tidy -diff`, which fails if `go.mod` or `go.sum` drifted from the imports, and `golangci-lint run` | Before committing |
| `make test` | Measures coverage of `internal/...` and fails below 85% | While working on tests |
| `make licenses` | `go-licenses check` for each of the six release platforms (`tools/check-licenses.go`), which fails if a dependency linked into the binary is under a license other than MIT, BSD-3-Clause, or Apache-2.0 | Before committing a dependency change |
| `make check` | `build`, `lint`, `test`, and `licenses` | Before pushing — CI runs exactly this |
| `make clean` | Removes the binary, `coverage.out`, `dist/`, and the license sets a snapshot generates, through `git clean`, touching nothing else | Any time |

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

Every other target needs Go alone; `make licenses` installs go-licenses into a
temporary directory itself. [`goreleaser`](https://goreleaser.com) v2 is needed only to
preview a release locally (see "Releasing"); the release itself installs it in the
workflow. The preview installs go-licenses with `go install` and runs it by name, so Go's install directory —
`$(go env GOBIN)`, or `$(go env GOPATH)/bin` when that is unset — must be on `PATH`.

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
  on Linux. File removal goes through `git clean`, and the checks that would otherwise need
  a shell live in Go programs under `tools/`, tagged `//go:build ignore` so that
  `go build ./...`, `go vet ./...`, and `go test ./...` do not see them.

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

## Project documents

Each document has one job, and new information goes to the document whose job it is.
`project-brief.md` and `architecture.md` also state their purpose at the top.

| Document | What it records | When it changes |
|---|---|---|
| `docs/project-brief.md` | Purpose, audience, scope, constraints, non-goals, and design principles | A product decision changes — not when a feature is added |
| `docs/architecture.md` | Technology stack, architecture, and project structure, with rationale | The stack, the modules, their interactions, or the top-level structure change |
| `README.md` | What tfscli does and how to use it | Any user-visible behaviour changes |
| `skills/tfscli/SKILL.md` | How an AI agent uses tfscli; ships in every release archive | A command, a flag, an error category, or an environment variable changes |
| `CHANGELOG.md` | User-visible changes, per release | Any user-visible change |
| `CONTRIBUTING.md` | How the project is built, checked, released, and worked on | The build, the rules, or the process change |
| `CLAUDE.md` | Guidance for AI agents working in the repository | Anything it summarises changes |
| `docs/backlog.md` | Accepted work not started yet, in priority order | Work is accepted, started, or dropped |
| `docs/tech-debt.md` | Consciously deferred problems | A problem is deferred or closed |
| `docs/YYYY-MM-DD-tasks-<slug>.md` | The plan and progress of one unit of work | The work progresses; archived to `docs/archive/` when done |
| `.claude/skills/` | The process rules below | The process changes |

## Process

Work is planned and tracked with four process skills kept in the repository. Each is a
plain Markdown file and states the full rules, for people as much as for AI agents,
completed by the project choices listed below:

- `.claude/skills/tasks/SKILL.md` — tasks files;
- `.claude/skills/backlog/SKILL.md` — `docs/backlog.md`;
- `.claude/skills/changelog/SKILL.md` — `CHANGELOG.md`;
- `.claude/skills/tech-debt/SKILL.md` — `docs/tech-debt.md`.

The plans, the backlog, and the tech debt are kept as files in the repository rather
than in an issue tracker, so that each decision is versioned together with the change it
concerns, is reviewed in the same pull request, and can be read by an AI agent without
access to an external service.

The skills are written to be portable: the same files are also used outside this
repository, which is why they make no assumptions about the project. A copy installed
as a personal skill takes precedence over the project one in Claude Code, so each skill
opens with a "Project version first" section that sends the reader to the repository
copy when one exists; in the repository copy that section has no effect.

The skills leave a few choices to the project. In tfscli:

- Tasks files live in `docs/`, and archived ones in `docs/archive/`. Work taken from the
  backlog and the fix of a tech debt entry are planned as a tasks file.
- The changelog preamble names Semantic Versioning, as declared in "Choosing the
  version", and the version heading is inserted by the release workflow (see
  "Releasing").

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

### Choosing the version

tfscli follows [Semantic Versioning 2.0.0](https://semver.org/spec/v2.0.0.html). While the
major version is 0, the usual convention for `0.y.z` applies:

- a release that breaks existing usage raises the minor version and resets the patch
  version — `0.1.0` after `0.0.6`. Breaking usage means removing or renaming a command, a
  flag, an environment variable, or a configuration key, or refusing input that used to
  be accepted. An entry under `Removed` in `[Unreleased]`, or an entry under `Changed`
  that does one of these, makes the release breaking. An entry under `Deprecated` does
  not: what it names still works;
- any other release — new features, deprecations, and fixes only — raises the patch
  version.

Version `1.0.0` is a separate, explicit decision. From then on, a breaking release raises
the major version.

### Steps

1. Everything going into the release is merged into `main` and recorded under
   `## [Unreleased]` in `CHANGELOG.md`.
2. `skills/tfscli/SKILL.md` still describes the tool as it now is: the command tree and
   the flags in `internal/cli`, the error categories in `internal/tfserr`, and the
   environment variables in `internal/config`. Nothing verifies this, and the file ships
   inside every archive as what an AI agent reads instead of `--help`.
3. **Actions → Release → Run workflow**, from `main`, with the version chosen as in
   "Choosing the version" — `0.1.0`, without the leading `v`.
4. When the job is green, open the releases page, read the notes, confirm the seven assets
   are there, and press **Publish**. Nothing is public until then.

There is no other entry point: the workflow runs on `workflow_dispatch` alone, and pushing
a tag by hand does nothing.

What the job does, in order: `make check` on the commit it is about to tag; then
`tools/release-section.go`, which inserts `## [X.Y.Z] - YYYY-MM-DD` directly below
`## [Unreleased]`, so that the unreleased entries become the new version and
`[Unreleased]` is left empty; then `tools/release-notes.go`, which reads that section back
from the rewritten changelog into a file outside the repository and prints it to the log;
then `goreleaser build --snapshot`, which builds the six binaries and publishes nothing,
and a check that each platform's license set was generated and that the linux/amd64
binary prints it; then the commit `Release X.Y.Z`, the tag, and the push; then
goreleaser, run directly by its action rather than through the `Makefile`. Both
goreleaser runs first install go-licenses v2.0.1, which they find because
`actions/setup-go` puts `$(go env GOPATH)/bin` on `PATH`; each build then saves the
licenses of its target's dependencies into
`internal/licenses/embed/<os>-<arch>/third-party/`, which the binary embeds and
`tfscli licenses` prints. The result is six archives — linux, windows, darwin × amd64,
arm64 — each unpacking into a directory of its own name and holding the binary,
`LICENSE`, and `skills/`, plus `checksums.txt`, attached to a draft release titled
`tfscli vX.Y.Z`. Its body is the extracted section, passed as `--release-notes`, followed
by `release.footer` from `.goreleaser.yaml`. What goes into each archive is also
`.goreleaser.yaml`; that the release is a draft is `draft: true` there.

The footer is the `release.footer` field of the configuration on purpose, not the
`--release-footer` flag. The flag decorates the changelog goreleaser generates from the
commit log, and `--release-notes` switches that generation off; passed together, the
footer is dropped without a warning. The field is added to the body whatever the notes
came from. Anything else that belongs at the end of every release page goes into that
field, never into a flag.

The body of the next release is the `[Unreleased]` section as it stands, followed by the
footer; `go run tools/release-notes.go Unreleased` prints the section. To see what the
archives would contain without making a release, run `goreleaser release --snapshot
--clean`, which builds them into `dist/` and publishes nothing. It also leaves the
generated license sets in `internal/licenses/embed/`, which git ignores and `make clean`
removes. A snapshot renders no release body: goreleaser skips the changelog step
entirely for it.

### When it goes wrong

Everything that can be checked is checked before the push, and a failure there leaves the
repository exactly as it was — fix it through a pull request and run the workflow again:

- `make check` is red on the commit being released;
- the version is not of the form `X.Y.Z`, or the changelog already has a section for it;
- `[Unreleased]` has no entries, so there is nothing to release;
- the snapshot build fails, or a binary would ship without its third-party licenses.

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
