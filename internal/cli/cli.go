// Package cli owns the cobra command tree, persistent flags, and the
// markdown printers. It orchestrates the chain config → apiclient →
// domain → printer → exit.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/dpleshakov/tfscli/internal/apiclient"
	"github.com/dpleshakov/tfscli/internal/config"
	"github.com/dpleshakov/tfscli/internal/log"
	"github.com/dpleshakov/tfscli/internal/tfserr"
)

// Run executes the root command and returns the process exit code. The version
// and commit are stamped into the binary at build time and reported by
// --version.
func Run(version, commit string) int {
	return run(build{version: version, commit: commit}, os.Args[1:], newStdinPrompter(), os.Stdout, os.Stderr)
}

// build is the version stamp linked into the binary.
type build struct {
	version string
	commit  string
}

// run is Run with the process environment passed in, so that tests can drive
// the whole command tree and read what it wrote.
func run(b build, args []string, stdin prompter, stdout, stderr io.Writer) int {
	// Ctrl-C cancels the request in flight; apiclient reports the canceled
	// round trip as category network instead of a Go panic trace.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	root := newRoot(b, stdin, stdout, stderr)
	root.SetArgs(args)

	if err := checkFlagOrder(root, args); err != nil {
		tfserr.Print(err, stderr)
		return tfserr.ExitCode(err)
	}
	if err := root.ExecuteContext(ctx); err != nil {
		tfserr.Print(categorized(err), stderr)
		return tfserr.ExitCode(err)
	}
	return 0
}

// globals holds the state shared by every command: the values of the
// persistent flags and the standard streams.
type globals struct {
	stdin  prompter
	stdout io.Writer
	stderr io.Writer

	verbose    bool
	apiVersion string
}

func newRoot(b build, stdin prompter, stdout, stderr io.Writer) *cobra.Command {
	g := &globals{stdin: stdin, stdout: stdout, stderr: stderr}

	root := &cobra.Command{
		Use:   "tfscli",
		Short: "Read-only access to on-premises TFS / Azure DevOps Server",
		Long: "tfscli reads TFS / Azure DevOps Server through its REST API and prints the\n" +
			"result as markdown. It is stateless: every call hits the server.\n\n" +
			"The server URL, the collection, and the personal access token are stored\n" +
			"together by \"tfscli auth login\" in $XDG_DATA_HOME/tfscli/auth.json (by\n" +
			"default ~/.local/share/tfscli/auth.json), or supplied as the same JSON in\n" +
			"TFSCLI_AUTH.\n\n" +
			"The other settings are read from $XDG_CONFIG_HOME/tfscli/config.json (by\n" +
			"default ~/.config/tfscli/config.json), which is optional, and can be\n" +
			"overridden by the TFSCLI_* environment variables and by the flags below, in\n" +
			"that order of precedence (flag wins).",
		SilenceUsage:  true,
		SilenceErrors: true,
		// A non-empty Version makes cobra add --version on its own. The
		// shorthand is left off: -v belongs to no flag here, and reserving it
		// for the version would collide with --verbose next to it.
		Version: b.version + " (" + b.commit + ")",
	}
	root.SetVersionTemplate("tfscli {{.Version}}\n")
	// Errors are printed by run in the contract format; cobra must not write
	// its own message or dump the usage text on top of it.
	root.SetOut(stdout)
	root.SetErr(stderr)

	f := root.PersistentFlags()
	f.BoolVar(&g.verbose, "verbose", false, "log every request to stderr (also TFSCLI_VERBOSE=1)")
	f.StringVar(&g.apiVersion, "api-version", "", "REST API version (TFSCLI_API_VERSION; by default it is negotiated with the server)")

	root.AddCommand(newAuthCmd(g))
	root.AddCommand(newWitCmd(g))
	root.AddCommand(newLicensesCmd(g))
	// Cobra adds help and completion only when it executes the tree; adding
	// them here lets checkFlagOrder walk them like any other command.
	root.InitDefaultHelpCmd()
	root.InitDefaultCompletionCmd()
	completion, _, _ := root.Find([]string{"completion"})
	makeGroup(completion)
	return root
}

// newGroupCmd builds a command that only holds subcommands, such as an API
// area or a resource.
func newGroupCmd(use, short string) *cobra.Command {
	cmd := &cobra.Command{Use: use, Short: short}
	makeGroup(cmd)
	return cmd
}

// makeGroup makes cmd, a command that only holds subcommands, reject an
// unknown subcommand. Cobra does so only on the root; below it, a group
// without RunE prints its help and exits 0, so a mistyped resource would pass
// for success. Here any argument left over after the subcommand lookup is
// reported as an unknown command instead. Unknown flags are ignored at this
// level so that the flags meant for the action do not hide the mistyped name.
// It also serves cobra's own completion group.
func makeGroup(cmd *cobra.Command) {
	cmd.FParseErrWhitelist = cobra.FParseErrWhitelist{UnknownFlags: true}
	// Cobra applies its default distance only when it looks up a suggestion
	// itself, which it does on the root alone.
	cmd.SuggestionsMinimumDistance = 2
	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return nil
		}
		next := fmt.Sprintf("run %q for the available commands", cmd.CommandPath()+" --help")
		if suggestions := cmd.SuggestionsFor(args[0]); len(suggestions) > 0 {
			next = fmt.Sprintf("did you mean %q?", suggestions[0])
		}
		return fmt.Errorf("unknown command %q for %q (%s)", args[0], cmd.CommandPath(), next)
	}
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return cmd.Help()
	}
}

// overrides collects the persistent flags the user actually typed. A flag left
// alone must not override the environment or the config file, which is what
// Changed distinguishes.
func (g *globals) overrides(cmd *cobra.Command) config.Overrides {
	var ov config.Overrides
	if cmd.Flags().Changed("api-version") {
		ov.APIVersion = &g.apiVersion
	}
	return ov
}

// connect resolves the effective configuration and builds a client from it.
func (g *globals) connect(ov config.Overrides) (*config.Config, *apiclient.Client, error) {
	path, err := config.DefaultPath()
	if err != nil {
		return nil, nil, err
	}
	authPath, err := config.DefaultAuthPath()
	if err != nil {
		return nil, nil, err
	}
	cfg, err := config.Load(path, authPath, ov)
	if err != nil {
		return nil, nil, err
	}
	client, err := apiclient.New(cfg, g.logger())
	if err != nil {
		return nil, nil, err
	}
	return cfg, client, nil
}

// logger returns the stderr logger when verbose output is requested by flag or
// environment, and the discarding one otherwise.
func (g *globals) logger() log.Logger {
	if g.verbose || os.Getenv("TFSCLI_VERBOSE") == "1" {
		return log.New(g.stderr)
	}
	return log.Noop()
}

// categorized makes sure everything reaching stderr carries a category. Errors
// raised by cobra itself — an unknown flag, a missing argument — are about how
// the command was invoked, which is the config category.
func categorized(err error) error {
	var te *tfserr.Error
	if err == nil || errors.As(err, &te) {
		return err
	}
	return &tfserr.Error{
		Category: tfserr.Config,
		Message:  err.Error(),
		Cause:    err,
	}
}
