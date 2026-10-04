package cli

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// noTerminal is stdin redirected from a file or a pipe.
type noTerminal struct{}

func (noTerminal) IsTerminal() bool            { return false }
func (noTerminal) ReadLine() (string, error)   { return "", io.EOF }
func (noTerminal) ReadSecret() (string, error) { return "", io.EOF }

// typist is a terminal with a person typing the given answers: lines for the
// visible prompts in order, then secret for the hidden one. When err is set,
// every visible prompt fails with it.
type typist struct {
	lines  []string
	secret string
	err    error
}

// typed is a typist answering the URL, the collection, and the token prompts.
func typed(url, collection, secret string) *typist {
	return &typist{lines: []string{url, collection}, secret: secret}
}

func (*typist) IsTerminal() bool { return true }

func (t *typist) ReadLine() (string, error) {
	if t.err != nil {
		return "", t.err
	}
	if len(t.lines) == 0 {
		return "", io.EOF
	}
	line := t.lines[0]
	t.lines = t.lines[1:]
	return line, nil
}

func (t *typist) ReadSecret() (string, error) { return t.secret, nil }

const connectionDataResponse = `{
  "authenticatedUser": {
    "id": "6f9e4c2a-1b3d-4e5f-8a7b-9c0d1e2f3a4b",
    "providerDisplayName": "Jane Doe"
  },
  "instanceId": "0f8a1c2e-3b4d-5e6f-7a8b-9c0d1e2f3a4b"
}`

// login runs `tfscli auth login` in an isolated environment with in as stdin
// and returns the path where the auth file is expected, along with what the
// process wrote and exited with.
func login(t *testing.T, in prompter) (authPath, stdout, stderr string, code int) {
	t.Helper()
	isolate(t)
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	authPath = filepath.Join(data, "tfscli", "auth.json")

	var out, errOut bytes.Buffer
	code = run(testBuild, []string{"auth", "login"}, in, &out, &errOut)
	return authPath, out.String(), errOut.String(), code
}

func TestAuthLoginStoresTheCredential(t *testing.T) {
	s := newServer(t, http.StatusOK, connectionDataResponse)
	// Mixed case and a trailing slash, as a URL pasted from a browser may be.
	pasted := strings.Replace(s.URL, "http://", "HTTP://", 1) + "/"

	authPath, stdout, stderr, code := login(t, typed(pasted, " DefaultCollection ", "secret-token"))

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	if want := "Logged in to " + s.URL + ", collection DefaultCollection, as Jane Doe\n"; stdout != want {
		t.Errorf("stdout = %q, want %q", stdout, want)
	}
	for _, prompt := range []string{"Server URL", "Collection", "Personal access token"} {
		if !strings.Contains(stderr, prompt) {
			t.Errorf("stderr = %q, want the %q prompt", stderr, prompt)
		}
	}
	if strings.Contains(stderr, "secret-token") {
		t.Error("stderr contains the PAT")
	}

	if want := "/DefaultCollection/_apis/connectionData"; s.path != want {
		t.Errorf("requested path %q, want %q", s.path, want)
	}
	if _, ok := s.query["api-version"]; ok {
		t.Errorf("query = %v, want no api-version", s.query)
	}
	if want := "Basic " + base64.StdEncoding.EncodeToString([]byte(":secret-token")); s.auth != want {
		t.Errorf("Authorization = %q, want %q", s.auth, want)
	}

	data, err := os.ReadFile(authPath)
	if err != nil {
		t.Fatalf("reading the auth file: %v", err)
	}
	var stored map[string]string
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatalf("auth file is not JSON: %v\n%s", err, data)
	}
	want := map[string]string{"url": s.URL, "collection": "DefaultCollection", "pat": "secret-token"}
	if !maps.Equal(stored, want) {
		t.Errorf("auth file = %v, want %v", stored, want)
	}

	if runtime.GOOS == "windows" {
		return
	}
	assertMode(t, authPath, 0o600)
	assertMode(t, filepath.Dir(authPath), 0o700|fs.ModeDir)
}

func TestAuthLoginReplacesTheCredential(t *testing.T) {
	s := newServer(t, http.StatusOK, connectionDataResponse)
	isolate(t)
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	authPath := filepath.Join(data, "tfscli", "auth.json")
	writeTestFile(t, authPath, `{"url": "https://old.example.com", "pat": "old-token"}`)

	var out, errOut bytes.Buffer
	code := run(testBuild, []string{"auth", "login"}, typed(s.URL, "DefaultCollection", "new-token"), &out, &errOut)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, errOut.String())
	}
	body, err := os.ReadFile(authPath)
	if err != nil {
		t.Fatalf("reading the auth file: %v", err)
	}
	if !strings.Contains(string(body), "new-token") || strings.Contains(string(body), "old-token") {
		t.Errorf("auth file = %s, want only the new credential", body)
	}
}

func TestAuthLoginIgnoresTheConfiguredAPIVersion(t *testing.T) {
	s := newServer(t, http.StatusOK, connectionDataResponse)
	isolate(t)
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("TFSCLI_API_VERSION", "7.0")
	writeTestFile(t, filepath.Join(cfgHome, "tfscli", "config.json"), `{"apiVersion": "6.0"}`)

	var out, errOut bytes.Buffer
	code := run(testBuild, []string{"auth", "login"}, typed(s.URL, "DefaultCollection", "secret-token"), &out, &errOut)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, errOut.String())
	}
	if _, ok := s.query["api-version"]; ok {
		t.Errorf("query = %v, want no api-version", s.query)
	}
}

func TestAuthLoginRefusesTheAPIVersionFlag(t *testing.T) {
	s := newServer(t, http.StatusOK, connectionDataResponse)
	isolate(t)
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)

	var out, errOut bytes.Buffer
	code := run(testBuild, []string{"auth", "login", "--api-version", "6.0"},
		typed(s.URL, "DefaultCollection", "secret-token"), &out, &errOut)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	want := "Error [config]: --api-version does not apply to tfscli auth login, which sends its check without an API version (remove the flag)\n"
	if errOut.String() != want {
		t.Errorf("stderr = %q, want %q", errOut.String(), want)
	}
	if s.calls != 0 {
		t.Errorf("the server was called %d times, want 0", s.calls)
	}
	if _, err := os.Stat(filepath.Join(data, "tfscli", "auth.json")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("auth file stat error = %v, want it to be absent", err)
	}
}

func TestAuthLoginFailuresWriteNothing(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		closed bool
		in     func(url string) prompter
		want   string
		calls  int
	}{
		{
			name:   "rejected token",
			status: http.StatusUnauthorized,
			in:     func(url string) prompter { return typed(url, "DefaultCollection", "bad-token") },
			want:   "Error [auth]: PAT is invalid or expired (HTTP 401)\n",
			calls:  1,
		},
		{
			name:   "anonymous answer",
			status: http.StatusOK,
			body:   `{"authenticatedUser": {}}`,
			in:     func(url string) prompter { return typed(url, "DefaultCollection", "bad-token") },
			want:   "Error [auth]: TFS did not identify a user for the PAT\n",
			calls:  1,
		},
		{
			name:   "unexpected answer",
			status: http.StatusOK,
			body:   `<html>proxy login page</html>`,
			in:     func(url string) prompter { return typed(url, "DefaultCollection", "secret-token") },
			want:   "Error [server]: TFS returned an unexpected response to the login check\n",
			calls:  1,
		},
		{
			name:   "unreachable server",
			closed: true,
			in:     func(url string) prompter { return typed(url, "DefaultCollection", "secret-token") },
			want:   "Error [network]: cannot reach ",
		},
		{
			name: "stdin is not a terminal",
			in:   func(string) prompter { return noTerminal{} },
			want: "Error [config]: tfscli auth login needs an interactive terminal (set TFSCLI_AUTH instead)\n",
		},
		{
			name: "url without a scheme",
			in:   func(string) prompter { return typed("tfs.company.com/tfs", "DefaultCollection", "secret-token") },
			want: "Error [config]: url \"tfs.company.com/tfs\" must start with http:// or https://\n",
		},
		{
			name: "empty collection",
			in:   func(url string) prompter { return typed(url, "  ", "secret-token") },
			want: "Error [config]: the collection is empty\n",
		},
		{
			name: "input closed at the collection",
			in:   func(url string) prompter { return &typist{lines: []string{url}} },
			want: "Error [config]: cannot read the collection\n",
		},
		{
			name: "empty token",
			in:   func(url string) prompter { return typed(url, "DefaultCollection", "  ") },
			want: "Error [config]: the personal access token is empty\n",
		},
		{
			name: "input closed",
			in:   func(string) prompter { return &typist{err: io.EOF} },
			want: "Error [config]: cannot read the server URL\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newServer(t, tt.status, tt.body)
			if tt.closed {
				s.Close()
			}

			authPath, stdout, stderr, code := login(t, tt.in(s.URL))

			if code != 1 {
				t.Errorf("exit code = %d, want 1", code)
			}
			if !strings.Contains(stderr, tt.want) {
				t.Errorf("stderr = %q, want it to contain %q", stderr, tt.want)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want nothing", stdout)
			}
			if s.calls != tt.calls {
				t.Errorf("the server was called %d times, want %d", s.calls, tt.calls)
			}
			if _, err := os.Stat(authPath); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("auth file stat error = %v, want it to be absent", err)
			}
		})
	}
}

func TestAuthLoginRefusesWithoutTerminalBeforePrompting(t *testing.T) {
	_, _, stderr, _ := login(t, noTerminal{})

	if strings.Contains(stderr, "Server URL") {
		t.Errorf("stderr = %q, want no prompt", stderr)
	}
}

func TestAuthLoginTakesNoArguments(t *testing.T) {
	_, stderr, code := execute(t, nil, "auth", "login", "secret-token")

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.HasPrefix(stderr, "Error [config]: ") {
		t.Errorf("stderr = %q, want a config error", stderr)
	}
}

func assertMode(t *testing.T, path string, want fs.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := info.Mode() & (fs.ModePerm | fs.ModeDir); got != want {
		t.Errorf("mode of %s = %v, want %v", path, got, want)
	}
}
