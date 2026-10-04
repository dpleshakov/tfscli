package cli

import "github.com/spf13/cobra"

// newWitCmd is the API area wit, Work Item Tracking. Its resources are named
// after the second segment of their operations' path in the REST API
// reference.
func newWitCmd(g *globals) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wit",
		Short: "Work Item Tracking (REST API area wit)",
	}
	cmd.AddCommand(newWorkItemsCmd(g))
	return cmd
}
