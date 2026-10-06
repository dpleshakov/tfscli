# Makefile for tfscli.
#
# Every verification the project has is reachable through `make check`, and CI
# runs that same target, so "green locally, red in CI" cannot happen by
# construction. Recipes avoid Unix-only utilities: the checks that would
# otherwise need a shell live in Go programs under tools/, which are tagged
# `//go:build ignore` and therefore invisible to go build, go vet, and go test,
# and file removal goes through git, which every platform has. Releasing is not
# done from here: .github/workflows/release.yml runs the release tools and
# goreleaser itself.
#
# Prerequisites: Go, and — for `lint` — golangci-lint v2 on PATH.

.PHONY: build lint test licenses check clean

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------

# Vet, test, and compile the binary into the repository root. The everyday
# target while working on the code.
build:
	go vet ./...
	go test ./...
	go build ./cmd/tfscli

# ---------------------------------------------------------------------------
# Test
# ---------------------------------------------------------------------------

# Make sure the module files match the imports, and run golangci-lint. Run it
# before committing; CI runs it as part of check. `go mod tidy -diff` writes
# nothing: it prints the changes the module files need and fails, and fixing
# them is a deliberate `go mod tidy` whose result goes into the commit.
lint:
	go mod tidy -diff
	golangci-lint run

# Measure statement coverage of the internal packages, print the per-function
# breakdown, and fail below the threshold. cmd/tfscli is left out on purpose:
# it is a handful of lines delegating to internal/cli.
test:
	go test -coverprofile=coverage.out ./internal/...
	go tool cover -func=coverage.out
	go run tools/check-coverage.go 85

# Fail when a dependency linked into the binary is under a license not on the
# list. Every license here allows a binary distribution that carries its text,
# which tfscli licenses does; a new one is a decision, not an edit of the list.
# `go run` with a version needs nothing installed and leaves go.mod alone. The
# check follows the dependencies of the platform it runs on.
licenses:
	go run github.com/google/go-licenses/v2@v2.0.1 check ./cmd/tfscli --allowed_licenses=MIT,BSD-3-Clause,Apache-2.0

# Everything the project verifies, in one target. This is what CI runs.
check: build lint test licenses

# ---------------------------------------------------------------------------
# Clean
# ---------------------------------------------------------------------------

# Remove every build artifact — the binary, the coverage profile, and the dist/
# and the third-party licenses a local goreleaser snapshot leaves — leaving the
# working tree as git sees it. -X limits git clean to ignored files, so the
# committed placeholders under internal/licenses/embed stay; -d lets it remove
# directories; and the paths restrict it to exactly these artifacts, so a local
# config.json, IDE settings, and untracked work are never touched.
clean:
	git clean -fdX -- tfscli tfscli.exe coverage.out dist internal/licenses/embed
