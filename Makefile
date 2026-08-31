# Makefile for tfscli.
#
# Every verification the project has is reachable through `make check`, and CI
# runs that same target, so "green locally, red in CI" cannot happen by
# construction. Recipes avoid Unix-only utilities: file removal and the checks
# that would otherwise need a shell live in Go programs under tools/, which are
# tagged `//go:build ignore` and therefore invisible to go build, go vet, and
# go test.
#
# Prerequisites: Go, and — for `lint` and the release targets — golangci-lint
# v2 and goreleaser v2 on PATH.

# The changelog section `make release-notes` reads. The goreleaser before-hook
# always passes it explicitly — Unreleased for a snapshot, the tag for a real
# release — so this default only serves the target run by hand, e.g.
# `make release-notes VERSION=0.1.0`.
VERSION ?= Unreleased

.PHONY: build lint test check release-notes release release-publish clean

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

# Write the body of the release for VERSION to docs/release-notes.md: the
# changelog section for that version, then docs/release-footer.md. Called by
# the goreleaser before-hook, and by the release workflow to read back what it
# has just written.
release-notes:
	go run tools/release-notes.go $(VERSION)

# Build a local snapshot release into dist/ — archives and checksums for every
# target platform — without tagging or publishing anything. Run it to see what
# a real release would contain. Not the release body: goreleaser skips the
# whole changelog step for a snapshot, so passing --release-notes here would
# only look like a check that never runs.
release:
	goreleaser release --snapshot --clean

# Publish the release for the current tag as a draft on GitHub. This is what
# .github/workflows/release.yml runs; it needs a tag and a GITHUB_TOKEN, and
# there is no reason to run it by hand. --release-notes is the entire body:
# it turns off goreleaser's own changelog generation, and with it the
# --release-header and --release-footer flags, which is why the footer is part
# of the file rather than a flag of its own.
release-publish:
	goreleaser release --clean --release-notes docs/release-notes.md

# ---------------------------------------------------------------------------
# Clean
# ---------------------------------------------------------------------------

# Remove every build artifact, leaving the working tree as git sees it.
clean:
	go run tools/rm.go tfscli tfscli.exe coverage.out docs/release-notes.md
	go run tools/rm.go -r dist
