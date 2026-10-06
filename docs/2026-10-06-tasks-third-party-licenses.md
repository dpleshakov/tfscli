# 2026-10-06-tasks-third-party-licenses.md

**Status:** Active

## Context

Goal: every tfscli binary carries the license texts its distribution requires,
and the release archive holds only what a user needs.

The binary statically contains eight third-party modules — MIT
(`JohannesKaufmann/html-to-markdown/v2`, `JohannesKaufmann/dom`), BSD-3-Clause
(`spf13/pflag`, `golang.org/x/term`, `x/sys`, `x/net`), Apache-2.0
(`spf13/cobra`, and `inconshreveable/mousetrap` on Windows only) — and the Go
standard library and runtime (BSD-3-Clause). Each of these licenses requires
its text, and for MIT and BSD the copyright notice, to accompany a binary
copy. No module carries a NOTICE file. Today the archives ship only the
project's own `LICENSE`.

Decisions already made:

- The license texts are embedded in the binary and printed by
  `tfscli licenses`, so that they reach every copy of the binary, including
  one taken out of the archive. This follows GitHub CLI (`cli/cli`:
  `script/licenses`, `internal/licenses`, `gh licenses`).
- `tfscli licenses` takes no flags and no arguments, prints to stdout, needs
  no credential and no configuration, and exits 0. Its output is a list of
  components, the license of the Go standard library, and the texts of the
  third-party licenses. The list is built from the directory paths of the
  embedded set: `go-licenses report` writes to stdout, and a goreleaser hook
  cannot redirect it without a shell. The license name is visible at the top
  of each text.
- A binary not built by the release (`go build`, `go install`, `make check`)
  prints the Go license and a note that the third-party texts are only in
  release builds.
- The set is generated per platform, `GOOS`/`GOARCH`, by `go-licenses` v2.0.1
  (`github.com/google/go-licenses/v2`), a build tool that is never linked into
  the binary and does not enter `go.mod`. Each binary embeds only the set of
  its own platform.
- Generation is configured in `.goreleaser.yaml` alone, with no script: a
  `before` hook runs `go install github.com/google/go-licenses/v2@v2.0.1`, and
  a build pre-hook runs `go-licenses save ./cmd/tfscli` with `GOOS` and
  `GOARCH` set from the target, into
  `internal/licenses/embed/<os>-<arch>/third-party`,
  `--ignore github.com/dpleshakov/tfscli --force`. A goreleaser hook runs
  without a shell and accepts templated `env`; a separate directory per target
  keeps the parallel builds from overwriting each other. `go run …@v2.0.1`
  cannot be used in the pre-hook: with the target's `GOOS` it builds
  go-licenses for that platform and cannot execute it.
- The generated `third-party/` directories are ignored by git. Each
  `embed/<os>-<arch>/` holds a committed placeholder so that `go:embed`
  compiles without a set; the placeholder sits outside `third-party/`, which
  `--force` deletes.
- The Go license is a committed copy, `internal/licenses/go.LICENSE`, embedded
  in every build; a test compares it with `$(go env GOROOT)/LICENSE`.
- `make check` gains a `licenses` target:
  `go run github.com/google/go-licenses/v2@v2.0.1 check ./cmd/tfscli
  --allowed_licenses=MIT,BSD-3-Clause,Apache-2.0`. `go run` needs nothing
  installed beforehand, so the tools required for `make check` do not change.
  The check covers the platform it runs on.
- The archive holds the binary, `LICENSE`, and `skills/`, wrapped in a
  directory named like the archive (`wrap_in_directory`). `README.md`,
  `CHANGELOG.md`, and `config.example.json` leave the archive;
  `config.example.json` is deleted from the repository, since README shows the
  same example.
- `docs/project-brief.md` does not change: the command naming rule covers
  REST API operations only, and `licenses` is not one.

---

### TASK-01 `licenses-package`
**Description:** Add the package `internal/licenses`: a placeholder in
`embed/<os>-<arch>/` for each of the six release platforms (linux, windows,
darwin × amd64, arm64), one build-tagged file per platform embedding its
directory and one for every other platform embedding nothing, the committed
`go.LICENSE`, and a function assembling the output described in Context.
Ignore the generated `third-party/` directories in `.gitignore`, and make
`make clean` remove them.
**Definition of done:** Tests cover the output with a set, without a set, and
the comparison of `go.LICENSE` with the toolchain's `LICENSE`; `make check` is
green with coverage of `internal/...` at 85% or above.
**Status:** Done

### TASK-02 `licenses-command`
**Description:** Add `tfscli licenses` to `internal/cli`, printing the output
of `internal/licenses` to stdout.
**Definition of done:** Tests show that the command prints the output, runs
without a credential or a config file, and refuses arguments; `make check` is
green.
**Status:** Done

### TASK-03 `release-generation`
**Description:** Add the `before` hook installing go-licenses and the build
pre-hook generating the set to `.goreleaser.yaml`.
**Definition of done:** `goreleaser release --snapshot --clean` succeeds;
`tfscli licenses` of the windows/amd64 binary lists the Go standard library
and the eight third-party modules and prints their texts; `git status` is
clean after the snapshot.
**Status:** Done

### TASK-04 `archive-contents`
**Description:** Reduce the archive files to `LICENSE` and `skills`, enable
`wrap_in_directory`, and delete `config.example.json` from the repository,
rewording its mentions in `README.md` and `.gitignore`.
**Definition of done:** The snapshot archives unpack into
`tfscli-<version>-<os>-<arch>/` holding exactly the binary, `LICENSE`, and
`skills/tfscli/SKILL.md`; no file of the repository mentions
`config.example.json` outside `docs/archive/`.
**Status:** Pending

### TASK-05 `licenses-check`
**Description:** Add the `licenses` target to the `Makefile` and include it in
`check`.
**Definition of done:** `make check` is green and runs the target; with a
license removed from `--allowed_licenses`, the target fails.
**Status:** Pending

### TASK-06 `docs`
**Description:** Document the change: `README.md` (the `licenses` command, the
archive contents and layout, installation), `CHANGELOG.md` (Added: the
command; Changed: the archive contents and layout), `docs/architecture.md`
(the package, go-licenses as a build tool), `CONTRIBUTING.md` (the `licenses`
target, go-licenses in the release and the snapshot, `$(go env GOPATH)/bin` on
`PATH` for a snapshot), `skills/tfscli/SKILL.md` (the command), and
`CLAUDE.md` where it summarises any of these.
**Definition of done:** The documents describe the new state; `make check` is
green.
**Status:** Pending
