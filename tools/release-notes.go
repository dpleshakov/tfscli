//go:build ignore

// Command release-notes assembles the body of a GitHub release — one version
// section of CHANGELOG.md followed by docs/release-footer.md — and writes it to
// docs/release-notes.md, which goreleaser passes to --release-notes. The
// changelog stays the single source of truth for what a release contains.
//
// The footer is appended here rather than handed to goreleaser as
// --release-footer. That flag decorates the changelog goreleaser generates
// itself, and --release-notes turns that generation off, so the two cannot be
// combined: a footer passed alongside custom notes is dropped without a word.
//
// Usage:
//
//	go run tools/release-notes.go <version> [changelog] [output]
//
// The version is the text inside the brackets of the heading, so "Unreleased"
// selects "## [Unreleased]" and "0.1.0" selects "## [0.1.0]". The changelog
// defaults to CHANGELOG.md and the output to docs/release-notes.md. A version
// with no matching section is an error.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// The last lines of every release body. What to do after downloading an
// archive belongs on the release page rather than in the changelog, and a
// missing footer would silently shorten the release, so its absence is an
// error like any other.
const footerFile = "docs/release-footer.md"

func main() {
	if len(os.Args) < 2 || len(os.Args) > 4 {
		fmt.Fprintln(os.Stderr, "usage: go run tools/release-notes.go <version> [changelog] [output]")
		os.Exit(2)
	}

	version := strings.TrimPrefix(os.Args[1], "v")
	changelog := "CHANGELOG.md"
	if len(os.Args) >= 3 {
		changelog = os.Args[2]
	}
	output := filepath.Join("docs", "release-notes.md")
	if len(os.Args) == 4 {
		output = os.Args[3]
	}

	source, err := os.ReadFile(changelog)
	if err != nil {
		fail(err)
	}

	section, err := extract(string(source), version)
	if err != nil {
		fail(err)
	}

	footer, err := os.ReadFile(footerFile)
	if err != nil {
		fail(err)
	}

	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		fail(err)
	}
	if err := os.WriteFile(output, []byte(assemble(section, string(footer))), 0o644); err != nil {
		fail(err)
	}
	fmt.Printf("release-notes: wrote the [%s] section of %s, then %s, to %s\n", version, changelog, footerFile, output)
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "release-notes: %v\n", err)
	os.Exit(1)
}

// extract returns the body of the "## [version]" section, from the line after
// the heading up to the next "## [" heading or the end of the file.
func extract(changelog, version string) (string, error) {
	heading := regexp.MustCompile(`(?m)^## \[` + regexp.QuoteMeta(version) + `\].*$`)

	loc := heading.FindStringIndex(changelog)
	if loc == nil {
		return "", fmt.Errorf("no section [%s] in the changelog", version)
	}

	body := changelog[loc[1]:]
	if next := regexp.MustCompile(`(?m)^## \[`).FindStringIndex(body); next != nil {
		body = body[:next[0]]
	}

	// A section followed by another one ends with the "---" that separates the
	// two, which belongs to neither and would render as a second rule above the
	// one the footer opens with.
	body = strings.TrimSpace(body)
	if rest, found := strings.CutSuffix(body, "---"); found {
		body = strings.TrimSpace(rest)
	}
	if body == "" {
		return "", fmt.Errorf("section [%s] of the changelog is empty", version)
	}
	return body + "\n", nil
}

// assemble joins the section and the footer with a blank line between them, which
// the footer needs: a "---" on the line directly below text is not a rule but
// an underline turning that text into a heading.
func assemble(section, footer string) string {
	return strings.TrimSpace(section) + "\n\n" + strings.TrimSpace(footer) + "\n"
}
