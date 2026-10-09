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
// The command is located twice, and the deeper result is taken. Cobra's Find
// decides whether a flag takes a value only from the flags of the command it
// stands on, so a flag unknown there — a flag of an action placed at the level
// of its group — swallows the next word and Find stops at the group. The walk
// in walkPath looks flags up in the whole subtree instead, but it can stop
// where Find goes on: at a word Find skips, such as "" or "-", or at a word
// Find takes as the value of a flag it does not know, such as X in
// "--help X". Whichever command is deeper is the one cobra may run, and it is
// what the check holds to.
func checkFlagOrder(root *cobra.Command, args []string) error {
	target := walkPath(root, args)
	if found, _, err := root.Find(args); err == nil && depth(found) > depth(target) {
		target = found
	}
	path := strings.Fields(target.CommandPath())[1:]
	if len(path) == 0 || slices.Equal(args[:min(len(path), len(args))], path) {
		return nil
	}

	// The corrected command is the path followed by every other word in its
	// original order; before is the part of it that preceded the path.
	var before, rest []string
	next := 0
	for _, arg := range args {
		if next < len(path) && arg == path[next] {
			next++
			continue
		}
		if next < len(path) {
			before = append(before, arg)
		}
		rest = append(rest, arg)
	}

	if slices.Contains(before, "--version") {
		return &tfserr.Error{
			Category: tfserr.Config,
			Message:  fmt.Sprintf("--version belongs to tfscli itself, not to %q (run it as: tfscli --version)", target.CommandPath()),
		}
	}
	if err := unknownFlag(target, before); err != nil {
		return err
	}
	return &tfserr.Error{
		Category: tfserr.Config,
		Message: fmt.Sprintf("the command %q must come first, before its flags and arguments (run it as: %s)",
			target.CommandPath(), shellJoin(append(append([]string{root.Name()}, path...), rest...))),
	}
}

// walkPath follows the command path through the arguments from the root and
// returns the deepest command it reaches. A word that follows a flag taking a
// value is that value, whatever it starts with or names, as pflag reads it.
func walkPath(root *cobra.Command, args []string) *cobra.Command {
	cmd := root
	pendingValue := false
	for _, arg := range args {
		switch {
		case arg == "--":
			return cmd
		case pendingValue:
			pendingValue = false
		case strings.HasPrefix(arg, "-") && arg != "-":
			name, short, inline := splitFlag(arg)
			pendingValue = !inline && takesValue(cmd, name, short)
		default:
			next := subcommand(cmd, arg)
			if next == nil {
				return cmd
			}
			cmd = next
		}
	}
	return cmd
}

// depth is the number of words in the command path of cmd below the root.
func depth(cmd *cobra.Command) int {
	n := 0
	for ; cmd.HasParent(); cmd = cmd.Parent() {
		n++
	}
	return n
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

// subcommand returns the child of cmd named name, or nil.
func subcommand(cmd *cobra.Command, name string) *cobra.Command {
	for _, c := range cmd.Commands() {
		if c.Name() == name {
			return c
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

// takesValue reports whether the flag named consumes the next argument as its
// value: a flag without a value for its bare form, such as --project, does,
// while a boolean such as --verbose does not. The flag is looked up on cmd,
// among the flags it inherits, and on every command below it, since a flag of
// an action may be placed while the walk still stands on its group. --help,
// -h, and --version, which cobra registers only when it executes a command,
// take no value, and neither does a flag found nowhere.
func takesValue(cmd *cobra.Command, name string, short bool) bool {
	if name == "help" || name == "version" || (short && name == "h") {
		return false
	}
	queue := []*cobra.Command{cmd}
	for len(queue) > 0 {
		c := queue[0]
		queue = append(queue[1:], c.Commands()...)
		if found, value := lookupFlag(c, name, short); found {
			return value
		}
	}
	return false
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
