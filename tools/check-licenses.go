//go:build ignore

// Command check-licenses fails the build when a dependency linked into tfscli
// on any release platform is under a license outside the allowed list. It runs
// `go-licenses check` once per platform, because the dependencies differ by
// platform: inconshreveable/mousetrap is linked on Windows only.
//
// go-licenses is installed into a temporary directory for the platform this
// program runs on, and only then run with each platform's GOOS and GOARCH.
// `go run` with those variables set would build go-licenses for the target
// platform, which this one cannot execute; setting them in a make recipe
// would need a Unix shell.
//
// Usage:
//
//	go run tools/check-licenses.go
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// goLicenses is the go-licenses release used here; .goreleaser.yaml installs
// the same one to generate the license sets, and the two change together.
const goLicenses = "github.com/google/go-licenses/v2@v2.0.1"

// allowed lists the licenses a dependency may be under. Each allows a binary
// distribution that carries its text, which tfscli licenses does; adding one
// is a decision, not an edit of the list.
const allowed = "MIT,BSD-3-Clause,Apache-2.0"

// platforms are the release targets, as builds in .goreleaser.yaml lists them.
var platforms = []string{
	"linux/amd64", "linux/arm64",
	"windows/amd64", "windows/arm64",
	"darwin/amd64", "darwin/arm64",
}

func main() {
	os.Exit(run())
}

func run() int {
	bin, err := os.MkdirTemp("", "check-licenses")
	if err != nil {
		fmt.Fprintf(os.Stderr, "check-licenses: %v\n", err)
		return 1
	}
	defer os.RemoveAll(bin)

	install := exec.Command("go", "install", goLicenses)
	install.Env = withEnv("GOBIN="+bin, "GOOS=", "GOARCH=")
	if out, err := install.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "check-licenses: go install %s: %v\n%s", goLicenses, err, out)
		return 1
	}
	tool := filepath.Join(bin, "go-licenses")
	if runtime.GOOS == "windows" {
		tool += ".exe"
	}

	failed := 0
	for _, p := range platforms {
		goos, goarch, _ := strings.Cut(p, "/")
		check := exec.Command(tool, "check", "./cmd/tfscli", "--allowed_licenses="+allowed)
		check.Env = withEnv("GOOS="+goos, "GOARCH="+goarch)
		// go-licenses logs warnings about assembly files it cannot inspect on
		// every run; they are shown only when the check fails.
		if out, err := check.CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "check-licenses: %s: %v\n%s", p, err, out)
			failed++
			continue
		}
		fmt.Printf("check-licenses: %s: every license is allowed\n", p)
	}
	if failed > 0 {
		fmt.Fprintf(os.Stderr, "check-licenses: the check failed on %d of %d platforms; allowed: %s\n", failed, len(platforms), allowed)
		return 1
	}
	return 0
}

// withEnv returns the environment of this process with the given NAME=value
// pairs replacing any variables of the same names.
func withEnv(pairs ...string) []string {
	env := make([]string, 0, len(os.Environ())+len(pairs))
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		replaced := false
		for _, p := range pairs {
			if pn, _, _ := strings.Cut(p, "="); strings.EqualFold(pn, name) {
				replaced = true
				break
			}
		}
		if !replaced {
			env = append(env, kv)
		}
	}
	return append(env, pairs...)
}
