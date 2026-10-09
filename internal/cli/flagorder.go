package cli

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dpleshakov/tfscli/internal/tfserr"
)

// checkFlagOrder refuses a command line whose command path does not occupy its
// first words, such as "tfscli --verbose wit work-items get 1". Agent
// permission rules match the command text by its prefix, so the command path
// has to open the command line; cobra itself accepts flags at any position.
//
// The command is the one reached from the root by taking, in order, every word
// that names a subcommand of the command reached so far, up to "--". No
// knowledge of flag values is needed: a command with subcommands takes no
// arguments and defines no flags of its own (TestEveryGroupOnlyGroups), so in
// a valid command line no word naming a subcommand can follow the path. This
// reaches at least as deep as cobra's Find, which descends only through such
// words, so cobra never runs a command whose path is not at the start.
//
// A shell completion request, which cobra serves with a hidden command it adds
// only when it executes the tree, is let through: it lists completions for the
// words after it and runs no command.
func checkFlagOrder(root *cobra.Command, args []string) error {
	if len(args) > 0 && (args[0] == cobra.ShellCompRequestCmd || args[0] == cobra.ShellCompNoDescRequestCmd) {
		return nil
	}
	cmd := root
	var path []int
	for i, arg := range args {
		if arg == "--" {
			break
		}
		if next := subcommand(cmd, arg); next != nil {
			cmd = next
			path = append(path, i)
		}
	}
	if len(path) == 0 || path[len(path)-1] == len(path)-1 {
		return nil
	}

	// The corrected command is the path followed by every other word in its
	// original order; before holds the words that preceded the last path word.
	last := path[len(path)-1]
	var before, rest []string
	for i, arg := range args {
		if slices.Contains(path, i) {
			continue
		}
		if i < last {
			before = append(before, arg)
		}
		rest = append(rest, arg)
	}

	if slices.Contains(before, "--version") {
		return &tfserr.Error{
			Category: tfserr.Config,
			Message:  fmt.Sprintf("--version belongs to tfscli itself, not to %q (run it as: tfscli --version)", cmd.CommandPath()),
		}
	}
	if err := unknownFlag(cmd, before); err != nil {
		return err
	}
	words := append(strings.Fields(cmd.CommandPath()), rest...)
	return &tfserr.Error{
		Category: tfserr.Config,
		Message: fmt.Sprintf("the command %q must come first, before its flags and arguments (run it as: %s)",
			cmd.CommandPath(), shellJoin(words)),
	}
}

// subcommand returns the child of cmd named name, or nil.
func subcommand(cmd *cobra.Command, name string) *cobra.Command {
	for _, c := range cmd.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

// unknownFlag reports the first flag among words, read as pflag reads them,
// that cmd does not define, in pflag's own wording. Moving such a flag after
// the path would not make the command valid, so it is reported as unknown
// rather than as misplaced.
func unknownFlag(cmd *cobra.Command, words []string) error {
	for i := 0; i < len(words); i++ {
		word := words[i]
		if word == "-" || !strings.HasPrefix(word, "-") {
			continue
		}
		name, short, inline := splitFlag(word)
		if name == "help" && !short || name == "h" && short {
			continue
		}
		found, value := lookupFlag(cmd, name, short)
		switch {
		case !found && short:
			return &tfserr.Error{Category: tfserr.Config, Message: fmt.Sprintf("unknown shorthand flag: '%s' in %s", name, word)}
		case !found:
			return &tfserr.Error{Category: tfserr.Config, Message: "unknown flag: --" + name}
		case value && !inline:
			i++
		}
	}
	return nil
}

// splitFlag takes a flag token apart: "--name", "--name=value", "-n", or
// "-nvalue". inline reports whether the token carries its value.
func splitFlag(arg string) (name string, short, inline bool) {
	if body, ok := strings.CutPrefix(arg, "--"); ok {
		name, _, inline = strings.Cut(body, "=")
		return name, false, inline
	}
	body := strings.TrimPrefix(arg, "-")
	if len(body) > 1 {
		return body[:1], true, true
	}
	return body, true, false
}

// lookupFlag finds a flag of cmd, its own or inherited, by name or by
// shorthand, and reports whether it takes a value.
func lookupFlag(cmd *cobra.Command, name string, short bool) (found, takesValue bool) {
	local, inherited := cmd.Flags(), cmd.InheritedFlags()
	lookup, fallback := local.Lookup, inherited.Lookup
	if short {
		lookup, fallback = local.ShorthandLookup, inherited.ShorthandLookup
	}
	f := lookup(name)
	if f == nil {
		f = fallback(name)
	}
	if f == nil {
		return false, false
	}
	return true, f.NoOptDefVal == ""
}

// shellJoin joins the arguments into a command line. A word with a character
// outside a safe set is put in single quotes, which bash and PowerShell read
// alike; a word that itself contains a single quote is put in double quotes
// instead.
func shellJoin(args []string) string {
	quoted := make([]string, len(args))
	for i, a := range args {
		switch {
		case a != "" && strings.Trim(a, shellSafe) == "":
		case !strings.Contains(a, "'"):
			a = "'" + a + "'"
		default:
			a = strconv.Quote(a)
		}
		quoted[i] = a
	}
	return strings.Join(quoted, " ")
}

// shellSafe holds the characters that need no quoting in a shell word.
const shellSafe = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_@%+=:,./-"
