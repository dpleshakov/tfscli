# 2026-08-09-tasks-build.md

**Status:** Active

## Context

Everything about building tfscli exists only as prose in `README.md` and `CLAUDE.md`.
There is no reproducible entry point for a build, the binary does not know its own
version, there is no release automation, and `.golangci.yml` enables nothing beyond the
standard set. `docs/project-brief.md` commits the project to "open source on GitHub,
pre-built binaries in releases"; none of the machinery for that exists.

The target state is threefold: a single `make check` that runs every verification the
project has and that CI reproduces verbatim; a binary that reports its version and
commit; and a `goreleaser` configuration that produces archives and checksums for six
platforms from a `vX.Y.Z` tag.

Decisions recorded for this scope:

- **Publishing to GitHub is out of scope.** Creating the remote repository, the first
  push, and the first tag are deliberately not part of this work. Every file is written
  and verified locally; `.github/workflows/*` are committed ready and start executing on
  the first push. The consequence is accepted knowingly: the workflow files themselves
  cannot be exercised until then.
- **Portability through Go helper programs.** Development happens on Windows, CI runs on
  Linux. Instead of `rm -rf` in recipes, the Makefile calls small programs under
  `tools/` carrying `//go:build ignore`, invoked with `go run`. The build tag keeps them
  invisible to `go build ./...`, `go vet ./...`, and `go test ./...`.
- **CI is literally `make check`.** The only substantive workflow step invokes the same
  target a developer runs locally, which makes "green locally, red in CI" impossible by
  construction.
- **Coverage threshold is 85%.** `internal/...` currently measures 93–100% per package,
  except `htmlmd` at 75%. 85% leaves headroom without becoming a formality. `cmd/tfscli`
  is excluded from the measurement: it is thirteen lines of delegation.
- **CGO is disabled explicitly.** The dependencies are pure Go and cross-compilation
  needs no C toolchain; pinning `CGO_ENABLED=0` keeps the build independent of the
  runner environment.
- **The licence is MIT.**
- **CI triggers on `master`**, the only branch that exists. Renaming the branch before
  publishing means changing one line in each workflow.
- **Windows PE metadata is out of scope.** An icon and file properties would require a
  separate toolchain and an `.ico` asset for the cosmetics of a console utility.

Local prerequisites: `goreleaser` v2 and `golangci-lint` v2. GNU Make is already
available.

---

### TASK-01 `version-stamping`
**Description:** Add `version = "dev"` and `commit = "unknown"` to `cmd/tfscli/main.go`
as the targets for `-ldflags -X`. Widen the entry point to `cli.Run(version, commit
string) int`, keeping `main` a thin wrapper. In `internal/cli/cli.go` set `root.Version`
to `<version> (<commit>)` and `root.SetVersionTemplate("tfscli {{.Version}}\n")` so the
output reads `tfscli 0.1.0 (abc1234)`. Cobra adds the `--version` flag itself once
`Version` is non-empty; do not assign the `-v` shorthand, because `--verbose` lives next
to it.
**Definition of done:** A test in `internal/cli` asserts that `--version` prints
`tfscli dev (unknown)` and exits 0; `go build ./...` and `go test ./...` are clean.

### TASK-02 `build-helpers`
**Description:** Create a `tools/` directory holding three programs, each under
`//go:build ignore` and each opening with a comment stating what it does and how it is
invoked. `tools/rm.go` implements `rm [-r] path…`, treating a missing path as success.
`tools/check-coverage.go` runs `go tool cover -func=coverage.out`, parses the `total:`
line, and exits 1 when the percentage is below the threshold passed as its argument.
`tools/release-notes.go` extracts the `## [<version>]` section of `CHANGELOG.md` up to
the next `## [` heading and writes it to `docs/release-notes.md`, failing with a
non-zero exit code when the section is absent.
**Definition of done:** Each program does what it claims when invoked as
`go run tools/<name>.go`; `go build ./...` and `go vet ./...` do not see them.

### TASK-03 `makefile`
**Description:** Write a `Makefile` organised into Build, Test, Release, and Clean
sections, with a comment above every target stating its purpose and when to run it, and
a single variable `VERSION ?= Unreleased`. Targets: `build` runs `go vet ./...`,
`go test ./...`, and `go build ./cmd/tfscli`; `lint` runs `go mod tidy` and
`golangci-lint run`; `test` runs `go test -coverprofile=coverage.out ./internal/...`,
`go tool cover -func=coverage.out`, and the coverage check at 85; `check` depends on
`build lint test` and then asserts `git diff --exit-code go.mod go.sum`;
`release-notes` runs the extractor with `$(VERSION)`; `release` depends on
`release-notes` and calls `goreleaser release --snapshot --clean --release-notes
docs/release-notes.md --release-footer docs/release-footer.md`; `clean` removes
`tfscli`, `tfscli.exe`, `coverage.out`, `docs/release-notes.md`, and `dist/` through the
`rm` helper. Alongside this, add `docs/release-notes.md` to `.gitignore` — `dist/`,
`*.out`, and the binaries are already listed — and allow `make` and `go run tools/…` in
`.claude/settings.json`.
**Definition of done:** `make build`, `make test`, and `make clean` succeed on Windows
without any external Unix utilities, and `make check` passes end to end.

### TASK-04 `linters`
**Description:** Extend `.golangci.yml`, which currently enables only the standard set.
Under the v2 schema add a `formatters` block with `gofmt` and `goimports`
(`local-prefixes: github.com/dpleshakov/tfscli`) and a `linters` block enabling
`errcheck`, `govet`, `staticcheck`, `unused`, `revive`, `gosec`, `gocritic`, `misspell`
with the US locale, and `whitespace`; configure the `revive` `exported` rule with
`checkPrivateReceivers` and exclude `gosec` and `gocritic` from `_test.go` files. Then
fix everything the new linters report in the existing code. Suppressions are per site
only, written as `//nolint:<linter> // <reason>`.
**Definition of done:** `golangci-lint run` is clean without weakening the configuration
to accommodate individual findings; `go test ./...` still passes.

### TASK-05 `license`
**Description:** Add an MIT `LICENSE` file at the repository root, copyright 2026 Dmitry
Pleshakov. It is needed both in its own right, since the repository is intended to be
open source, and as a file the release archives include.
**Definition of done:** `LICENSE` exists at the root with the correct year and copyright
holder.

### TASK-06 `goreleaser`
**Description:** Write `.goreleaser.yaml` with `version: 2` and `project_name: tfscli`.
A `before` hook regenerates the release notes as `make release-notes VERSION={{ if
.IsSnapshot }}Unreleased{{ else }}{{ .Version }}{{ end }}`, so a snapshot picks up the
`[Unreleased]` section while a real tag picks up its own. The build entry uses
`main: ./cmd/tfscli`, `binary: tfscli`, `env: [CGO_ENABLED=0]`, ldflags
`-X main.version={{.Version}} -X main.commit={{.Commit}}`, and the matrix
`goos: [linux, windows, darwin]` × `goarch: [amd64, arm64]`. Archives are named
`tfscli-{{ .Version }}-{{ .Os }}-{{ .Arch }}`, `tar.gz` everywhere and `zip` on Windows,
and carry `LICENSE`, `README.md`, `CHANGELOG.md`, and `config.example.json` next to the
binary. The release section sets owner `dpleshakov`, name `tfscli`, the title
`tfscli v{{ .Version }}`, and `draft: true` so every release is reviewed before it is
published. Add `docs/release-footer.md`, a single line linking to the configuration
section of the README, appended to every release body.
**Definition of done:** `goreleaser check` passes.

### TASK-07 `github-workflows`
**Description:** Create `.github/workflows/ci.yml` and `.github/workflows/release.yml`.
CI triggers on `pull_request` into `master` and on `push` to `master` — the project is
worked on solo and may have no pull requests, in which case a PR-only trigger would
never fire. It runs on `ubuntu-latest` with `actions/checkout@v6`, `actions/setup-go@v6`
pinned to Go 1.26 to match `go.mod`, and `golangci/golangci-lint-action@v9` in
`install-mode: binary` with `install-only: true` and `version: v2.10`, so the action
installs the linter and the Makefile invokes it; the final step is `make check`. The
release workflow triggers on tags matching `v*`, declares `permissions: contents:
write`, checks out with `fetch-depth: 0` because goreleaser needs full history and tags,
sets up Go, and runs `goreleaser/goreleaser-action@v6` with `version: '~> v2'` and
`args: release --clean --release-notes docs/release-notes.md --release-footer
docs/release-footer.md`, passing `GITHUB_TOKEN` from `secrets.GITHUB_TOKEN`.
**Definition of done:** Both files are valid YAML; the Go version, the branch name, and
the main package path agree with `go.mod` and the repository; the steps have been
checked by hand against the Makefile targets they invoke.

### TASK-08 `release-dry-run`
**Description:** Run `make check` and then `make release` locally. Verify that `dist/`
holds six archives covering linux, windows, and darwin on amd64 and arm64, plus a
checksums file; unpack the windows/amd64 archive and run `tfscli.exe --version`, which
must report a version of the form `0.1.0-SNAPSHOT-<sha>` together with the commit hash
rather than `dev (unknown)`. Confirm that `docs/release-notes.md` contains the
`[Unreleased]` section of `CHANGELOG.md`. Finish with `make clean` and check that the
working tree is clean.
**Definition of done:** The results of the run — the contents of `dist/` and the
`--version` output — are appended below this definition of done in this file; any
discrepancy is either fixed or recorded as a new task or as tech debt.

### TASK-09 `docs`
**Description:** Rewrite the Installation section of `README.md` to lead with
downloading a pre-built archive from the releases page — which archives exist, where the
checksums are — and to keep building from source as the second option; replace the loose
list of `go` commands in the Development section with a table of Makefile targets; and
mention `--version`. Add the user-facing `CHANGELOG.md` entry for `--version` following
the `changelog` skill. In `CLAUDE.md`, rewrite the "Build / test commands" section
around the Makefile targets and remove the stale "Project state" paragraph, which still
describes the packages as doc comments and `cli.Run()` as a stub.
**Definition of done:** `README.md`, `CHANGELOG.md`, and `CLAUDE.md` describe the actual
state of the project, and no command quoted in the documentation disagrees with the
Makefile.
