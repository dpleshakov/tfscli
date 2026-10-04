package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/dpleshakov/tfscli/internal/apiclient"
	"github.com/dpleshakov/tfscli/internal/config"
	"github.com/dpleshakov/tfscli/internal/tfserr"
)

// prompter is the interactive side of stdin: whether a person is at the
// keyboard, and what they type. Tests replace the terminal with a script.
type prompter interface {
	// IsTerminal reports whether stdin is an interactive terminal.
	IsTerminal() bool
	// ReadLine reads one line of visible input, without the line ending.
	ReadLine() (string, error)
	// ReadSecret reads one line without echoing it.
	ReadSecret() (string, error)
}

// stdinPrompter reads from the process stdin.
type stdinPrompter struct {
	lines *bufio.Reader
}

func newStdinPrompter() *stdinPrompter {
	return &stdinPrompter{lines: bufio.NewReader(os.Stdin)}
}

// IsTerminal reports whether stdin is a terminal.
func (p *stdinPrompter) IsTerminal() bool {
	return term.IsTerminal(stdinFd())
}

// ReadLine reads one line from stdin.
func (p *stdinPrompter) ReadLine() (string, error) {
	line, err := p.lines.ReadString('\n')
	if err != nil && (!errors.Is(err, io.EOF) || line == "") {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// ReadSecret reads one line from the terminal with echo turned off.
func (p *stdinPrompter) ReadSecret() (string, error) {
	secret, err := term.ReadPassword(stdinFd())
	return string(secret), err
}

func stdinFd() int {
	return int(os.Stdin.Fd()) //nolint:gosec // a file descriptor always fits in an int
}

func newAuthCmd(g *globals) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Credential commands",
	}
	cmd.AddCommand(newAuthLoginCmd(g))
	return cmd
}

func newAuthLoginCmd(g *globals) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Store the server URL and a personal access token",
		Long: "Ask for the server URL and a personal access token, check them against the\n" +
			"server, and store them together in $XDG_DATA_HOME/tfscli/auth.json (by\n" +
			"default ~/.local/share/tfscli/auth.json), readable by the current user only.\n" +
			"The token is only ever sent to the URL stored with it. Running the command\n" +
			"again replaces the stored credential.\n\n" +
			"The token is typed with echo off and is never accepted as an argument. The\n" +
			"command needs an interactive terminal; where there is none, such as in CI,\n" +
			"set TFSCLI_AUTH to {\"url\": \"…\", \"pat\": \"…\"} instead.\n\n" +
			"The TLS settings (caBundle, insecureSkipVerify) are taken from the config\n" +
			"file when it exists. The API version settings (--api-version,\n" +
			"TFSCLI_API_VERSION, apiVersion) do not apply: the check is sent without an\n" +
			"API version, and the server chooses one.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if cmd.Flags().Changed("api-version") {
				return &tfserr.Error{
					Category: tfserr.Config,
					Message:  "--api-version does not apply to tfscli auth login, which sends its check without an API version (remove the flag)",
				}
			}
			return g.login(cmd.Context())
		},
	}
}

// login runs the interactive login: prompt, verify, store. Nothing is written
// unless the server accepted the credential.
func (g *globals) login(ctx context.Context) error {
	if !g.stdin.IsTerminal() {
		return &tfserr.Error{
			Category: tfserr.Config,
			Message:  "tfscli auth login needs an interactive terminal (set " + config.AuthEnv + " instead)",
		}
	}

	path, err := config.DefaultPath()
	if err != nil {
		return err
	}
	cfg, err := config.LoadFile(path)
	if err != nil {
		return err
	}
	authPath, err := config.DefaultAuthPath()
	if err != nil {
		return err
	}

	auth, err := g.promptAuth()
	if err != nil {
		return err
	}

	cfg.URL = auth.URL
	cfg.PAT = auth.PAT
	user, err := g.verify(ctx, cfg)
	if err != nil {
		return err
	}

	if err := config.SaveAuth(authPath, auth); err != nil {
		return err
	}
	_, err = fmt.Fprintf(g.stdout, "Logged in to %s as %s\n", auth.URL, user)
	return err
}

// promptAuth asks for the URL and the token. Prompts go to stderr so that
// stdout carries only the result.
func (g *globals) promptAuth() (*config.Auth, error) {
	_, _ = fmt.Fprint(g.stderr, "Server URL (e.g. https://tfs.company.com:8080/tfs): ")
	raw, err := g.stdin.ReadLine()
	if err != nil {
		return nil, inputError("the server URL", err)
	}
	url, err := config.NormalizeURL(raw)
	if err != nil {
		return nil, err
	}

	_, _ = fmt.Fprint(g.stderr, "Personal access token: ")
	pat, err := g.stdin.ReadSecret()
	// The terminal swallowed the Enter key along with the echo.
	_, _ = fmt.Fprintln(g.stderr)
	if err != nil {
		return nil, inputError("the personal access token", err)
	}
	pat = strings.TrimSpace(pat)
	if pat == "" {
		return nil, &tfserr.Error{
			Category: tfserr.Config,
			Message:  "the personal access token is empty",
		}
	}
	return &config.Auth{URL: url, PAT: pat}, nil
}

func inputError(what string, err error) error {
	return &tfserr.Error{
		Category: tfserr.Config,
		Message:  "cannot read " + what,
		Cause:    err,
	}
}

// verify checks the credential with a request to _apis/connectionData at the
// server level and returns the display name of the user the token belongs
// to. HTTP failures come back in their usual categories.
func (g *globals) verify(ctx context.Context, cfg *config.Config) (string, error) {
	// The request carries no api-version, whatever the config file says:
	// connectionData has no released version, so a version pinned for the
	// other commands would be refused for it, and without one the server
	// answers at its latest preview version.
	cfg.APIVersion = ""
	client, err := apiclient.New(cfg, g.logger())
	if err != nil {
		return "", err
	}
	body, err := client.Get(ctx, "_apis/connectionData", nil)
	if err != nil {
		return "", err
	}

	var data struct {
		AuthenticatedUser struct {
			ID                  string `json:"id"`
			ProviderDisplayName string `json:"providerDisplayName"`
		} `json:"authenticatedUser"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", &tfserr.Error{
			Category: tfserr.Server,
			Message:  "TFS returned an unexpected response to the login check",
			Cause:    err,
		}
	}
	// A server that lets anonymous requests through answers without a user;
	// the token was then not what got the request in.
	if data.AuthenticatedUser.ID == "" {
		return "", &tfserr.Error{
			Category: tfserr.Auth,
			Message:  "TFS did not identify a user for the PAT",
		}
	}
	if name := data.AuthenticatedUser.ProviderDisplayName; name != "" {
		return name, nil
	}
	return data.AuthenticatedUser.ID, nil
}
