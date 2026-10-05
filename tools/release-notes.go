//go:build ignore

// Command release-notes extracts one version section of CHANGELOG.md, which the
// release workflow passes to goreleaser as --release-notes. The changelog stays
// the single source of truth for what a release contains; the footer of every
// release page is release.footer in .goreleaser.yaml.
//
// Usage:
//
//	go run tools/release-notes.go <version> [changelog] [output]
//
// The version is the text inside the brackets of the heading, so "Unreleased"
// selects "## [Unreleased]" and "0.1.0" selects "## [0.1.0]". The changelog
// defaults to CHANGELOG.md, and the section is written to standard output
// unless an output file is given. A version with no matching section, or with
// an empty one, is an error.
package main

import (
	"fmt"
	"os"
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

	source, err := os.ReadFile(changelog)
	if err != nil {
		fail(err)
	}

	section, err := extract(string(source), version)
	if err != nil {
		fail(err)
	}

	if len(os.Args) < 4 {
		fmt.Print(section)
		return
	}
	output := os.Args[3]
	if err := os.WriteFile(output, []byte(section), 0o644); err != nil {
		fail(err)
	}
	fmt.Printf("release-notes: wrote the [%s] section of %s to %s\n", version, changelog, output)
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

	body = strings.TrimSpace(body)
	if body == "" {
		return "", fmt.Errorf("section [%s] of the changelog is empty", version)
	}
	return body + "\n", nil
}
