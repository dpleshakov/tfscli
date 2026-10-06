package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dpleshakov/tfscli/internal/licenses"
)

func TestLicensesPrintsTheLicenses(t *testing.T) {
	stdout, stderr, code := execute(t, nil, "licenses")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want nothing", stderr)
	}
	if stdout != licenses.Text() {
		t.Errorf("stdout = %q, want the text of internal/licenses", stdout)
	}
}

// TestLicensesNeedsNoSettings runs the command with no credential and a config
// file that cannot be parsed: it reads neither.
func TestLicensesNeedsNoSettings(t *testing.T) {
	isolate(t)
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	writeTestFile(t, filepath.Join(cfgHome, "tfscli", "config.json"), "{not json")

	var out, errOut bytes.Buffer
	code := run(testBuild, []string{"licenses"}, noTerminal{}, &out, &errOut)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, errOut.String())
	}
	if !strings.HasPrefix(out.String(), "tfscli includes the following components:") {
		t.Errorf("stdout = %q, want the licenses", out.String())
	}
}

func TestLicensesRefusesArguments(t *testing.T) {
	stdout, stderr, code := execute(t, nil, "licenses", "extra")

	if code == 0 {
		t.Fatalf("exit code = 0, want non-zero (stdout: %s)", stdout)
	}
	if !strings.HasPrefix(stderr, "Error [config]: ") {
		t.Errorf("stderr = %q, want a config error", stderr)
	}
}
