package cli

import (
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"

	"github.com/dpleshakov/tfscli/internal/tfserr"
)

// refused is the error for a command line whose command path does not come
// first.
func refused(command, corrected string) string {
	return `Error [config]: the command "` + command + `" must come first, before its flags and arguments (run it as: ` + corrected + ")\n"
}

// unordered is the error for a command line whose command path does not come
// first and that has a word no shell quoting reads alike in bash and
// PowerShell.
func unordered(command string) string {
	return `Error [config]: the command "` + command + `" must come first, before its flags and arguments (move it to the start and keep every other word in its order)` + "\n"
}

// A command path that does not open the command line would slip past an agent
// permission rule written as a prefix, so it is refused before any request.
func TestFlagBeforeTheCommandPathIsRefused(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "persistent flag at the root",
			args: []string{"--verbose", "wit", "work-items", "get", "-p", "MyProject", "12345"},
			want: refused("tfscli wit work-items get", "tfscli wit work-items get --verbose -p MyProject 12345"),
		},
		{
			name: "flag with a separate value between path words",
			args: []string{"wit", "--api-version", "7.1", "work-items", "get", "-p", "MyProject", "12345"},
			want: refused("tfscli wit work-items get", "tfscli wit work-items get --api-version 7.1 -p MyProject 12345"),
		},
		{
			name: "flag with an inline value and a quoted argument",
			args: []string{"--api-version=7.1", "wit", "wiql", "query-by-wiql", "--query", "SELECT [System.Id] FROM WorkItems"},
			want: refused("tfscli wit wiql query-by-wiql", "tfscli wit wiql query-by-wiql --api-version=7.1 --query 'SELECT [System.Id] FROM WorkItems'"),
		},
		{
			name: "flags of the action before its name",
			args: []string{"wit", "work-items", "-p", "MyProject", "--verbose", "get", "12345"},
			want: refused("tfscli wit work-items get", "tfscli wit work-items get -p MyProject --verbose 12345"),
		},
		{
			name: "shorthand with an attached value",
			args: []string{"wit", "work-items", "-pMyProject", "get", "12345"},
			want: refused("tfscli wit work-items get", "tfscli wit work-items get -pMyProject 12345"),
		},
		{
			name: "shorthand with a value after an equals sign",
			args: []string{"wit", "work-items", "-p=MyProject", "get", "12345"},
			want: refused("tfscli wit work-items get", "tfscli wit work-items get -p=MyProject 12345"),
		},
		{
			name: "long help flag before the path",
			args: []string{"--help", "wit", "work-items", "get"},
			want: refused("tfscli wit work-items get", "tfscli wit work-items get --help"),
		},
		{
			name: "short help flag before the path",
			args: []string{"-h", "wit", "work-items", "get"},
			want: refused("tfscli wit work-items get", "tfscli wit work-items get -h"),
		},
		{
			name: "help flag between path words",
			args: []string{"wit", "--help", "work-items", "get"},
			want: refused("tfscli wit work-items get", "tfscli wit work-items get --help"),
		},
		{
			name: "help flag followed by a stray word",
			args: []string{"--help", "X", "wit", "work-items", "get"},
			want: refused("tfscli wit work-items get", "tfscli wit work-items get --help X"),
		},
		{
			name: "boolean flag of the action at the level of its group",
			args: []string{"wit", "wiql", "--time-precision", "query-by-wiql", "--query", "x"},
			want: refused("tfscli wit wiql query-by-wiql", "tfscli wit wiql query-by-wiql --time-precision --query x"),
		},
		{
			name: "value that names a subcommand",
			args: []string{"wit", "work-items", "-p", "get", "get", "12345"},
			want: refused("tfscli wit work-items get", "tfscli wit work-items get -p get 12345"),
		},
		{
			name: "value that starts with a dash",
			args: []string{"wit", "work-items", "--fields", "-p", "get", "5", "-p", "MyProject"},
			want: refused("tfscli wit work-items get", "tfscli wit work-items get --fields -p 5 -p MyProject"),
		},
		{
			name: "lone dash before the path",
			args: []string{"-", "wit", "work-items", "get"},
			want: refused("tfscli wit work-items get", "tfscli wit work-items get -"),
		},
		{
			name: "empty word before the path",
			args: []string{"", "wit", "work-items", "get", "5"},
			want: refused("tfscli wit work-items get", "tfscli wit work-items get '' 5"),
		},
		{
			name: "word a shell would interpret",
			args: []string{"--verbose", "wit", "wiql", "query-by-wiql", "--team", "R&D", "--query", "x"},
			want: refused("tfscli wit wiql query-by-wiql", "tfscli wit wiql query-by-wiql --verbose --team 'R&D' --query x"),
		},
		{
			name: "word with a single quote",
			args: []string{"--verbose", "wit", "wiql", "query-by-wiql", "--query", "it's"},
			want: refused("tfscli wit wiql query-by-wiql", `tfscli wit wiql query-by-wiql --verbose --query "it's"`),
		},
		{
			name: "word with a single quote and a backslash",
			args: []string{"--verbose", "wit", "wiql", "query-by-wiql", "--query", `UNDER 'Proj\Team'`},
			want: unordered("tfscli wit wiql query-by-wiql"),
		},
		{
			name: "word with a single and a double quote",
			args: []string{"--verbose", "wit", "wiql", "query-by-wiql", "--query", `'a' "b"`},
			want: unordered("tfscli wit wiql query-by-wiql"),
		},
		{
			name: "word with a single quote and a dollar sign",
			args: []string{"--verbose", "wit", "wiql", "query-by-wiql", "--query", "'$x'"},
			want: unordered("tfscli wit wiql query-by-wiql"),
		},
		{
			name: "word with a typographic single quote",
			args: []string{"--verbose", "wit", "wiql", "query-by-wiql", "--query", "it’s"},
			want: refused("tfscli wit wiql query-by-wiql", `tfscli wit wiql query-by-wiql --verbose --query "it’s"`),
		},
		{
			name: "word with a line break",
			args: []string{"--verbose", "wit", "wiql", "query-by-wiql", "--query", "SELECT [System.Id]\nFROM WorkItems"},
			want: unordered("tfscli wit wiql query-by-wiql"),
		},
		{
			name: "word that PowerShell would splat",
			args: []string{"--verbose", "wit", "wiql", "query-by-wiql", "--query", "@x"},
			want: refused("tfscli wit wiql query-by-wiql", "tfscli wit wiql query-by-wiql --verbose --query '@x'"),
		},
		{
			name: "flag before help",
			args: []string{"--verbose", "help", "wit"},
			want: refused("tfscli help", "tfscli help --verbose wit"),
		},
		{
			name: "flag before completion",
			args: []string{"--verbose", "completion", "powershell"},
			want: refused("tfscli completion powershell", "tfscli completion powershell --verbose"),
		},
		{
			name: "flag before a shell completion request",
			args: []string{"--verbose", "__complete", "wit", ""},
			want: refused("tfscli __complete", "tfscli __complete --verbose wit ''"),
		},
		{
			name: "double dash taken for the value of a flag",
			args: []string{"--api-version", "--", "completion", "bash"},
			want: refused("tfscli completion bash", "tfscli completion bash --api-version --"),
		},
		{
			name: "path words in reverse order",
			args: []string{"-", "login", "auth"},
			want: refused("tfscli auth login", "tfscli auth login -"),
		},
		{
			name: "double dash before the path",
			args: []string{"--", "wit", "work-items", "get", "5"},
			want: refused("tfscli wit work-items get", "tfscli wit work-items get -- 5"),
		},
		{
			name: "unknown flag before a path found in a second pass",
			args: []string{"work-items", "wit", "--nope", "get", "5"},
			want: "Error [config]: unknown flag: --nope\n",
		},
		{
			name: "version flag before a path found in a second pass",
			args: []string{"work-items", "wit", "--version", "get", "5"},
			want: `Error [config]: --version belongs to tfscli itself, not to "tfscli wit work-items get" (run it as: tfscli --version)` + "\n",
		},
		{
			name: "long flag without a name",
			args: []string{"--=x", "wit", "work-items", "get", "5"},
			want: "Error [config]: bad flag syntax: --=x\n",
		},
		{
			name: "long flag with three dashes",
			args: []string{"---x", "wit", "work-items", "get", "5"},
			want: "Error [config]: bad flag syntax: ---x\n",
		},
		{
			name: "version flag before a path",
			args: []string{"--version", "wit"},
			want: `Error [config]: --version belongs to tfscli itself, not to "tfscli wit" (run it as: tfscli --version)` + "\n",
		},
		{
			name: "flag of the action before a mistyped action",
			args: []string{"-p", "MyProject", "wit", "work-items", "gte", "1"},
			want: refused("tfscli wit work-items", "tfscli wit work-items -p MyProject gte 1"),
		},
		{
			name: "unknown flag before the path",
			args: []string{"--nope", "wit", "work-items", "get", "-p", "MyProject", "12345"},
			want: "Error [config]: unknown flag: --nope\n",
		},
		{
			name: "unknown flag swallowing a stray word",
			args: []string{"--nope", "X", "wit", "work-items", "get", "-p", "MyProject", "12345"},
			want: "Error [config]: unknown flag: --nope\n",
		},
		{
			name: "unknown shorthand before the path",
			args: []string{"-z", "wit", "work-items", "get", "-p", "MyProject", "12345"},
			want: "Error [config]: unknown shorthand flag: 'z' in -z\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newServer(t, http.StatusOK, workItemResponse)

			stdout, stderr, code := execute(t, s, tt.args...)

			if code != 1 {
				t.Errorf("exit code = %d, want 1", code)
			}
			if stderr != tt.want {
				t.Errorf("stderr = %q, want %q", stderr, tt.want)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want nothing", stdout)
			}
			if s.calls != 0 || s.options != 0 {
				t.Errorf("the server was called %d times and asked for options %d times, want no request at all", s.calls, s.options)
			}
		})
	}
}

// Help, the version, and cobra's own commands work where the rule puts their
// flags: after the command path, or with no path at all.
func TestHelpAndVersionAfterThePath(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{args: []string{"--version"}, want: "tfscli dev (unknown)"},
		{args: []string{"--help"}, want: "Available Commands:"},
		{args: []string{"-h"}, want: "Available Commands:"},
		{args: []string{"wit", "--help"}, want: "Available Commands:"},
		{args: []string{"wit", "work-items", "--help"}, want: "Available Commands:"},
		{args: []string{"wit", "work-items", "get", "--help"}, want: "--fields"},
		{args: []string{"wit", "wiql", "query-by-wiql", "-h"}, want: "--time-precision"},
		{args: []string{"help", "wit", "work-items", "get"}, want: "--fields"},
		{args: []string{"completion", "powershell"}, want: "Register-ArgumentCompleter"},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			stdout, stderr, code := execute(t, nil, tt.args...)

			if code != 0 {
				t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
			}
			if stderr != "" {
				t.Errorf("stderr = %q, want nothing", stderr)
			}
			if !strings.Contains(stdout, tt.want) {
				t.Errorf("stdout does not contain %q:\n%s", tt.want, stdout)
			}
		})
	}
}

// The completion scripts ask for completions with cobra's hidden __complete
// command followed by the words typed so far; such a request runs no command,
// so the words after it need not start with a command path.
func TestShellCompletionRequest(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{args: []string{"__complete", "wit", ""}, want: "work-items\t"},
		{args: []string{"__completeNoDesc", "wit", "wiql", ""}, want: "query-by-wiql\n"},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			stdout, stderr, code := execute(t, nil, tt.args...)

			if code != 0 {
				t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
			}
			if !strings.Contains(stdout, tt.want) {
				t.Errorf("stdout does not contain %q:\n%s", tt.want, stdout)
			}
		})
	}
}

// The tests above compare the check with expectations written by hand, which
// miss the lines where cobra resolves a command differently from what the
// expectations assume. Here cobra itself is the oracle, over every line up to
// three words long built from the words that matter to it and a fixed-seed
// sample of longer lines:
//
//   - a line the check accepts makes cobra select a command whose path
//     occupies the first words of the line;
//   - the corrected line proposed for a misplaced path is accepted by the
//     check, cobra selects for it the command the message names, and cobra
//     rejects none of the flags that preceded the path;
//   - a line refused for a flag before the path is rejected by cobra too, with
//     the same message, once the path is moved to the front.
func TestFlagOrderAgreesWithCobra(t *testing.T) {
	words := oracleWords()
	var lines [][]string
	var build func(line []string)
	build = func(line []string) {
		lines = append(lines, line)
		if len(line) == 3 {
			return
		}
		for _, w := range words {
			build(append(slices.Clip(line), w))
		}
	}
	build(nil)
	rng := rand.New(rand.NewPCG(1, 2))
	for range 20000 {
		line := make([]string, 4+rng.IntN(4))
		for i := range line {
			line[i] = words[rng.IntN(len(words))]
		}
		lines = append(lines, line)
	}

	failures := make(chan string, len(lines))
	next := make(chan []string)
	var wg sync.WaitGroup
	for range runtime.GOMAXPROCS(0) {
		wg.Go(func() {
			// The check merges persistent flags into the tree it walks, so
			// each worker builds its own.
			root := newRoot(testBuild, noTerminal{}, io.Discard, io.Discard)
			for line := range next {
				if msg := checkAgainstCobra(root, line); msg != "" {
					failures <- msg
				}
			}
		})
	}
	for _, line := range lines {
		next <- line
	}
	close(next)
	wg.Wait()
	close(failures)

	var msgs []string
	for msg := range failures {
		msgs = append(msgs, msg)
	}
	slices.Sort(msgs)
	for i, msg := range msgs {
		if i == 20 {
			t.Errorf("%d further failures omitted", len(msgs)-i)
			break
		}
		t.Error(msg)
	}
}

// checkAgainstCobra checks one command line against the properties of
// TestFlagOrderAgreesWithCobra and describes the first one it breaks.
func checkAgainstCobra(root *cobra.Command, line []string) string {
	shown := fmt.Sprintf("tfscli %q", line)
	cmd, name, corrected, before := reorder(root, line)
	err := checkFlagOrder(root, line)
	switch {
	case corrected == nil && err != nil:
		return fmt.Sprintf("%s: refused although its path comes first: %v", shown, err)
	case corrected == nil:
		// Cobra serves a completion request by construction; executing it
		// would only print cobra's completion diagnostics to the real stderr.
		if len(line) > 0 && isCompletionRequest(line[0]) {
			return ""
		}
		got := selected(line)
		if path := strings.Fields(got)[1:]; len(path) > len(line) || !slices.Equal(path, line[:len(path)]) {
			return fmt.Sprintf("%s: accepted, but cobra selects %q", shown, got)
		}
		return ""
	case err == nil:
		return fmt.Sprintf("%s: accepted, but its path does not come first", shown)
	}

	var te *tfserr.Error
	if !errors.As(err, &te) {
		return fmt.Sprintf("%s: refused with an error of no category: %v", shown, err)
	}
	fixed := fmt.Sprintf("tfscli %q", corrected)
	moved := append(slices.Clip(corrected[:len(strings.Fields(name))-1]), before...)
	// A group ignores unknown flags, and the flags before a completion
	// request belong to the root, which is a group too.
	leaf := !cmd.HasSubCommands()

	if !strings.Contains(te.Message, "must come first") {
		if !leaf {
			return ""
		}
		// Cobra stops at the first flag value it cannot parse, which the
		// check does not read, so a later flag goes unreported.
		switch got := flagError(moved); {
		case strings.HasPrefix(got, invalidValue):
		case got == "":
			return fmt.Sprintf("%s: refused with %q, but cobra accepts the flags before the path", shown, te.Message)
		case got != te.Message && !strings.HasPrefix(te.Message, "--version"):
			return fmt.Sprintf("%s: refused with %q, but cobra rejects it with %q", shown, te.Message, got)
		}
		return ""
	}

	if err := checkFlagOrder(root, corrected); err != nil {
		return fmt.Sprintf("%s: corrected to %s, which is refused: %v", shown, fixed, err)
	}
	if cmd != root {
		if got := selected(corrected); got != name {
			return fmt.Sprintf("%s: corrected to %s, for which cobra selects %q instead of %q", shown, fixed, got, name)
		}
	}
	if !leaf {
		return ""
	}
	if got := flagError(moved); got != "" && !strings.HasPrefix(got, invalidValue) {
		return fmt.Sprintf("%s: corrected to %s, though cobra rejects a flag before the path: %s", shown, fixed, got)
	}
	return ""
}

// invalidValue opens cobra's message for a flag value it cannot parse.
const invalidValue = "invalid argument "

// selected returns the path, as typed, of the command cobra selects for args,
// whether it then runs the command, prints its help, or rejects its flags or
// arguments.
func selected(args []string) string {
	root := stubbedRoot()
	root.SetArgs(args)
	cmd, _ := root.ExecuteC()
	if cmd.HasParent() && cmd.CalledAs() != "" {
		return cmd.Parent().CommandPath() + " " + cmd.CalledAs()
	}
	return cmd.CommandPath()
}

// flagError returns the message with which cobra rejects a flag of args as
// unknown or malformed or its value as invalid, or "" if it rejects none.
func flagError(args []string) string {
	root := stubbedRoot()
	root.SetArgs(args)
	_, err := root.ExecuteC()
	if err == nil {
		return ""
	}
	for _, kind := range []string{"unknown flag: ", "unknown shorthand flag: ", "bad flag syntax: ", invalidValue} {
		if strings.HasPrefix(err.Error(), kind) {
			return err.Error()
		}
	}
	return ""
}

// stubbedRoot is the command tree with every action replaced by one that does
// nothing, so that cobra can execute any command line without a server.
func stubbedRoot() *cobra.Command {
	root := newRoot(testBuild, noTerminal{}, io.Discard, io.Discard)
	var stub func(cmd *cobra.Command)
	stub = func(cmd *cobra.Command) {
		cmd.Run = nil
		cmd.RunE = func(*cobra.Command, []string) error { return nil }
		for _, child := range cmd.Commands() {
			stub(child)
		}
	}
	stub(root)
	return root
}

// oracleWords is the alphabet of the generated command lines: every command
// name and every flag of the tree, a value flag also with its value attached,
// and the words that pflag and cobra treat specially.
func oracleWords() []string {
	words := []string{
		"--", "-", "", "--=X", "--help", "-h", "--version", "--bogus", "-z", "X", "5",
		cobra.ShellCompRequestCmd, cobra.ShellCompNoDescRequestCmd,
	}
	var walk func(cmd *cobra.Command)
	walk = func(cmd *cobra.Command) {
		if cmd.HasParent() {
			words = append(words, cmd.Name())
		}
		for _, m := range flagUsage.FindAllStringSubmatch(cmd.LocalFlags().FlagUsages(), -1) {
			short, name := m[1], m[2]
			_, value := lookupFlag(cmd, name, false)
			words = append(words, "--"+name)
			if value {
				words = append(words, "--"+name+"=X")
			}
			if short != "" {
				words = append(words, "-"+short)
				if value {
					words = append(words, "-"+short+"X", "-"+short+"=X")
				}
			}
		}
		for _, child := range cmd.Commands() {
			walk(child)
		}
	}
	walk(newRoot(testBuild, noTerminal{}, io.Discard, io.Discard))
	slices.Sort(words)
	return slices.Compact(words)
}

// flagUsage matches the start of a line of cobra's flag usage text, such as
// "  -p, --project string" or "      --verbose", capturing the shorthand and
// the name. The flags are read from the usage text because visiting the flag
// set would name pflag's Flag type and so make pflag a direct dependency;
// a hidden flag is not listed there and therefore not exercised.
var flagUsage = regexp.MustCompile(`(?m)^\s+(?:-(\w), )?--([\w-]+)`)
