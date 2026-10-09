package cli

import (
	"net/http"
	"strings"
	"testing"
)

// refused is the error for a command line whose command path does not come
// first.
func refused(command, corrected string) string {
	return `Error [config]: the command "` + command + `" must come first, before its flags and arguments (run it as: ` + corrected + ")\n"
}

// A command path that does not open the command line would slip past an agent
// permission rule written as a prefix, so it is refused before any request.
func TestFlagBeforeTheCommandPathIsRefused(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "persistent flag at the root",
			args: []string{"--verbose", "wit", "work-items", "get", "-p", "MyProject", "12345"},
			want: refused("tfscli wit work-items get", "tfscli wit work-items get --verbose -p MyProject 12345"),
		},
		{
			name: "flag with a separate value between path words",
			args: []string{"wit", "--api-version", "7.1", "work-items", "get", "-p", "MyProject", "12345"},
			want: refused("tfscli wit work-items get", "tfscli wit work-items get --api-version 7.1 -p MyProject 12345"),
		},
		{
			name: "flag with an inline value and a quoted argument",
			args: []string{"--api-version=7.1", "wit", "wiql", "query-by-wiql", "--query", "SELECT [System.Id] FROM WorkItems"},
			want: refused("tfscli wit wiql query-by-wiql", "tfscli wit wiql query-by-wiql --api-version=7.1 --query 'SELECT [System.Id] FROM WorkItems'"),
		},
		{
			name: "flags of the action before its name",
			args: []string{"wit", "work-items", "-p", "MyProject", "--verbose", "get", "12345"},
			want: refused("tfscli wit work-items get", "tfscli wit work-items get -p MyProject --verbose 12345"),
		},
		{
			name: "shorthand with an attached value",
			args: []string{"wit", "work-items", "-pMyProject", "get", "12345"},
			want: refused("tfscli wit work-items get", "tfscli wit work-items get -pMyProject 12345"),
		},
		{
			name: "long help flag before the path",
			args: []string{"--help", "wit", "work-items", "get"},
			want: refused("tfscli wit work-items get", "tfscli wit work-items get --help"),
		},
		{
			name: "short help flag before the path",
			args: []string{"-h", "wit", "work-items", "get"},
			want: refused("tfscli wit work-items get", "tfscli wit work-items get -h"),
		},
		{
			name: "help flag between path words",
			args: []string{"wit", "--help", "work-items", "get"},
			want: refused("tfscli wit work-items get", "tfscli wit work-items get --help"),
		},
		{
			name: "help flag followed by a stray word",
			args: []string{"--help", "X", "wit", "work-items", "get"},
			want: refused("tfscli wit work-items get", "tfscli wit work-items get --help X"),
		},
		{
			name: "boolean flag of the action at the level of its group",
			args: []string{"wit", "wiql", "--time-precision", "query-by-wiql", "--query", "x"},
			want: refused("tfscli wit wiql query-by-wiql", "tfscli wit wiql query-by-wiql --time-precision --query x"),
		},
		{
			name: "value that names a subcommand",
			args: []string{"wit", "work-items", "-p", "get", "get", "12345"},
			want: refused("tfscli wit work-items get", "tfscli wit work-items get -p get 12345"),
		},
		{
			name: "value that starts with a dash",
			args: []string{"wit", "work-items", "--fields", "-p", "get", "5", "-p", "MyProject"},
			want: refused("tfscli wit work-items get", "tfscli wit work-items get --fields -p 5 -p MyProject"),
		},
		{
			name: "lone dash before the path",
			args: []string{"-", "wit", "work-items", "get"},
			want: refused("tfscli wit work-items get", "tfscli wit work-items get -"),
		},
		{
			name: "empty word before the path",
			args: []string{"", "wit", "work-items", "get", "5"},
			want: refused("tfscli wit work-items get", "tfscli wit work-items get '' 5"),
		},
		{
			name: "word a shell would interpret",
			args: []string{"--verbose", "wit", "wiql", "query-by-wiql", "--team", "R&D", "--query", "x"},
			want: refused("tfscli wit wiql query-by-wiql", "tfscli wit wiql query-by-wiql --verbose --team 'R&D' --query x"),
		},
		{
			name: "word with a single quote",
			args: []string{"--verbose", "wit", "wiql", "query-by-wiql", "--query", "it's"},
			want: refused("tfscli wit wiql query-by-wiql", `tfscli wit wiql query-by-wiql --verbose --query "it's"`),
		},
		{
			name: "flag before help",
			args: []string{"--verbose", "help", "wit"},
			want: refused("tfscli help", "tfscli help --verbose wit"),
		},
		{
			name: "flag before completion",
			args: []string{"--verbose", "completion", "powershell"},
			want: refused("tfscli completion powershell", "tfscli completion powershell --verbose"),
		},
		{
			name: "version flag before a path",
			args: []string{"--version", "wit"},
			want: `Error [config]: --version belongs to tfscli itself, not to "tfscli wit" (run it as: tfscli --version)` + "\n",
		},
		{
			name: "unknown flag before the path",
			args: []string{"--nope", "wit", "work-items", "get", "-p", "MyProject", "12345"},
			want: "Error [config]: unknown flag: --nope\n",
		},
		{
			name: "unknown flag swallowing a stray word",
			args: []string{"--nope", "X", "wit", "work-items", "get", "-p", "MyProject", "12345"},
			want: "Error [config]: unknown flag: --nope\n",
		},
		{
			name: "unknown shorthand before the path",
			args: []string{"-z", "wit", "work-items", "get", "-p", "MyProject", "12345"},
			want: "Error [config]: unknown shorthand flag: 'z' in -z\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newServer(t, http.StatusOK, workItemResponse)

			stdout, stderr, code := execute(t, s, tt.args...)

			if code != 1 {
				t.Errorf("exit code = %d, want 1", code)
			}
			if stderr != tt.want {
				t.Errorf("stderr = %q, want %q", stderr, tt.want)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want nothing", stdout)
			}
			if s.calls != 0 || s.options != 0 {
				t.Errorf("the server was called %d times and asked for options %d times, want no request at all", s.calls, s.options)
			}
		})
	}
}

// Help, the version, and cobra's own commands work where the rule puts their
// flags: after the command path, or with no path at all.
func TestHelpAndVersionAfterThePath(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{args: []string{"--version"}, want: "tfscli dev (unknown)"},
		{args: []string{"--help"}, want: "Available Commands:"},
		{args: []string{"-h"}, want: "Available Commands:"},
		{args: []string{"wit", "--help"}, want: "Available Commands:"},
		{args: []string{"wit", "work-items", "--help"}, want: "Available Commands:"},
		{args: []string{"wit", "work-items", "get", "--help"}, want: "--fields"},
		{args: []string{"wit", "wiql", "query-by-wiql", "-h"}, want: "--time-precision"},
		{args: []string{"help", "wit", "work-items", "get"}, want: "--fields"},
		{args: []string{"completion", "powershell"}, want: "Register-ArgumentCompleter"},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			stdout, stderr, code := execute(t, nil, tt.args...)

			if code != 0 {
				t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
			}
			if stderr != "" {
				t.Errorf("stderr = %q, want nothing", stderr)
			}
			if !strings.Contains(stdout, tt.want) {
				t.Errorf("stdout does not contain %q:\n%s", tt.want, stdout)
			}
		})
	}
}

// The completion scripts ask for completions with cobra's hidden __complete
// command followed by the words typed so far; such a request runs no command,
// so the words after it need not start with a command path.
func TestShellCompletionRequest(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{args: []string{"__complete", "wit", ""}, want: "work-items\t"},
		{args: []string{"__completeNoDesc", "wit", "wiql", ""}, want: "query-by-wiql\n"},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			stdout, stderr, code := execute(t, nil, tt.args...)

			if code != 0 {
				t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
			}
			if !strings.Contains(stdout, tt.want) {
				t.Errorf("stdout does not contain %q:\n%s", tt.want, stdout)
			}
		})
	}
}
