package cli

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
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
      "displayName": "Jane Doe",
      "uniqueName": "COMPANY\\j.doe",
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

	calls  int
	method string
	path   string
	query  url.Values
	auth   string
	sent   string
}

func newServer(t *testing.T, status int, body string) *server {
	t.Helper()
	s := &server{status: status, body: body}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.calls++
		s.method = r.Method
		s.path = r.URL.Path
		s.query = r.URL.Query()
		s.auth = r.Header.Get("Authorization")
		sent, _ := io.ReadAll(r.Body)
		s.sent = string(sent)

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
// would have written and exited with. It runs in an isolated environment;
// when s is not nil, TFSCLI_AUTH holds a credential for it.
func execute(t *testing.T, s *server, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	isolate(t)
	if s != nil {
		t.Setenv("TFSCLI_AUTH", authFor(s))
	}

	var out, errOut bytes.Buffer
	code = run(testBuild, args, noTerminal{}, &out, &errOut)
	return out.String(), errOut.String(), code
}

// authFor is the credential JSON for s.
func authFor(s *server) string {
	return `{"url": "` + s.URL + `", "collection": "DefaultCollection", "pat": "secret-token"}`
}

// isolate cuts the test off from the developer's own environment: a real
// config or auth file, an exported XDG_* directory, or an exported TFSCLI_*
// variable must not decide what the command under test sees.
func isolate(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_DATA_HOME", "")
	for _, name := range []string{
		"TFSCLI_AUTH",
		"TFSCLI_PROJECT", "TFSCLI_API_VERSION", "TFSCLI_VERBOSE",
	} {
		t.Setenv(name, "")
	}
}

// getArgs is a complete `wit work-items get` invocation, with every required
// setting other than the credential passed as a flag.
func getArgs(extra ...string) []string {
	args := []string{
		"wit", "work-items", "get",
		"-p", "MyProject",
	}
	return append(append(args, extra...), "12345")
}

func TestWorkItemGetPrintsMarkdown(t *testing.T) {
	s := newServer(t, http.StatusOK, workItemResponse)

	stdout, stderr, code := execute(t, s, getArgs()...)

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
		`System.AssignedTo: Jane Doe <COMPANY\j.doe>`,
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

	_, stderr, code := execute(t, s, getArgs("--fields", "System.Title, System.State")...)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	if want := "/DefaultCollection/MyProject/_apis/wit/workitems/12345"; s.path != want {
		t.Errorf("requested path %q, want %q", s.path, want)
	}
	if got, want := s.query.Get("fields"), "System.Title,System.State"; got != want {
		t.Errorf("fields = %q, want %q — spaces around the comma are trimmed", got, want)
	}
	if _, ok := s.query["api-version"]; ok {
		t.Errorf("query = %v, want no api-version when none is configured", s.query)
	}
	if want := "Basic " + base64.StdEncoding.EncodeToString([]byte(":secret-token")); s.auth != want {
		t.Errorf("Authorization = %q, want %q", s.auth, want)
	}
}

func TestWorkItemGetConfiguredAPIVersion(t *testing.T) {
	tests := []struct {
		name string
		flag string
		env  string
		file string
		want string
	}{
		{name: "flag", flag: "5.0", want: "5.0"},
		{name: "environment", env: "6.0-preview.3", want: "6.0-preview.3"},
		{name: "config file", file: `{"apiVersion": "4.1"}`, want: "4.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newServer(t, http.StatusOK, workItemResponse)
			isolate(t)
			t.Setenv("TFSCLI_AUTH", authFor(s))
			t.Setenv("TFSCLI_API_VERSION", tt.env)
			if tt.file != "" {
				cfgHome := t.TempDir()
				t.Setenv("XDG_CONFIG_HOME", cfgHome)
				writeTestFile(t, filepath.Join(cfgHome, "tfscli", "config.json"), tt.file)
			}
			args := getArgs()
			if tt.flag != "" {
				args = getArgs("--api-version", tt.flag)
			}

			var out, errOut bytes.Buffer
			code := run(testBuild, args, noTerminal{}, &out, &errOut)

			if code != 0 {
				t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, errOut.String())
			}
			if got := s.query.Get("api-version"); got != tt.want {
				t.Errorf("api-version = %q, want %q unchanged", got, tt.want)
			}
		})
	}
}

func TestWorkItemGetRefusedAPIVersionNamesItsSource(t *testing.T) {
	const serverMessage = "The requested REST API version of 7.2 is out of range for this server. " +
		"The latest REST API version this server supports is 7.1."
	const body = `{"$id":"1","message":"` + serverMessage + `","typeKey":"VssVersionOutOfRangeException"}`
	const step = "; remove it to let the server choose the version, or set one the server supports) (HTTP 400)\n"

	tests := []struct {
		name   string
		flag   bool
		env    bool
		file   bool
		source func(cfgPath string) string
	}{
		{name: "flag", flag: true, source: func(string) string { return "--api-version" }},
		{name: "environment", env: true, source: func(string) string { return "TFSCLI_API_VERSION" }},
		{name: "config file", file: true, source: func(p string) string { return `"apiVersion" in ` + p }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newServer(t, http.StatusBadRequest, body)
			isolate(t)
			t.Setenv("TFSCLI_AUTH", authFor(s))
			cfgHome := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", cfgHome)
			cfgPath := filepath.Join(cfgHome, "tfscli", "config.json")
			if tt.file {
				writeTestFile(t, cfgPath, `{"apiVersion": "7.2"}`)
			}
			if tt.env {
				t.Setenv("TFSCLI_API_VERSION", "7.2")
			}
			args := getArgs()
			if tt.flag {
				args = getArgs("--api-version", "7.2")
			}

			var out, errOut bytes.Buffer
			code := run(testBuild, args, noTerminal{}, &out, &errOut)

			if code != 1 {
				t.Errorf("exit code = %d, want 1", code)
			}
			want := "Error [config]: " + serverMessage + ` (api-version "7.2" is set by ` + tt.source(cfgPath) + step
			if errOut.String() != want {
				t.Errorf("stderr = %q, want %q", errOut.String(), want)
			}
		})
	}
}

func TestWorkItemGetProjectFromEnvironment(t *testing.T) {
	s := newServer(t, http.StatusOK, workItemResponse)
	isolate(t)
	t.Setenv("TFSCLI_AUTH", authFor(s))
	t.Setenv("TFSCLI_PROJECT", "EnvProject")

	var out, errOut bytes.Buffer
	code := run(testBuild, []string{
		"wit", "work-items", "get", "12345",
	}, noTerminal{}, &out, &errOut)

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
			want:   "Error [auth]: the server did not accept the PAT (it may be invalid, expired, or revoked; if it is valid, IIS Basic Authentication may be enabled on the server, which only an administrator can turn off) (HTTP 401)\n",
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

			stdout, stderr, code := execute(t, s, getArgs()...)

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

	_, stderr, code := execute(t, s, getArgs()...)

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
				"wit", "work-items", "get",
				"-p", "MyProject", "--", tt.id,
			}
			stdout, stderr, code := execute(t, s, args...)

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

	args := []string{"wit", "work-items", "get", "12345"}
	_, stderr, code := execute(t, s, args...)

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

func TestMissingCredentialIsAConfigError(t *testing.T) {
	// No auth file in the isolated home and no TFSCLI_AUTH.
	_, stderr, code := execute(t, nil, getArgs()...)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.HasPrefix(stderr, "Error [config]: not logged in: ") ||
		!strings.Contains(stderr, "tfscli auth login") {
		t.Errorf("stderr = %q, want the not-logged-in error", stderr)
	}
}

func TestWorkItemGetReadsTheAuthFile(t *testing.T) {
	s := newServer(t, http.StatusOK, workItemResponse)
	isolate(t)
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	writeTestFile(t, filepath.Join(data, "tfscli", "auth.json"), authFor(s))

	var out, errOut bytes.Buffer
	code := run(testBuild, getArgs(), noTerminal{}, &out, &errOut)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, errOut.String())
	}
	if want := "Basic " + base64.StdEncoding.EncodeToString([]byte(":secret-token")); s.auth != want {
		t.Errorf("Authorization = %q, want %q", s.auth, want)
	}
}

func TestCredentialFlagsAreGone(t *testing.T) {
	for _, flag := range []string{"--url", "--collection", "--pat"} {
		t.Run(flag, func(t *testing.T) {
			s := newServer(t, http.StatusOK, workItemResponse)

			_, stderr, code := execute(t, s, getArgs(flag, "value")...)

			if code != 1 {
				t.Errorf("exit code = %d, want 1", code)
			}
			if want := "Error [config]: unknown flag: " + flag + "\n"; stderr != want {
				t.Errorf("stderr = %q, want %q", stderr, want)
			}
			if s.calls != 0 {
				t.Errorf("the server was called %d times, want no call at all", s.calls)
			}
		})
	}
}

// writeTestFile writes body to path, creating the parent directories.
func writeTestFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("creating %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

func TestUsageErrorsCarryACategory(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "unknown flag", args: []string{"wit", "work-items", "get", "--nope", "12345"}},
		{name: "unknown command", args: []string{"nosuchthing"}},
		{name: "no id", args: []string{"wit", "work-items", "get", "-p", "MyProject"}},
		{name: "two ids", args: []string{"wit", "work-items", "get", "-p", "MyProject", "1", "2"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, stderr, code := execute(t, nil, tt.args...)

			if code != 1 {
				t.Errorf("exit code = %d, want 1", code)
			}
			if !strings.HasPrefix(stderr, "Error [config]: ") {
				t.Errorf("stderr = %q, want the contract format with a category", stderr)
			}
		})
	}
}

// The commands were named workitem get, list, and get-batch before they were
// placed in the area wit; the old names are gone without aliases.
func TestOldWorkItemCommandIsUnknown(t *testing.T) {
	s := newServer(t, http.StatusOK, workItemResponse)

	stdout, stderr, code := execute(t, s, "workitem", "get", "-p", "MyProject", "12345")

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if want := `Error [config]: unknown command "workitem" for "tfscli"`; !strings.HasPrefix(stderr, want) {
		t.Errorf("stderr = %q, want it to start with %q", stderr, want)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want nothing", stdout)
	}
	if s.calls != 0 {
		t.Errorf("the server was called %d times, want no call at all", s.calls)
	}
}

func TestVerboseLogsTheRequestToStderr(t *testing.T) {
	s := newServer(t, http.StatusOK, workItemResponse)

	stdout, stderr, code := execute(t, s, getArgs("--verbose")...)

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
	stdout, stderr, code := execute(t, nil, "--version")

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
	stdout, stderr, code := execute(t, nil, "wit", "work-items", "get", "--help")

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

// listResponse is a Work Items - List response with errorPolicy=omit: 299 was
// not returned, and the server answers in an order of its own.
const listResponse = `{
  "count": 3,
  "value": [
    {"id": 297, "rev": 1, "fields": {"System.Title": "Customer can sign in"}},
    null,
    {"id": 12345, "rev": 7, "fields": {"System.Title": "Payment confirmation email is not sent for partial refunds"}}
  ]
}`

// batchCommands are the commands that read several work items. They share
// their flags, checks, and output, so those cases are run against both.
var batchCommands = []string{"list", "get-batch"}

// batchArgs is a complete invocation of the command named, with every
// required setting other than the credential passed as a flag.
func batchArgs(command string, extra ...string) []string {
	return append([]string{"wit", "work-items", command, "-p", "MyProject", "--ids", "12345,299,297"}, extra...)
}

func TestWorkItemBatchPrintsMarkdown(t *testing.T) {
	for _, command := range batchCommands {
		t.Run(command, func(t *testing.T) {
			s := newServer(t, http.StatusOK, listResponse)

			stdout, stderr, code := execute(t, s, batchArgs(command, "--error-policy", "omit")...)

			if code != 0 {
				t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
			}
			if stderr != "" {
				t.Errorf("stderr = %q, want nothing", stderr)
			}

			want := strings.Join([]string{
				"# Work item 297 (rev 1)",
				"",
				"System.Title: Customer can sign in",
				"",
				"# Work item 12345 (rev 7)",
				"",
				"System.Title: Payment confirmation email is not sent for partial refunds",
				"",
				"# Work item 299 (not returned: it does not exist, or the PAT has no access to it)",
				"",
			}, "\n")
			if stdout != want {
				t.Errorf("stdout:\n%s\nwant:\n%s", stdout, want)
			}
		})
	}
}

func TestWorkItemListRequest(t *testing.T) {
	tests := []struct {
		name  string
		extra []string
		want  url.Values
	}{
		{
			name: "ids only",
			want: url.Values{"ids": {"12345,299,297"}},
		},
		{
			name:  "every flag",
			extra: []string{"--fields", "System.Title, System.State", "--as-of", "2026-06-14T09:00:00Z", "--error-policy", "omit"},
			want: url.Values{
				"ids":         {"12345,299,297"},
				"fields":      {"System.Title,System.State"},
				"asOf":        {"2026-06-14T09:00:00Z"},
				"errorPolicy": {"omit"},
			},
		},
		{
			// Values the server may refuse are still its to refuse.
			name:  "values passed through unchecked",
			extra: []string{"--as-of", "yesterday", "--error-policy", "ignore"},
			want: url.Values{
				"ids":         {"12345,299,297"},
				"asOf":        {"yesterday"},
				"errorPolicy": {"ignore"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newServer(t, http.StatusOK, listResponse)

			_, stderr, code := execute(t, s, batchArgs("list", tt.extra...)...)

			if code != 0 {
				t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
			}
			if s.method != http.MethodGet {
				t.Errorf("method = %s, want GET", s.method)
			}
			if want := "/DefaultCollection/MyProject/_apis/wit/workitems"; s.path != want {
				t.Errorf("requested path %q, want %q", s.path, want)
			}
			if got, want := s.query.Encode(), tt.want.Encode(); got != want {
				t.Errorf("query = %q, want %q", got, want)
			}
		})
	}
}

func TestWorkItemBatchReportsServerErrors(t *testing.T) {
	for _, command := range batchCommands {
		t.Run(command, func(t *testing.T) {
			body := `{"message":"TF401232: Work item 299 does not exist, or you do not have permissions to read it.","typeKey":"WorkItemUnauthorizedAccessException"}`
			s := newServer(t, http.StatusNotFound, body)

			stdout, stderr, code := execute(t, s, batchArgs(command)...)

			if code != 1 {
				t.Errorf("exit code = %d, want 1", code)
			}
			want := "Error [not_found]: TF401232: Work item 299 does not exist, or you do not have permissions to read it. (HTTP 404)\n"
			if stderr != want {
				t.Errorf("stderr = %q, want %q", stderr, want)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want nothing", stdout)
			}
		})
	}
}

func TestWorkItemBatchRejectsBadInputBeforeCalling(t *testing.T) {
	for _, command := range batchCommands {
		t.Run(command, func(t *testing.T) {
			tests := []struct {
				name string
				args []string
				want string
			}{
				{
					name: "no --ids",
					args: []string{"wit", "work-items", command, "-p", "MyProject"},
					want: "Error [config]: no work item ids given (pass them with --ids, e.g. --ids 297,299,300)\n",
				},
				{
					name: "only commas",
					args: []string{"wit", "work-items", command, "-p", "MyProject", "--ids", " , ,"},
					want: "Error [config]: no work item ids given (pass them with --ids, e.g. --ids 297,299,300)\n",
				},
				{
					name: "an id is not a number",
					args: []string{"wit", "work-items", command, "-p", "MyProject", "--ids", "297,twelve"},
					want: "Error [config]: work item id \"twelve\" is not a positive integer\n",
				},
				{
					name: "an id is zero",
					args: []string{"wit", "work-items", command, "-p", "MyProject", "--ids", "0,297"},
					want: "Error [config]: work item id \"0\" is not a positive integer\n",
				},
				{
					name: "ids as arguments",
					args: []string{"wit", "work-items", command, "-p", "MyProject", "297", "299"},
					want: "Error [config]: " + command + " takes no arguments; pass the work item ids with --ids, e.g. --ids 297,299\n",
				},
				{
					name: "no project",
					args: []string{"wit", "work-items", command, "--ids", "297"},
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					s := newServer(t, http.StatusOK, listResponse)

					stdout, stderr, code := execute(t, s, tt.args...)

					if code != 1 {
						t.Errorf("exit code = %d, want 1", code)
					}
					if tt.want == "" {
						if !strings.HasPrefix(stderr, "Error [config]: project is not set") {
							t.Errorf("stderr = %q, want the missing-project config error", stderr)
						}
					} else if stderr != tt.want {
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
		})
	}
}

func TestWorkItemGetBatchRequest(t *testing.T) {
	tests := []struct {
		name  string
		extra []string
		want  string
	}{
		{
			name: "ids only",
			want: `{"ids":[12345,299,297]}`,
		},
		{
			name:  "every flag",
			extra: []string{"--fields", "System.Title, System.State", "--as-of", "2026-06-14T09:00:00Z", "--error-policy", "omit"},
			want:  `{"ids":[12345,299,297],"fields":["System.Title","System.State"],"asOf":"2026-06-14T09:00:00Z","errorPolicy":"omit"}`,
		},
		{
			// Values the server may refuse are still its to refuse.
			name:  "values passed through unchecked",
			extra: []string{"--as-of", "yesterday", "--error-policy", "ignore"},
			want:  `{"ids":[12345,299,297],"asOf":"yesterday","errorPolicy":"ignore"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newServer(t, http.StatusOK, listResponse)

			_, stderr, code := execute(t, s, batchArgs("get-batch", tt.extra...)...)

			if code != 0 {
				t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
			}
			if s.method != http.MethodPost {
				t.Errorf("method = %s, want POST", s.method)
			}
			if want := "/DefaultCollection/MyProject/_apis/wit/workitemsbatch"; s.path != want {
				t.Errorf("requested path %q, want %q", s.path, want)
			}
			if len(s.query) != 0 {
				t.Errorf("query = %v, want none: the parameters belong in the body", s.query)
			}
			if s.sent != tt.want {
				t.Errorf("body = %s, want %s", s.sent, tt.want)
			}
		})
	}
}

func TestWorkItemGetBatchNamesTheServerItNeedsOnABare404(t *testing.T) {
	for _, body := range []string{"", "<html><body>404 - File or directory not found.</body></html>"} {
		s := newServer(t, http.StatusNotFound, body)

		_, stderr, code := execute(t, s, batchArgs("get-batch")...)

		if code != 1 {
			t.Errorf("exit code = %d, want 1", code)
		}
		want := "Error [not_found]: resource not found (Get Work Items Batch needs Azure DevOps Server 2019 or later; " +
			"on an older server use tfscli wit work-items list) (HTTP 404)\n"
		if stderr != want {
			t.Errorf("stderr = %q, want %q", stderr, want)
		}
	}
}

func TestWorkItemListLeavesABare404Alone(t *testing.T) {
	s := newServer(t, http.StatusNotFound, "")

	_, stderr, _ := execute(t, s, batchArgs("list")...)

	if want := "Error [not_found]: resource not found (HTTP 404)\n"; stderr != want {
		t.Errorf("stderr = %q, want %q", stderr, want)
	}
}
