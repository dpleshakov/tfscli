// Package cli owns the cobra command tree, persistent flags, and the
// markdown printers. It orchestrates the chain config → apiclient →
// domain → printer → exit.
package cli

import (
	"context"
	"errors"
	"io"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/dpleshakov/tfscli/internal/apiclient"
	"github.com/dpleshakov/tfscli/internal/config"
	"github.com/dpleshakov/tfscli/internal/log"
	"github.com/dpleshakov/tfscli/internal/tfserr"
)

// Run executes the root command and returns the process exit code.
func Run() int {
	return run(os.Args[1:], os.Stdout, os.Stderr)
}

// run is Run with the process environment passed in, so that tests can drive
// the whole command tree and read what it wrote.
func run(args []string, stdout, stderr io.Writer) int {
	// Ctrl-C cancels the request in flight; apiclient reports the cancelled
	// round trip as category network instead of a Go panic trace.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	root := newRoot(stdout, stderr)
	root.SetArgs(args)

	if err := root.ExecuteContext(ctx); err != nil {
		tfserr.Print(categorized(err), stderr)
		return tfserr.ExitCode(err)
	}
	return 0
}

// globals holds the state shared by every command: the values of the
// persistent flags and the streams to write to.
type globals struct {
	stdout io.Writer
	stderr io.Writer

	verbose    bool
	url        string
	collection string
	pat        string
	apiVersion string
}

func newRoot(stdout, stderr io.Writer) *cobra.Command {
	g := &globals{stdout: stdout, stderr: stderr}

	root := &cobra.Command{
		Use:   "tfscli",
		Short: "Read-only access to on-premises TFS / Azure DevOps Server",
		Long: "tfscli reads TFS / Azure DevOps Server through its REST API and prints the\n" +
			"result as markdown. It is stateless: every call hits the server.\n\n" +
			"Configuration is read from ~/.tfscli/config.json and can be overridden by\n" +
			"the TFSCLI_* environment variables and by the flags below, in that order of\n" +
			"precedence (flag wins).",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	// Errors are printed by run in the contract format; cobra must not write
	// its own message or dump the usage text on top of it.
	root.SetOut(stdout)
	root.SetErr(stderr)

	f := root.PersistentFlags()
	f.BoolVar(&g.verbose, "verbose", false, "log every request to stderr (also TFSCLI_VERBOSE=1)")
	f.StringVar(&g.url, "url", "", "server URL, e.g. https://tfs.company.com:8080/tfs (TFSCLI_URL)")
	f.StringVar(&g.collection, "collection", "", "collection name (TFSCLI_COLLECTION)")
	f.StringVar(&g.pat, "pat", "", "personal access token (TFSCLI_PAT)")
	f.StringVar(&g.apiVersion, "api-version", "", "REST API version (TFSCLI_API_VERSION, default "+config.DefaultAPIVersion+")")

	root.AddCommand(newWorkItemCmd(g))
	return root
}

// overrides collects the persistent flags the user actually typed. A flag left
// alone must not override the environment or the config file, which is what
// Changed distinguishes.
func (g *globals) overrides(cmd *cobra.Command) config.Overrides {
	var ov config.Overrides
	flags := cmd.Flags()
	if flags.Changed("url") {
		ov.URL = &g.url
	}
	if flags.Changed("collection") {
		ov.Collection = &g.collection
	}
	if flags.Changed("pat") {
		ov.PAT = &g.pat
	}
	if flags.Changed("api-version") {
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
	cfg, err := config.Load(path, ov)
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
