//go:build ignore

// Command check-coverage fails the build when the total statement coverage of
// a profile falls below a threshold. It shells out to `go tool cover` rather
// than parsing the profile itself, so that the number it checks is exactly the
// number `make test` prints.
//
// Usage:
//
//	go run tools/check-coverage.go <threshold> [profile]
//
// The profile defaults to coverage.out. The threshold is a percentage, with or
// without the % sign.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// totalLine matches the summary `go tool cover -func` prints last, e.g.
// "total:\t(statements)\t93.4%".
var totalLine = regexp.MustCompile(`(?m)^total:.*?([0-9.]+)%`)

func main() {
	if len(os.Args) < 2 || len(os.Args) > 3 {
		fmt.Fprintln(os.Stderr, "usage: go run tools/check-coverage.go <threshold> [profile]")
		os.Exit(2)
	}

	threshold, err := strconv.ParseFloat(strings.TrimSuffix(os.Args[1], "%"), 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "check-coverage: %q is not a percentage: %v\n", os.Args[1], err)
		os.Exit(2)
	}

	profile := "coverage.out"
	if len(os.Args) == 3 {
		profile = os.Args[2]
	}

	total, err := totalCoverage(profile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "check-coverage: %v\n", err)
		os.Exit(1)
	}

	if total < threshold {
		fmt.Fprintf(os.Stderr, "check-coverage: total coverage %.1f%% is below the required %.1f%%\n", total, threshold)
		os.Exit(1)
	}
	fmt.Printf("check-coverage: total coverage %.1f%% meets the required %.1f%%\n", total, threshold)
}

func totalCoverage(profile string) (float64, error) {
	out, err := exec.Command("go", "tool", "cover", "-func="+profile).CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("go tool cover: %v: %s", err, strings.TrimSpace(string(out)))
	}

	m := totalLine.FindSubmatch(out)
	if m == nil {
		return 0, fmt.Errorf("no total line in the output of go tool cover -func=%s", profile)
	}
	return strconv.ParseFloat(string(m[1]), 64)
}
