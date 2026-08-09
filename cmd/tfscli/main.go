// Command tfscli is a stateless CLI for read-only access to on-premises
// TFS / Azure DevOps Server.
package main

import (
	"os"

	"github.com/dpleshakov/tfscli/internal/cli"
)

// version and commit are stamped at build time with
// -ldflags "-X main.version=... -X main.commit=...". The defaults are what a
// plain `go build` produces.
var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	os.Exit(cli.Run(version, commit))
}
