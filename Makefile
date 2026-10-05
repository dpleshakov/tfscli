# Makefile for tfscli.
#
# Every verification the project has is reachable through `make check`, and CI
# runs that same target, so "green locally, red in CI" cannot happen by
# construction. Recipes avoid Unix-only utilities: the checks that would
# otherwise need a shell live in Go programs under tools/, which are tagged
# `//go:build ignore` and therefore invisible to go build, go vet, and go test,
# and file removal goes through git, which every platform has. Releasing is not done from here: .github/workflows/release.yml runs
# the release tools and goreleaser itself.
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

# Run golangci-lint, and make sure the module files match the imports. Run it
# before committing; CI runs it as part of check.
lint:
	go mod tidy
	golangci-lint run

# Measure statement coverage of the internal packages, print the per-function
# breakdown, and fail below the threshold. cmd/tfscli is left out on purpose:
# it is a handful of lines delegating to internal/cli.
test:
	go test -coverprofile=coverage.out ./internal/...
	go tool cover -func=coverage.out
	go run tools/check-coverage.go 85

# Everything the project verifies, in one target. This is what CI runs. The
# final step catches a `go mod tidy` in lint that changed the module files:
# the dependency set has drifted from the imports and the change belongs in a
# commit, not in a working tree nobody looked at.
check: build lint test
	git diff --exit-code go.mod go.sum

# ---------------------------------------------------------------------------
# Clean
# ---------------------------------------------------------------------------

# Remove every build artifact, leaving the working tree as git sees it. -X
# limits git clean to ignored files and the paths to exactly these artifacts,
# so a local config.json, IDE settings, and untracked work are never touched.
clean:
	git clean -fX -- tfscli tfscli.exe coverage.out
