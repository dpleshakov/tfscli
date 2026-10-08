package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dpleshakov/tfscli/internal/tfserr"
)

// checkFlagOrder refuses a flag placed before the last word of the command
// path, such as --verbose in "tfscli --verbose wit work-items get 1". Agent
// permission rules match the command text by its prefix, so the command path
// has to open the command line; cobra itself accepts flags at any position.
//
// The path is found by walking the arguments from the root rather than with
// cobra's Find. Find decides whether a flag takes a value only from the flags
// of the command it stands on, so a flag unknown there — --help before cobra
// registers it, or a flag of an action placed at the level of its group —
// swallows the next word, and Find stops short of the real command.
func checkFlagOrder(root *cobra.Command, args []string) error {
	cmd := root
	last := -1
	inPath := map[int]bool{}
	pendingValue := false
	for i, arg := range args {
		if arg == "--" {
			break
		}
		if strings.HasPrefix(arg, "-") && arg != "-" {
			name, short, inline := splitFlag(arg)
			pendingValue = !inline && takesValue(cmd, name, short)
			continue
		}
		if pendingValue {
			pendingValue = false
			continue
		}
		next := subcommand(cmd, arg)
		if next == nil {
			break
		}
		cmd, last, inPath[i] = next, i, true
	}
	if last < 0 {
		return nil
	}

	var misplaced, moved []string
	for i, arg := range args[:last] {
		if inPath[i] {
			continue
		}
		moved = append(moved, arg)
		if strings.HasPrefix(arg, "-") {
			misplaced = append(misplaced, strings.SplitN(arg, "=", 2)[0])
		}
	}
	if len(moved) == 0 {
		return nil
	}

	for _, flag := range misplaced {
		if flag == "--version" {
			return &tfserr.Error{
				Category: tfserr.Config,
				Message:  fmt.Sprintf("--version belongs to tfscli itself, not to %q (run it as: tfscli --version)", cmd.CommandPath()),
			}
		}
	}

	verb := "comes"
	if len(misplaced) > 1 {
		verb = "come"
	}
	corrected := append(append(strings.Fields(cmd.CommandPath()), moved...), args[last+1:]...)
	return &tfserr.Error{
		Category: tfserr.Config,
		Message: fmt.Sprintf("flags must follow the command %q, and %s %s before its end (run it as: %s)",
			cmd.CommandPath(), strings.Join(misplaced, ", "), verb, shellJoin(corrected)),
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
	sets := []*cobra.Command{cmd}
	for len(sets) > 0 {
		c := sets[0]
		sets = append(sets[1:], c.Commands()...)

		local, inherited := c.Flags(), c.InheritedFlags()
		f := local.Lookup(name)
		if f == nil {
			f = inherited.Lookup(name)
		}
		if short {
			f = local.ShorthandLookup(name)
			if f == nil {
				f = inherited.ShorthandLookup(name)
			}
		}
		if f != nil {
			return f.NoOptDefVal == ""
		}
	}
	return false
}

// shellJoin joins the arguments into a command line, quoting those a shell
// would split or drop.
func shellJoin(args []string) string {
	quoted := make([]string, len(args))
	for i, a := range args {
		if a == "" || strings.ContainsAny(a, " \t\"'") {
			a = strconv.Quote(a)
		}
		quoted[i] = a
	}
	return strings.Join(quoted, " ")
}
