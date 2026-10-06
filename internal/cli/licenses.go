package cli

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/dpleshakov/tfscli/internal/licenses"
)

// newLicensesCmd builds "tfscli licenses", a local command like auth login: it
// reads neither the credential nor the config file and calls no server.
func newLicensesCmd(g *globals) *cobra.Command {
	return &cobra.Command{
		Use:   "licenses",
		Short: "Print the licenses of the code included in tfscli",
		Long: "Print the components included in this binary — the Go standard library\n" +
			"and the third-party modules — followed by the license text of each. The\n" +
			"third-party texts are embedded only in the release builds.",
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			_, err := io.WriteString(g.stdout, licenses.Text())
			return err
		},
	}
}
