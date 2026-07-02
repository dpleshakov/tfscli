// Command tfscli is a stateless CLI for read-only access to on-premises
// TFS / Azure DevOps Server.
package main

import (
	"os"

	"github.com/dpleshakov/tfscli/internal/cli"
)

func main() {
	os.Exit(cli.Run())
}
