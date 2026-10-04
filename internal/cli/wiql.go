package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dpleshakov/tfscli/internal/tfserr"
	"github.com/dpleshakov/tfscli/internal/wiql"
)

// exampleWiql is the query the errors and the help text show.
const exampleWiql = "SELECT [System.Id] FROM WorkItems WHERE [System.State] = 'Active'"

// newWiqlCmd is the resource Wiql of the area wit, named after the path of
// its operations in the REST API reference (wit/wiql).
func newWiqlCmd(g *globals) *cobra.Command {
	cmd := newGroupCmd("wiql", "WIQL: find work items with a query")
	cmd.AddCommand(newWiqlQueryByWiqlCmd(g))
	return cmd
}

// newWiqlQueryByWiqlCmd runs a WIQL query. The flags are the API parameters:
// the query is passed through as written, and --top and --time-precision are
// sent only when given, so that the server's defaults apply otherwise.
func newWiqlQueryByWiqlCmd(g *globals) *cobra.Command {
	var (
		project       string
		team          string
		query         string
		top           int
		timePrecision bool
	)

	cmd := &cobra.Command{
		Use:   "query-by-wiql --query <WIQL>",
		Short: "Find work items with a WIQL query and print their ids",
		Long: "Run a WIQL query with Query By Wiql. The query is passed to the server as\n" +
			"written. Without a project from any source the query runs at collection\n" +
			"level; --team narrows it to a team of the project, which @CurrentIteration\n" +
			"and similar macros need.\n\n" +
			"The server returns ids, not work items, so the command prints the ids: as a\n" +
			"comma-separated list for a flat query, and as one line per link for a tree\n" +
			"or one-hop query. Read the work items themselves with\n" +
			"`tfscli wit work-items list --ids <ids>`, at most 200 ids per call; use --top\n" +
			"when only the first results are needed.",
		Example: "  tfscli wit wiql query-by-wiql -p MyProject --query \"" + exampleWiql + "\"\n" +
			"  tfscli wit wiql query-by-wiql -p MyProject --team \"Web Team\" --top 20 --query \"SELECT [System.Id] FROM WorkItems WHERE [System.IterationPath] = @CurrentIteration\"",
		// A query given as an argument is the likely mistake; the error says
		// where it goes instead.
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) > 0 {
				return &tfserr.Error{
					Category: tfserr.Config,
					Message:  fmt.Sprintf("query-by-wiql takes no arguments; pass the query with --query, e.g. --query %q", strings.Join(args, " ")),
				}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(query) == "" {
				return &tfserr.Error{
					Category: tfserr.Config,
					Message:  fmt.Sprintf("no WIQL query given (pass it with --query, e.g. --query %q)", exampleWiql),
				}
			}

			ov := g.overrides(cmd)
			if cmd.Flags().Changed("project") {
				ov.Project = &project
			}
			cfg, client, err := g.connect(ov)
			if err != nil {
				return err
			}
			// The team follows the project in the path, so it cannot be used
			// on its own.
			if team != "" && cfg.Project == "" {
				return &tfserr.Error{
					Category: tfserr.Config,
					Message:  "--team needs a project, and the project is not set (pass -p, set TFSCLI_PROJECT, or add \"project\" to the config file)",
				}
			}

			req := wiql.Request{Project: cfg.Project, Team: team, Query: query}
			if cmd.Flags().Changed("top") {
				req.Top = &top
			}
			if cmd.Flags().Changed("time-precision") {
				req.TimePrecision = &timePrecision
			}

			result, err := wiql.QueryByWiql(cmd.Context(), client, req)
			if err != nil {
				return err
			}
			return printWiqlResult(g.stdout, result)
		},
	}

	cmd.Flags().StringVarP(&project, "project", "p", "", "team project (TFSCLI_PROJECT or the config file; without one the query runs at collection level)")
	cmd.Flags().StringVar(&team, "team", "", "team of the project to run the query for, e.g. for @CurrentIteration")
	cmd.Flags().StringVar(&query, "query", "", "the WIQL query, passed to the server as written (required)")
	cmd.Flags().IntVar(&top, "top", 0, "return at most this many results (default: the server's own limit)")
	cmd.Flags().BoolVar(&timePrecision, "time-precision", false, "use the time of day, not only the date, in date comparisons (server default: false)")
	return cmd
}
