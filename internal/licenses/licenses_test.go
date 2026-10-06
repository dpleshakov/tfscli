package licenses

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestTextWithSet(t *testing.T) {
	set := fstest.MapFS{
		"github.com/spf13/cobra/LICENSE.txt":           {Data: []byte("Apache License\n")},
		"golang.org/x/term/LICENSE":                    {Data: []byte("Copyright 2009 The Go Authors.\n")},
		"github.com/example/notice/LICENSE":            {Data: []byte("MIT License\n")},
		"github.com/example/notice/NOTICE":             {Data: []byte("Notice text\n")},
		"github.com/JohannesKaufmann/dom/LICENSE":      {Data: []byte("MIT License\n\nCopyright (c) 2024\n")},
		"github.com/JohannesKaufmann/dom/sub/LICENSE":  {Data: []byte("Nested\n")},
		"github.com/JohannesKaufmann/dom/LICENSE.more": {Data: []byte("More\n")},
	}

	got := text("GO LICENSE\n", set)

	wantList := "tfscli includes the following components:\n\n" +
		"  Go standard library and runtime\n" +
		"  github.com/JohannesKaufmann/dom\n" +
		"  github.com/JohannesKaufmann/dom/sub\n" +
		"  github.com/example/notice\n" +
		"  github.com/spf13/cobra\n" +
		"  golang.org/x/term\n"
	if !strings.HasPrefix(got, wantList) {
		t.Errorf("text() does not start with the component list:\n%s", got)
	}
	if strings.Contains(got, "only in the release builds") {
		t.Errorf("text() with a set says the set is missing:\n%s", got)
	}

	framed := func(name, body string) string {
		return "\n" + separator + "\n" + name + "\n" + separator + "\n\n" + body
	}
	for _, want := range []string{
		framed("Go standard library and runtime", "GO LICENSE\n"),
		framed("github.com/spf13/cobra", "Apache License\n"),
		framed("github.com/example/notice", "MIT License\n\nNotice text\n"),
		framed("github.com/JohannesKaufmann/dom", "MIT License\n\nCopyright (c) 2024\n\nMore\n"),
	} {
		if !strings.Contains(got, want) {
			t.Errorf("text() lacks %q:\n%s", want, got)
		}
	}
	if strings.Index(got, framed("Go standard library and runtime", "")) >
		strings.Index(got, framed("github.com/JohannesKaufmann/dom", "")) {
		t.Errorf("text() does not put the Go license first:\n%s", got)
	}
}

func TestTextWithoutSet(t *testing.T) {
	for name, set := range map[string]fs.FS{
		"nil":     nil,
		"empty":   fstest.MapFS{},
		"missing": sub(t, fstest.MapFS{"PLACEHOLDER": {}}, "third-party"),
	} {
		t.Run(name, func(t *testing.T) {
			got := text("GO LICENSE\n", set)
			want := "tfscli includes the following components:\n\n" +
				"  Go standard library and runtime\n\n" +
				"This build does not carry the licenses of the third-party modules it\n" +
				"includes: they are embedded only in the release builds, available at\n" +
				"https://github.com/dpleshakov/tfscli/releases.\n" +
				"\n" + separator + "\nGo standard library and runtime\n" + separator + "\n\n" +
				"GO LICENSE\n"
			if got != want {
				t.Errorf("text() =\n%s\nwant\n%s", got, want)
			}
		})
	}
}

// sub is fs.Sub that fails the test on an error, which only an invalid dir
// can cause.
func sub(t *testing.T, fsys fs.FS, dir string) fs.FS {
	t.Helper()
	s, err := fs.Sub(fsys, dir)
	if err != nil {
		t.Fatalf("fs.Sub(%q): %v", dir, err)
	}
	return s
}

// TestTextOfThisBuild covers Text over the embedded set of the platform the
// tests run on. The set is usually absent, but a local goreleaser snapshot
// leaves one behind until make clean, so the test holds either way.
func TestTextOfThisBuild(t *testing.T) {
	got := Text()
	if want := text(goLicense, sub(t, platformFS, platformDir+"/third-party")); got != want {
		t.Errorf("Text() =\n%s\nwant\n%s", got, want)
	}
	if !strings.Contains(got, goLicense) {
		t.Errorf("Text() lacks the Go license:\n%s", got)
	}
}

// TestGoLicenseMatchesToolchain keeps the committed copy of the Go license
// equal to the one of the toolchain that builds tfscli.
func TestGoLicenseMatchesToolchain(t *testing.T) {
	out, err := exec.Command("go", "env", "GOROOT").Output()
	if err != nil {
		t.Fatalf("go env GOROOT: %v", err)
	}
	want, err := os.ReadFile(filepath.Join(strings.TrimSpace(string(out)), "LICENSE"))
	if err != nil {
		t.Fatalf("reading the toolchain license: %v", err)
	}
	if goLicense != string(want) {
		t.Errorf("internal/licenses/go.LICENSE differs from LICENSE of the Go toolchain; " +
			"copy $(go env GOROOT)/LICENSE over it")
	}
}
