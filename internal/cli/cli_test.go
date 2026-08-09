package cli

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// workItemResponse is a Get Work Item response trimmed to the fields the
// printer has to distinguish. The full-response case lives in the workitem
// package; here it only has to reach stdout intact.
const workItemResponse = `{
  "id": 12345,
  "rev": 7,
  "fields": {
    "System.WorkItemType": "Bug",
    "System.Title": "Payment confirmation email is not sent for partial refunds",
    "System.State": "Active",
    "System.AssignedTo": {
      "displayName": "Anna Ivanova",
      "uniqueName": "COMPANY\\a.ivanova",
      "imageUrl": "https://tfs.company.com:8080/DefaultCollection/_api/_common/identityImage?id=2f1a8c3e"
    },
    "System.CreatedDate": "2026-06-14T09:12:33.117Z",
    "System.Description": "<div>Customers who receive a <b>partial</b> refund never get the confirmation email.</div>"
  }
}`

// server records the request the CLI made and answers with a canned response.
type server struct {
	*httptest.Server

	status int
	body   string

	calls int
	path  string
	query url.Values
	auth  string
}

func newServer(t *testing.T, status int, body string) *server {
	t.Helper()
	s := &server{status: status, body: body}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.calls++
		s.path = r.URL.Path
		s.query = r.URL.Query()
		s.auth = r.Header.Get("Authorization")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(s.status)
		_, _ = fmt.Fprint(w, s.body)
	}))
	t.Cleanup(s.Close)
	return s
}

// testBuild is the stamp an unstamped `go build` leaves behind.
var testBuild = build{version: "dev", commit: "unknown"}

// execute runs the command tree the way main does and returns what the process
// would have written and exited with.
func execute(t *testing.T, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	isolate(t)

	var out, errOut bytes.Buffer
	code = run(testBuild, args, &out, &errOut)
	return out.String(), errOut.String(), code
}

// isolate cuts the test off from the developer's own environment: a real
// ~/.tfscli/config.json or an exported TFSCLI_* variable must not decide what
// the command under test sees.
func isolate(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	for _, name := range []string{
		"TFSCLI_URL", "TFSCLI_COLLECTION", "TFSCLI_PAT",
		"TFSCLI_PROJECT", "TFSCLI_API_VERSION", "TFSCLI_VERBOSE",
	} {
		t.Setenv(name, "")
	}
}

// getArgs is a complete `workitem get` invocation against s, with every
// required setting passed as a flag.
func getArgs(s *server, extra ...string) []string {
	args := []string{
		"workitem", "get",
		"--url", s.URL,
		"--collection", "DefaultCollection",
		"--pat", "secret-token",
		"-p", "MyProject",
	}
	return append(append(args, extra...), "12345")
}

func TestWorkItemGetPrintsMarkdown(t *testing.T) {
	s := newServer(t, http.StatusOK, workItemResponse)

	stdout, stderr, code := execute(t, getArgs(s)...)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want nothing", stderr)
	}

	want := strings.Join([]string{
		"# Work item 12345 (rev 7)",
		"",
		"System.WorkItemType: Bug",
		"System.Title: Payment confirmation email is not sent for partial refunds",
		"System.State: Active",
		`System.AssignedTo: Anna Ivanova <COMPANY\a.ivanova>`,
		"System.CreatedDate: 2026-06-14T09:12:33Z",
		"",
		"## System.Description",
		"",
		"Customers who receive a **partial** refund never get the confirmation email.",
		"",
	}, "\n")
	if stdout != want {
		t.Errorf("stdout:\n%s\nwant:\n%s", stdout, want)
	}
}

func TestWorkItemGetRequest(t *testing.T) {
	s := newServer(t, http.StatusOK, workItemResponse)

	_, stderr, code := execute(t, getArgs(s, "--fields", "System.Title, System.State")...)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	if want := "/DefaultCollection/MyProject/_apis/wit/workitems/12345"; s.path != want {
		t.Errorf("requested path %q, want %q", s.path, want)
	}
	if got, want := s.query.Get("fields"), "System.Title,System.State"; got != want {
		t.Errorf("fields = %q, want %q — spaces around the comma are trimmed", got, want)
	}
	if got, want := s.query.Get("api-version"), "7.2"; got != want {
		t.Errorf("api-version = %q, want the built-in default %q", got, want)
	}
	if want := "Basic " + base64.StdEncoding.EncodeToString([]byte(":secret-token")); s.auth != want {
		t.Errorf("Authorization = %q, want %q", s.auth, want)
	}
}

func TestWorkItemGetAPIVersionFlag(t *testing.T) {
	s := newServer(t, http.StatusOK, workItemResponse)

	_, stderr, code := execute(t, getArgs(s, "--api-version", "5.0")...)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	if got := s.query.Get("api-version"); got != "5.0" {
		t.Errorf("api-version = %q, want the flag value %q", got, "5.0")
	}
}

func TestWorkItemGetProjectFromEnvironment(t *testing.T) {
	s := newServer(t, http.StatusOK, workItemResponse)
	isolate(t)
	t.Setenv("TFSCLI_PROJECT", "EnvProject")

	var out, errOut bytes.Buffer
	code := run(testBuild, []string{
		"workitem", "get",
		"--url", s.URL, "--collection", "DefaultCollection", "--pat", "secret-token",
		"12345",
	}, &out, &errOut)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, errOut.String())
	}
	if want := "/DefaultCollection/EnvProject/_apis/wit/workitems/12345"; s.path != want {
		t.Errorf("requested path %q, want %q", s.path, want)
	}
}

func TestWorkItemGetReportsServerErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{
			name:   "not found",
			status: http.StatusNotFound,
			body:   `{"message":"TF401232: Work item 12345 does not exist."}`,
			want:   "Error [not_found]: TF401232: Work item 12345 does not exist. (HTTP 404)\n",
		},
		{
			name:   "auth",
			status: http.StatusUnauthorized,
			body:   "",
			want:   "Error [auth]: PAT is invalid or expired (HTTP 401)\n",
		},
		{
			name:   "forbidden",
			status: http.StatusForbidden,
			body:   `{"message":"Access denied."}`,
			want:   "Error [forbidden]: Access denied. (HTTP 403)\n",
		},
		{
			name:   "server",
			status: http.StatusInternalServerError,
			body:   "",
			want:   "Error [server]: TFS returned a server error (HTTP 500)\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newServer(t, tt.status, tt.body)

			stdout, stderr, code := execute(t, getArgs(s)...)

			if code != 1 {
				t.Errorf("exit code = %d, want 1", code)
			}
			if stderr != tt.want {
				t.Errorf("stderr = %q, want %q", stderr, tt.want)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want nothing", stdout)
			}
		})
	}
}

func TestWorkItemGetReportsUnreachableServer(t *testing.T) {
	// A server that is closed straight away leaves a URL nothing listens on.
	s := newServer(t, http.StatusOK, workItemResponse)
	s.Close()

	_, stderr, code := execute(t, getArgs(s)...)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.HasPrefix(stderr, "Error [network]: cannot reach ") {
		t.Errorf("stderr = %q, want a network error", stderr)
	}
}

func TestWorkItemGetRejectsBadInputBeforeCalling(t *testing.T) {
	tests := []struct {
		name  string
		extra []string
		id    string
		want  string
	}{
		{
			name: "id is not a number",
			id:   "twelve",
			want: "Error [config]: work item id \"twelve\" is not a positive integer\n",
		},
		{
			name: "id is zero",
			id:   "0",
			want: "Error [config]: work item id \"0\" is not a positive integer\n",
		},
		{
			name: "id is negative",
			id:   "-5",
			want: "Error [config]: work item id \"-5\" is not a positive integer\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newServer(t, http.StatusOK, workItemResponse)

			args := []string{
				"workitem", "get",
				"--url", s.URL, "--collection", "DefaultCollection", "--pat", "secret-token",
				"-p", "MyProject", "--", tt.id,
			}
			stdout, stderr, code := execute(t, args...)

			if code != 1 {
				t.Errorf("exit code = %d, want 1", code)
			}
			if stderr != tt.want {
				t.Errorf("stderr = %q, want %q", stderr, tt.want)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want nothing", stdout)
			}
			if s.calls != 0 {
				t.Errorf("the server was called %d times, want no call at all", s.calls)
			}
		})
	}
}

func TestWorkItemGetRequiresProject(t *testing.T) {
	s := newServer(t, http.StatusOK, workItemResponse)

	args := []string{
		"workitem", "get",
		"--url", s.URL, "--collection", "DefaultCollection", "--pat", "secret-token",
		"12345",
	}
	_, stderr, code := execute(t, args...)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.HasPrefix(stderr, "Error [config]: project is not set") {
		t.Errorf("stderr = %q, want the missing-project config error", stderr)
	}
	if s.calls != 0 {
		t.Errorf("the server was called %d times, want no call at all", s.calls)
	}
}

func TestMissingConfigurationIsAConfigError(t *testing.T) {
	// No config file in the isolated home, no environment, no flags.
	_, stderr, code := execute(t, "workitem", "get", "-p", "MyProject", "12345")

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.HasPrefix(stderr, "Error [config]: config file not found at ") {
		t.Errorf("stderr = %q, want the missing-config error", stderr)
	}
}

func TestUsageErrorsCarryACategory(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "unknown flag", args: []string{"workitem", "get", "--nope", "12345"}},
		{name: "unknown command", args: []string{"nosuchthing"}},
		{name: "no id", args: []string{"workitem", "get", "-p", "MyProject"}},
		{name: "two ids", args: []string{"workitem", "get", "-p", "MyProject", "1", "2"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, stderr, code := execute(t, tt.args...)

			if code != 1 {
				t.Errorf("exit code = %d, want 1", code)
			}
			if !strings.HasPrefix(stderr, "Error [config]: ") {
				t.Errorf("stderr = %q, want the contract format with a category", stderr)
			}
		})
	}
}

func TestVerboseLogsTheRequestToStderr(t *testing.T) {
	s := newServer(t, http.StatusOK, workItemResponse)

	stdout, stderr, code := execute(t, getArgs(s, "--verbose")...)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	if !strings.HasPrefix(stderr, "GET "+s.URL) {
		t.Errorf("stderr = %q, want a logged GET request", stderr)
	}
	if strings.Contains(stderr, "secret-token") {
		t.Error("the log line contains the PAT")
	}
	if !strings.HasPrefix(stdout, "# Work item 12345") {
		t.Errorf("stdout = %q, want the work item on stdout", stdout)
	}
}

func TestVersionFlagPrintsTheBuildStamp(t *testing.T) {
	stdout, stderr, code := execute(t, "--version")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want nothing", stderr)
	}
	if want := "tfscli dev (unknown)\n"; stdout != want {
		t.Errorf("stdout = %q, want %q", stdout, want)
	}
}

func TestHelpGoesToStdoutWithoutError(t *testing.T) {
	stdout, stderr, code := execute(t, "workitem", "get", "--help")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want nothing", stderr)
	}
	for _, want := range []string{"--fields", "--project", "--api-version", "--verbose"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("help does not mention %s", want)
		}
	}
}
