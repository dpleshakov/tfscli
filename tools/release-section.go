//go:build ignore

// Command release-section turns the unreleased entries of CHANGELOG.md into a
// released section: it inserts "## [X.Y.Z] - YYYY-MM-DD" directly below
// "## [Unreleased]", so that the entries collected so far belong to the new
// version and "[Unreleased]" is left empty. The rest of the file is kept as it
// is. The release workflow runs this instead of asking a human to edit the
// file in a format that tools/release-notes.go then has to match.
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

var (
	versionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$`)
	unreleased     = regexp.MustCompile(`(?m)^## \[Unreleased\][ \t]*$`)
	anySection     = regexp.MustCompile(`(?m)^## \[`)
	anyEntry       = regexp.MustCompile(`(?m)^[ \t]*- `)
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
	fmt.Printf("release-section: %s now has the section [%s] - %s\n", changelog, version, date)
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "release-section: %v\n", err)
	os.Exit(1)
}

// release returns the changelog with a heading for version inserted below the
// unreleased heading.
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

	// The unreleased entries run from the heading up to the next section
	// heading, or to the end of the file.
	body := changelog[loc[1]:]
	if next := anySection.FindStringIndex(body); next != nil {
		body = body[:next[0]]
	}
	if !anyEntry.MatchString(body) {
		return "", errors.New("section [Unreleased] has no entries to release")
	}

	heading := fmt.Sprintf("\n\n## [%s] - %s", version, date)
	return changelog[:loc[1]] + heading + changelog[loc[1]:], nil
}
