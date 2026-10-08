package cli

import (
	"net/http"
	"strings"
	"testing"
)

// A flag before the end of the command path would slip past an agent
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
			want: `Error [config]: flags must follow the command "tfscli wit work-items get", and --verbose comes before its end (run it as: tfscli wit work-items get --verbose -p MyProject 12345)` + "\n",
		},
		{
			name: "flag with a separate value between path words",
			args: []string{"wit", "--api-version", "7.1", "work-items", "get", "-p", "MyProject", "12345"},
			want: `Error [config]: flags must follow the command "tfscli wit work-items get", and --api-version comes before its end (run it as: tfscli wit work-items get --api-version 7.1 -p MyProject 12345)` + "\n",
		},
		{
			name: "flag with an inline value and a quoted argument",
			args: []string{"--api-version=7.1", "wit", "wiql", "query-by-wiql", "--query", "SELECT [System.Id] FROM WorkItems"},
			want: `Error [config]: flags must follow the command "tfscli wit wiql query-by-wiql", and --api-version comes before its end (run it as: tfscli wit wiql query-by-wiql --api-version=7.1 --query "SELECT [System.Id] FROM WorkItems")` + "\n",
		},
		{
			name: "flags of the action before its name",
			args: []string{"wit", "work-items", "-p", "MyProject", "--verbose", "get", "12345"},
			want: `Error [config]: flags must follow the command "tfscli wit work-items get", and -p, --verbose come before its end (run it as: tfscli wit work-items get -p MyProject --verbose 12345)` + "\n",
		},
		{
			name: "shorthand with an attached value",
			args: []string{"wit", "work-items", "-pMyProject", "get", "12345"},
			want: `Error [config]: flags must follow the command "tfscli wit work-items get", and -pMyProject comes before its end (run it as: tfscli wit work-items get -pMyProject 12345)` + "\n",
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

// The flags cobra handles itself and its built-in commands keep working
// wherever they are placed, since they send no request.
func TestHelpAndVersionAreNotRefusedBeforeThePath(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{args: []string{"--version"}, want: "tfscli dev (unknown)"},
		{args: []string{"--help"}, want: "Available Commands:"},
		{args: []string{"-h", "wit"}, want: "Available Commands:"},
		{args: []string{"wit", "--help"}, want: "Available Commands:"},
		{args: []string{"wit", "work-items", "--help"}, want: "Available Commands:"},
		{args: []string{"wit", "work-items", "get", "--help"}, want: "--fields"},
		{args: []string{"wit", "--help", "work-items", "get"}, want: "Available Commands:"},
		{args: []string{"help", "wit", "work-items", "get"}, want: "--fields"},
		{args: []string{"--verbose", "help", "wit"}, want: "Available Commands:"},
		{args: []string{"completion", "powershell"}, want: "Register-ArgumentCompleter"},
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

// A flag the target command does not define is reported by cobra as unknown,
// not as misplaced: moving it would not make the command valid.
func TestUnknownFlagBeforeThePathIsLeftToCobra(t *testing.T) {
	_, stderr, code := execute(t, nil, "--nope=1", "wit", "work-items", "get", "-p", "MyProject", "12345")

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if want := "Error [config]: unknown flag: --nope"; !strings.HasPrefix(stderr, want) {
		t.Errorf("stderr = %q, want it to start with %q", stderr, want)
	}
}
