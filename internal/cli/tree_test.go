package cli

import (
	"io"
	"testing"

	"github.com/spf13/cobra"
)

// Agent permission rules match the command text, so each command must have
// exactly one form: an alias or prefix matching would let a command run under
// a name no rule was written for.
func TestEveryCommandHasOneForm(t *testing.T) {
	if cobra.EnablePrefixMatching {
		t.Error("cobra.EnablePrefixMatching is on; command names must be matched in full")
	}

	var walk func(cmd *cobra.Command)
	walk = func(cmd *cobra.Command) {
		if len(cmd.Aliases) > 0 {
			t.Errorf("%q has aliases %q; a command has exactly one name", cmd.CommandPath(), cmd.Aliases)
		}
		for _, child := range cmd.Commands() {
			walk(child)
		}
	}
	walk(newRoot(testBuild, noTerminal{}, io.Discard, io.Discard))
}
