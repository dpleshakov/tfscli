package cli

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/spf13/cobra"

	"github.com/dpleshakov/tfscli/internal/tfserr"
)

// checkFlagOrder refuses a command line whose command path does not occupy its
// first words, such as "tfscli --verbose wit work-items get 1". Agent
// permission rules match the command text by its prefix, so the command path
// has to open the command line; cobra itself accepts flags at any position.
//
// The command is the one reached from the root by taking, in order, every word
// that names a subcommand of the command reached so far. The walk does not
// stop at "--", since pflag takes it for the value of a flag placed before it.
// No knowledge of flag values is needed: a group takes no arguments and
// defines no flags of its own (TestEveryGroupOnlyGroups), and the flags cobra
// adds itself, --help and the root's --version, take no value, so in a valid
// command line no word naming a subcommand can follow the path. This reaches
// at least as deep as cobra's Find, which descends only through such words,
// so cobra never runs a command whose path is not at the start.
//
// Cobra's hidden command for shell completion requests, which it adds to the
// root only when it executes the tree, is taken as a subcommand of the root
// that ends the path: the words after it are the command line to complete.
func checkFlagOrder(root *cobra.Command, args []string) error {
	cmd, name, corrected, before := reorder(root, args)
	if corrected == nil {
		return nil
	}

	// A group ignores the flags it does not know, which before a group are
	// usually those of the intended command placed before a mistyped name;
	// the corrected command lets cobra report that name instead.
	if err := unknownFlag(cmd, name, before); err != nil && (!cmd.HasSubCommands() || errors.Is(err, errVersionFlag)) {
		return err
	}
	next := "move it to the start and keep every other word in its order"
	if line, ok := shellJoin(append([]string{root.Name()}, corrected...)); ok {
		next = "run it as: " + line
	}
	return &tfserr.Error{
		Category: tfserr.Config,
		Message:  fmt.Sprintf("the command %q must come first, before its flags and arguments (%s)", name, next),
	}
}

// reorder finds the command that args name, as checkFlagOrder describes, and
// returns it with its name as typed; for a shell completion request, cmd is
// the root, whose flags are the ones that can precede it. When the path does
// not occupy the first words, corrected is args with the path moved to the
// front and every other word kept in its original order, and before holds
// the words of args that preceded the last word of the path; otherwise both
// are nil.
//
// Moving the path to the front can turn a word that preceded one of its steps
// into a further step, as "login" in "tfscli - login auth", so the move is
// repeated until the path stays where it is; it lengthens the path each time.
func reorder(root *cobra.Command, args []string) (cmd *cobra.Command, name string, corrected, before []string) {
	// order holds the indices in args of the words of the line being read.
	order := make([]int, len(args))
	for i := range order {
		order[i] = i
	}
	var path []int
	for {
		line := make([]string, len(order))
		for i, o := range order {
			line[i] = args[o]
		}
		cmd, name, path = findPath(root, line)
		if len(path) == 0 || path[len(path)-1] == len(path)-1 {
			if slices.IsSorted(order) {
				return cmd, name, nil, nil
			}
			corrected = line
			break
		}
		next := make([]int, 0, len(order))
		for _, p := range path {
			next = append(next, order[p])
		}
		for i, o := range order {
			if !slices.Contains(path, i) {
				next = append(next, o)
			}
		}
		order = next
	}

	last := slices.Max(order[:len(path)])
	for _, o := range order[len(path):] {
		if o < last {
			before = append(before, args[o])
		}
	}
	return cmd, name, corrected, before
}

// findPath walks line from the root as checkFlagOrder describes and returns
// the command reached, its name as typed, and the indices in line of the
// words of its path.
func findPath(root *cobra.Command, line []string) (cmd *cobra.Command, name string, path []int) {
	cmd = root
	words := []string{root.Name()}
	for i, arg := range line {
		if cmd == root && (arg == cobra.ShellCompRequestCmd || arg == cobra.ShellCompNoDescRequestCmd) {
			words = append(words, arg)
			path = append(path, i)
			break
		}
		if next := subcommand(cmd, arg); next != nil {
			cmd = next
			words = append(words, arg)
			path = append(path, i)
		}
	}
	return cmd, strings.Join(words, " "), path
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
// that cmd, named name, does not define or that is malformed, in pflag's own
// wording. Moving such a flag after the path would not make the command
// valid, so it is reported as it is rather than as misplaced. --version,
// which cobra defines on the root alone, is reported with the way to run it.
// Words after "--" are arguments, as they are to pflag.
func unknownFlag(cmd *cobra.Command, name string, words []string) error {
	for i := 0; i < len(words); i++ {
		word := words[i]
		switch {
		case word == "--":
			return nil
		case word == "-" || !strings.HasPrefix(word, "-"):
			continue
		case strings.HasPrefix(word, "--"):
			body := word[2:]
			if body[0] == '-' || body[0] == '=' {
				return &tfserr.Error{Category: tfserr.Config, Message: "bad flag syntax: " + word}
			}
			flag, _, inline := strings.Cut(body, "=")
			if flag == "help" {
				continue
			}
			found, value := lookupFlag(cmd, flag, false)
			switch {
			case !found && flag == "version":
				return &tfserr.Error{
					Category: tfserr.Config,
					Message:  fmt.Sprintf("--version belongs to tfscli itself, not to %q (run it as: tfscli --version)", name),
					Cause:    errVersionFlag,
				}
			case !found:
				return &tfserr.Error{Category: tfserr.Config, Message: "unknown flag: --" + flag}
			}
			if value && !inline {
				i++
			}
		default:
			takesNext, err := shorthands(cmd, word[1:])
			if err != nil {
				return err
			}
			if takesNext {
				i++
			}
		}
	}
	return nil
}

// errVersionFlag is the cause of the error for --version placed before a
// command path.
var errVersionFlag = errors.New("--version before a command path")

// shorthands reads a group of shorthand flags, such as "v" in "-v" or "vpX"
// in "-vpX", as pflag reads it, and reports whether the last flag of the group
// takes its value from the next word. The help flag, which cobra adds only
// when it executes the tree, takes no value.
func shorthands(cmd *cobra.Command, group string) (takesNext bool, err error) {
	for ; group != ""; group = group[1:] {
		if group[0] == 'h' {
			continue
		}
		found, value := lookupFlag(cmd, group[:1], true)
		switch {
		case !found:
			return false, &tfserr.Error{
				Category: tfserr.Config,
				Message:  fmt.Sprintf("unknown shorthand flag: %q in -%s", group[0], group),
			}
		case len(group) > 1 && group[1] == '=':
			return false, nil
		case value:
			return len(group) == 1, nil
		}
	}
	return false, nil
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

// shellJoin joins the arguments into a command line that bash and PowerShell
// read alike, and reports false when a word has no such form. A word with a
// character outside a safe set is put in single quotes, or, when it contains
// a single quote, in double quotes, which the two shells read alike only for a
// word free of the characters either of them interprets there. A control
// character, such as the line break of a multi-line query, is read alike in
// neither.
func shellJoin(args []string) (string, bool) {
	quoted := make([]string, len(args))
	for i, a := range args {
		switch {
		case strings.ContainsFunc(a, unicode.IsControl):
			return "", false
		case a != "" && strings.Trim(a, shellSafe) == "":
		case !strings.ContainsAny(a, singleQuotes):
			a = "'" + a + "'"
		case !strings.ContainsAny(a, doubleQuoted):
			a = `"` + a + `"`
		default:
			return "", false
		}
		quoted[i] = a
	}
	return strings.Join(quoted, " "), true
}

// The character sets of shellJoin. shellSafe holds the characters that need
// no quoting in a word, without "@" and "%", with which PowerShell splats a
// variable or stops parsing; singleQuotes the characters that end a single-quoted
// string in bash or PowerShell, which takes typographic quotes for ASCII ones,
// and doubleQuoted the characters that either shell interprets inside double
// quotes.
const (
	shellSafe    = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_+=:,./-"
	singleQuotes = "'‘’‚‛"
	doubleQuoted = "\"“”„$`\\!"
)
