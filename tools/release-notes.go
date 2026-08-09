//go:build ignore

// Command release-notes extracts one version section from CHANGELOG.md and
// writes it to docs/release-notes.md, which goreleaser passes to
// --release-notes. The changelog stays the single source of truth for what a
// release contains.
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
		fmt.Fprintf(os.Stderr, "release-notes: %v\n", err)
		os.Exit(1)
	}

	section, err := extract(string(source), version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "release-notes: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "release-notes: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(output, []byte(section), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "release-notes: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("release-notes: wrote the [%s] section of %s to %s\n", version, changelog, output)
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

	body = strings.TrimSpace(body)
	if body == "" {
		return "", fmt.Errorf("section [%s] of the changelog is empty", version)
	}
	return body + "\n", nil
}
