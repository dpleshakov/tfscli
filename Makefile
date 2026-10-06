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

.PHONY: build lint test check clean

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

# Everything the project verifies, in one target. This is what CI runs.
check: build lint test

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
