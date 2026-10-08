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
// Whatever the check cannot place is left to cobra: an unknown command, a
// flag the target command does not define (--help and --version among them,
// since cobra adds those only when it executes), the "--" terminator, and an
// argument that is neither a flag nor a word of the path.
func checkFlagOrder(root *cobra.Command, args []string) error {
	cmd, _, err := root.Find(args)
	if err != nil || cmd == root {
		return nil
	}
	path := strings.Fields(cmd.CommandPath())[1:]

	var misplaced, moved []string
	i, next := 0, 0
	for ; i < len(args) && next < len(path); i++ {
		arg := args[i]
		if arg == path[next] {
			next++
			continue
		}
		if arg == "--" || !strings.HasPrefix(arg, "-") {
			return nil
		}
		name, short, inline := splitFlag(arg)
		found, takesValue := lookupFlag(cmd, name, short)
		if !found {
			return nil
		}
		misplaced = append(misplaced, strings.SplitN(arg, "=", 2)[0])
		moved = append(moved, arg)
		if takesValue && !inline && i+1 < len(args) {
			i++
			moved = append(moved, args[i])
		}
	}
	if len(misplaced) == 0 {
		return nil
	}

	verb := "comes"
	if len(misplaced) > 1 {
		verb = "come"
	}
	corrected := append(append(append([]string{root.Name()}, path...), moved...), args[i:]...)
	return &tfserr.Error{
		Category: tfserr.Config,
		Message: fmt.Sprintf("flags must follow the command %q, and %s %s before its end (run it as: %s)",
			cmd.CommandPath(), strings.Join(misplaced, ", "), verb, shellJoin(corrected)),
	}
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

// lookupFlag finds a flag of cmd by name or shorthand, its own or inherited
// from a parent, and reports whether it takes a value: a flag without a value
// for its bare form, such as --project, consumes the next argument, while a
// boolean such as --verbose does not.
func lookupFlag(cmd *cobra.Command, name string, short bool) (found, takesValue bool) {
	local, inherited := cmd.Flags(), cmd.InheritedFlags()
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
	if f == nil {
		return false, false
	}
	return true, f.NoOptDefVal == ""
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
