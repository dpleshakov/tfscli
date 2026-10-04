package cli

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dpleshakov/tfscli/internal/apiclient"
	"github.com/dpleshakov/tfscli/internal/tfserr"
	"github.com/dpleshakov/tfscli/internal/workitem"
)

func newWorkItemCmd(g *globals) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workitem",
		Short: "Work item commands",
	}
	cmd.AddCommand(newWorkItemGetCmd(g))
	cmd.AddCommand(newWorkItemListCmd(g))
	cmd.AddCommand(newWorkItemGetBatchCmd(g))
	return cmd
}

func newWorkItemGetBatchCmd(g *globals) *cobra.Command {
	return newWorkItemBatchCmd(g, batchCommand{
		use:   "get-batch",
		short: "Print several work items as markdown, for requests too long for list",
		long: "Get several work items by id with Get Work Items Batch, a POST request that\n" +
			"carries the ids and the other parameters in its body, so that it is not\n" +
			"limited by the length of the URL as workitem list is; at most 200 ids are\n" +
			"accepted by the server. It needs Azure DevOps Server 2019 or later; on an\n" +
			"older server use workitem list.\n\n",
		example: "  tfscli workitem get-batch -p MyProject --ids 297,299,300\n" +
			"  tfscli workitem get-batch -p MyProject --ids 297,299,300 --fields System.Title,System.State --error-policy omit",
		read: workitem.GetBatch,
		// A server older than 2019 has no such route and answers with a bare
		// 404, which says nothing of the version it needs.
		bareNotFound: "Get Work Items Batch needs Azure DevOps Server 2019 or later; on an older server use tfscli workitem list",
	})
}

func newWorkItemListCmd(g *globals) *cobra.Command {
	return newWorkItemBatchCmd(g, batchCommand{
		use:   "list",
		short: "Print several work items as markdown",
		long: "Get several work items by id with Work Items - List, a GET request that\n" +
			"carries the ids in the URL; at most 200 ids are accepted by the server.\n\n",
		example: "  tfscli workitem list -p MyProject --ids 297,299,300\n" +
			"  tfscli workitem list -p MyProject --ids 297,299,300 --fields System.Title,System.State --error-policy omit",
		read: workitem.List,
	})
}

// batchCommand describes a command that reads several work items: the text
// that tells it apart and the workitem function that makes the request.
// bareNotFound, when set, is the next step added to a 404 that carries no
// message from TFS.
type batchCommand struct {
	use, short, long, example string
	read                      func(context.Context, workitem.APIClient, string, workitem.BatchRequest) (*workitem.Batch, error)
	bareNotFound              string
}

// newWorkItemBatchCmd builds a command reading several work items. Its flags
// are the API parameters, passed through as given: --as-of and --error-policy
// are not checked here, so the server decides what it accepts, and a flag
// that is not given is not sent.
func newWorkItemBatchCmd(g *globals, bc batchCommand) *cobra.Command {
	var (
		project     string
		ids         string
		fields      string
		asOf        string
		errorPolicy string
	)

	cmd := &cobra.Command{
		Use:   bc.use + " --ids <id,...>",
		Short: bc.short,
		Long: bc.long +
			"Work items are printed as `workitem get` prints them, in the order the server\n" +
			"returns them. With --error-policy omit, a work item that does not exist or\n" +
			"cannot be read is listed after the others as \"not returned\" instead of\n" +
			"failing the command.",
		Example: bc.example,
		// Ids given as arguments, the way workitem get takes one, are the
		// likely mistake; the error says where they go instead.
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) > 0 {
				return &tfserr.Error{
					Category: tfserr.Config,
					Message:  fmt.Sprintf("%s takes no arguments; pass the work item ids with --ids, e.g. --ids %s", bc.use, strings.Join(args, ",")),
				}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			parsed, err := parseIDs(ids)
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

			batch, err := bc.read(cmd.Context(), client, cfg.Project, workitem.BatchRequest{
				IDs:         parsed,
				Fields:      splitFields(fields),
				AsOf:        asOf,
				ErrorPolicy: errorPolicy,
			})
			if err != nil {
				return withNotFoundHint(err, bc.bareNotFound)
			}
			return printWorkItems(g.stdout, batch)
		},
	}

	cmd.Flags().StringVarP(&project, "project", "p", "", "team project (TFSCLI_PROJECT; required unless set in the config file)")
	cmd.Flags().StringVar(&ids, "ids", "", "comma-separated work item ids, e.g. 297,299,300 (required)")
	cmd.Flags().StringVar(&fields, "fields", "", "comma-separated field names, e.g. System.Title,System.State (default: all fields)")
	cmd.Flags().StringVar(&asOf, "as-of", "", "read the work items as they were at this UTC time, e.g. 2026-06-14T09:00:00Z")
	cmd.Flags().StringVar(&errorPolicy, "error-policy", "", "fail or omit: whether a work item that cannot be returned fails the request (server default: fail)")
	return cmd
}

// withNotFoundHint adds hint to a 404 whose response carried no TFS message
// of its own. A 404 with a message — a work item that does not exist, say —
// already says what is wrong, and every other error passes through unchanged.
func withNotFoundHint(err error, hint string) error {
	var te *tfserr.Error
	if hint == "" || !errors.As(err, &te) || te.Category != tfserr.NotFound || te.Message != apiclient.NotFoundMessage {
		return err
	}
	return &tfserr.Error{
		Category:   tfserr.NotFound,
		Message:    te.Message + " (" + hint + ")",
		HTTPStatus: te.HTTPStatus,
		Cause:      err,
	}
}

// parseIDs turns the --ids value into work item ids, checking each as parseID
// does. Spaces and empty entries are tolerated as in splitFields.
func parseIDs(value string) ([]int, error) {
	var ids []int
	for arg := range strings.SplitSeq(value, ",") {
		if arg = strings.TrimSpace(arg); arg == "" {
			continue
		}
		id, err := parseID(arg)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, &tfserr.Error{
			Category: tfserr.Config,
			Message:  "no work item ids given (pass them with --ids, e.g. --ids 297,299,300)",
		}
	}
	return ids, nil
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

// splitFields turns the --fields value into the field list workitem.Get and
// workitem.BatchRequest take.
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
