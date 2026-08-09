package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dpleshakov/tfscli/internal/tfserr"
	"github.com/dpleshakov/tfscli/internal/workitem"
)

func newWorkItemCmd(g *globals) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workitem",
		Short: "Work item commands",
	}
	cmd.AddCommand(newWorkItemGetCmd(g))
	return cmd
}

func newWorkItemGetCmd(g *globals) *cobra.Command {
	var (
		project string
		fields  string
	)

	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Print one work item as markdown",
		Long: "Get one work item by id.\n\n" +
			"All fields are printed unless --fields narrows the request; the fields are\n" +
			"then printed in the order they were asked for. HTML fields (Description,\n" +
			"Repro Steps, System Info, Acceptance Criteria) are converted to markdown.",
		Example: "  tfscli workitem get -p MyProject 12345\n" +
			"  tfscli workitem get -p MyProject 12345 --fields System.Title,System.State,System.Description",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}

			ov := g.overrides(cmd)
			if cmd.Flags().Changed("project") {
				ov.Project = &project
			}
			cfg, client, err := g.connect(ov)
			if err != nil {
				return err
			}
			if err := cfg.RequireProject(); err != nil {
				return err
			}

			wi, err := workitem.Get(cmd.Context(), client, cfg.Project, id, splitFields(fields))
			if err != nil {
				return err
			}
			return printWorkItem(g.stdout, wi)
		},
	}

	cmd.Flags().StringVarP(&project, "project", "p", "", "team project (TFSCLI_PROJECT; required unless set in the config file)")
	cmd.Flags().StringVar(&fields, "fields", "", "comma-separated field names, e.g. System.Title,System.State (default: all fields)")
	return cmd
}

// parseID rejects anything that is not a positive work item id before a request
// is made, so that a typo is answered locally rather than as an HTTP 400.
func parseID(arg string) (int, error) {
	id, err := strconv.Atoi(arg)
	if err != nil || id <= 0 {
		return 0, &tfserr.Error{
			Category: tfserr.Config,
			Message:  fmt.Sprintf("work item id %q is not a positive integer", arg),
		}
	}
	return id, nil
}

// splitFields turns the --fields value into the list workitem.Get expects.
// Surrounding spaces are tolerated so that a quoted "System.Title, System.State"
// works, and empty entries are dropped so that a stray comma is harmless.
func splitFields(value string) []string {
	var fields []string
	for name := range strings.SplitSeq(value, ",") {
		if name = strings.TrimSpace(name); name != "" {
			fields = append(fields, name)
		}
	}
	return fields
}
