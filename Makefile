# Makefile for tfscli.
#
# Every verification the project has is reachable through `make check`, and CI
# runs that same target, so "green locally, red in CI" cannot happen by
# construction. Recipes avoid Unix-only utilities: file removal and the checks
# that would otherwise need a shell live in Go programs under tools/, which are
# tagged `//go:build ignore` and therefore invisible to go build, go vet, and
# go test.
#
# Prerequisites: Go, and — for `lint` and `release` — golangci-lint v2 and
# goreleaser v2 on PATH.

# The changelog section release notes are taken from. `make release` builds a
# snapshot and therefore defaults to the section of the unreleased changes; a
# real release overrides it, e.g. `make release-notes VERSION=0.1.0`.
VERSION ?= Unreleased

.PHONY: build lint test check release-notes release clean

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
# Release
# ---------------------------------------------------------------------------

# Extract the release notes for VERSION from CHANGELOG.md. Called by the
# goreleaser before-hook; rarely worth running by hand.
release-notes:
	go run tools/release-notes.go $(VERSION)

# Build a local snapshot release into dist/ — archives and checksums for every
# target platform — without tagging or publishing anything. Run it to see what
# a real release would contain.
release: release-notes
	goreleaser release --snapshot --clean --release-notes docs/release-notes.md --release-footer docs/release-footer.md

# ---------------------------------------------------------------------------
# Clean
# ---------------------------------------------------------------------------

# Remove every build artifact, leaving the working tree as git sees it.
clean:
	go run tools/rm.go tfscli tfscli.exe coverage.out docs/release-notes.md
	go run tools/rm.go -r dist
