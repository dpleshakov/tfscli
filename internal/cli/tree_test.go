package cli

import (
	"io"
	"testing"

	"github.com/spf13/cobra"
)

// Agent permission rules match the command text, so each command must have
// exactly one form: an alias, prefix matching, or case-insensitive matching
// would let a command run under a name no rule was written for.
func TestEveryCommandHasOneForm(t *testing.T) {
	if cobra.EnablePrefixMatching {
		t.Error("cobra.EnablePrefixMatching is on; command names must be matched in full")
	}
	if cobra.EnableCaseInsensitive {
		t.Error("cobra.EnableCaseInsensitive is on; command names must be matched as written")
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

// checkFlagOrder takes every word naming a subcommand as a step of the path,
// which holds only while a command with subcommands just groups them: a
// group's own argument or flag value could otherwise name a subcommand and be
// taken for one. Persistent flags are allowed, since they are used after the
// full path.
func TestEveryGroupOnlyGroups(t *testing.T) {
	var walk func(cmd *cobra.Command)
	walk = func(cmd *cobra.Command) {
		if !cmd.HasSubCommands() {
			return
		}
		// Cobra itself rejects an argument to the root that names no
		// subcommand, when the root leaves Args unset.
		if (cmd.HasParent() || cmd.Args != nil) && cmd.ValidateArgs([]string{"x"}) == nil {
			t.Errorf("%q accepts an argument; a command with subcommands takes none", cmd.CommandPath())
		}
		if own := cmd.LocalNonPersistentFlags(); own.HasFlags() {
			t.Errorf("%q defines flags of its own; a command with subcommands has none:\n%s", cmd.CommandPath(), own.FlagUsages())
		}
		for _, child := range cmd.Commands() {
			walk(child)
		}
	}
	walk(newRoot(testBuild, noTerminal{}, io.Discard, io.Discard))
}
