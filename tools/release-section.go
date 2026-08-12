//go:build ignore

// Command release-section turns the unreleased section of CHANGELOG.md into a
// released one: "## [Unreleased]" becomes "## [X.Y.Z] — YYYY-MM-DD", the
// subsections that stayed empty are dropped, and a fresh empty "[Unreleased]"
// is inserted above it. The release workflow runs this instead of asking a
// human to edit the file in a format that tools/release-notes.go then has to
// match.
//
// Usage:
//
//	go run tools/release-section.go <version> [changelog] [date]
//
// The version is X.Y.Z, optionally with a prerelease suffix and a leading "v".
// The changelog defaults to CHANGELOG.md and the date to today. Nothing is
// written unless the whole rewrite succeeds: a version that already has a
// section, a missing "[Unreleased]", and an "[Unreleased]" with no entries are
// all errors.
package main

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// The empty section that collects changes made after this release.
const scaffold = "## [Unreleased]\n\n### Added\n### Fixed\n### Changed\n### Removed\n"

// The separator the changelog puts between two sections.
const separator = "\n---\n\n"

var (
	versionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$`)
	unreleased     = regexp.MustCompile(`(?m)^## \[Unreleased\].*$`)
	anySection     = regexp.MustCompile(`(?m)^## \[`)
	anySubsection  = regexp.MustCompile(`(?m)^### `)
)

func main() {
	if len(os.Args) < 2 || len(os.Args) > 4 {
		fmt.Fprintln(os.Stderr, "usage: go run tools/release-section.go <version> [changelog] [date]")
		os.Exit(2)
	}

	version := strings.TrimPrefix(os.Args[1], "v")
	changelog := "CHANGELOG.md"
	if len(os.Args) >= 3 {
		changelog = os.Args[2]
	}
	date := time.Now().Format(time.DateOnly)
	if len(os.Args) == 4 {
		date = os.Args[3]
		if _, err := time.Parse(time.DateOnly, date); err != nil {
			fail(fmt.Errorf("%q is not a date of the form YYYY-MM-DD", date))
		}
	}

	source, err := os.ReadFile(changelog)
	if err != nil {
		fail(err)
	}

	result, err := release(string(source), version, date)
	if err != nil {
		fail(err)
	}

	if err := os.WriteFile(changelog, []byte(result), 0o644); err != nil {
		fail(err)
	}
	fmt.Printf("release-section: %s now has the section [%s] — %s\n", changelog, version, date)
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "release-section: %v\n", err)
	os.Exit(1)
}

// release returns the changelog with its unreleased section renamed to version
// and a new empty unreleased section above it.
func release(changelog, version, date string) (string, error) {
	if !versionPattern.MatchString(version) {
		return "", fmt.Errorf("%q is not a version of the form X.Y.Z", version)
	}
	released := regexp.MustCompile(`(?m)^## \[` + regexp.QuoteMeta(version) + `\]`)
	if released.MatchString(changelog) {
		return "", fmt.Errorf("the changelog already has a section [%s]", version)
	}

	loc := unreleased.FindStringIndex(changelog)
	if loc == nil {
		return "", errors.New("no section [Unreleased] in the changelog")
	}

	// Everything above the heading is kept as it is; everything from the next
	// heading down is the previous releases, which this rewrite does not touch.
	head, body, tail := changelog[:loc[0]], changelog[loc[1]:], ""
	if next := anySection.FindStringIndex(body); next != nil {
		body, tail = body[:next[0]], body[next[0]:]
	}

	entries := filled(trimSeparator(body))
	if entries == "" {
		return "", errors.New("section [Unreleased] has no entries to release")
	}

	var out strings.Builder
	out.WriteString(head)
	out.WriteString(scaffold)
	out.WriteString(separator)
	fmt.Fprintf(&out, "## [%s] — %s\n\n", version, date)
	out.WriteString(entries)
	out.WriteString("\n")
	if tail != "" {
		out.WriteString(separator)
		out.WriteString(tail)
	}
	return out.String(), nil
}

// trimSeparator removes the "---" that separates a section from the one below
// it, which a section body read up to the next heading ends with.
func trimSeparator(body string) string {
	body = strings.TrimSpace(body)
	if rest, found := strings.CutSuffix(body, "---"); found {
		body = strings.TrimSpace(rest)
	}
	return body
}

// filled returns the section body without the subsections that have no entries.
func filled(body string) string {
	at := anySubsection.FindAllStringIndex(body, -1)
	if at == nil {
		return strings.TrimSpace(body)
	}

	var kept []string
	if lead := strings.TrimSpace(body[:at[0][0]]); lead != "" {
		kept = append(kept, lead)
	}
	for i, start := range at {
		end := len(body)
		if i+1 < len(at) {
			end = at[i+1][0]
		}
		if block := strings.TrimSpace(body[start[0]:end]); hasEntry(block) {
			kept = append(kept, block)
		}
	}
	return strings.Join(kept, "\n\n")
}

// hasEntry reports whether a subsection has at least one entry under its
// heading.
func hasEntry(block string) bool {
	lines := strings.Split(block, "\n")
	for _, line := range lines[1:] {
		if strings.HasPrefix(strings.TrimSpace(line), "-") {
			return true
		}
	}
	return false
}
