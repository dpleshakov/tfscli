package apiclient

import (
	"context"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dpleshakov/tfscli/internal/config"
	"github.com/dpleshakov/tfscli/internal/log"
	"github.com/dpleshakov/tfscli/internal/tfserr"
)

const testPAT = "secret-pat"

// requestEntry is one LogRequest call captured by recordingLogger.
type requestEntry struct {
	method string
	url    string
	status int
	dur    time.Duration
}

type recordingLogger struct {
	requests []requestEntry
	warnings []string
}

func (l *recordingLogger) LogRequest(method, url string, status int, dur time.Duration) {
	l.requests = append(l.requests, requestEntry{method, url, status, dur})
}

func (l *recordingLogger) Warn(msg string) { l.warnings = append(l.warnings, msg) }

var _ log.Logger = (*recordingLogger)(nil)

// newTestClient builds a Client pointed at serverURL, with the recording
// logger it also returns.
func newTestClient(t *testing.T, serverURL string) (*Client, *recordingLogger) {
	t.Helper()
	logger := &recordingLogger{}
	client, err := New(&config.Config{
		URL:        serverURL,
		Collection: "DefaultCollection",
		PAT:        testPAT,
		APIVersion: "7.2",
	}, logger)
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}
	return client, logger
}

// errorServer replies to every request with status and body.
func errorServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestGetSetsAuthorizationHeader(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Authorization")
	}))
	t.Cleanup(srv.Close)

	client, _ := newTestClient(t, srv.URL)
	if _, err := client.Get(t.Context(), "/MyProject/_apis/wit/workitems/123", nil); err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}

	want := "Basic " + base64.StdEncoding.EncodeToString([]byte(":"+testPAT))
	if got != want {
		t.Errorf("Authorization = %q, want %q", got, want)
	}
}

func TestGetBuildsURL(t *testing.T) {
	var (
		gotPath  string
		gotQuery url.Values
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
	}))
	t.Cleanup(srv.Close)

	client, _ := newTestClient(t, srv.URL)
	query := url.Values{"fields": {"System.Title,System.State"}}
	if _, err := client.Get(t.Context(), "/MyProject/_apis/wit/workitems/123", query); err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}

	if want := "/DefaultCollection/MyProject/_apis/wit/workitems/123"; gotPath != want {
		t.Errorf("path = %q, want %q", gotPath, want)
	}
	if want := "7.2"; gotQuery.Get("api-version") != want {
		t.Errorf("api-version = %q, want %q", gotQuery.Get("api-version"), want)
	}
	if want := "System.Title,System.State"; gotQuery.Get("fields") != want {
		t.Errorf("fields = %q, want %q", gotQuery.Get("fields"), want)
	}
}

func TestGetDoesNotMutateCallerQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	t.Cleanup(srv.Close)

	client, _ := newTestClient(t, srv.URL)
	query := url.Values{"fields": {"System.Title"}}
	if _, err := client.Get(t.Context(), "/MyProject/_apis/wit/workitems/123", query); err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}

	if _, ok := query["api-version"]; ok {
		t.Errorf("caller query gained an api-version key: %v", query)
	}
}

func TestGetReturnsBody(t *testing.T) {
	const body = `{"id":123,"rev":4}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	client, _ := newTestClient(t, srv.URL)
	got, err := client.Get(t.Context(), "/MyProject/_apis/wit/workitems/123", nil)
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	if string(got) != body {
		t.Errorf("body = %q, want %q", got, body)
	}
}

func TestGetClassifiesResponseStatus(t *testing.T) {
	tests := []struct {
		status  int
		want    tfserr.Category
		message string
	}{
		{http.StatusUnauthorized, tfserr.Auth, "PAT is invalid or expired"},
		{http.StatusForbidden, tfserr.Forbidden, "access denied by TFS"},
		{http.StatusNotFound, tfserr.NotFound, "resource not found"},
		{http.StatusInternalServerError, tfserr.Server, "TFS returned a server error"},
		{http.StatusServiceUnavailable, tfserr.Server, "TFS returned a server error"},
		{http.StatusBadRequest, tfserr.Config, "TFS rejected the request"},
		{http.StatusTooManyRequests, tfserr.Config, "TFS rejected the request"},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.status), func(t *testing.T) {
			srv := errorServer(t, tt.status, "")
			client, _ := newTestClient(t, srv.URL)

			_, err := client.Get(t.Context(), "/MyProject/_apis/wit/workitems/123", nil)
			te := assertTFSError(t, err, tt.want)
			if te.HTTPStatus != tt.status {
				t.Errorf("HTTPStatus = %d, want %d", te.HTTPStatus, tt.status)
			}
			if te.Message != tt.message {
				t.Errorf("message = %q, want %q", te.Message, tt.message)
			}
		})
	}
}

func TestGetPrefersServerMessage(t *testing.T) {
	const body = `{"$id":"1","message":"TF401232: Work item 99999 does not exist.","typeKey":"WorkItemNotFoundException"}`
	srv := errorServer(t, http.StatusNotFound, body)
	client, _ := newTestClient(t, srv.URL)

	_, err := client.Get(t.Context(), "/MyProject/_apis/wit/workitems/99999", nil)
	te := assertTFSError(t, err, tfserr.NotFound)
	if want := "TF401232: Work item 99999 does not exist."; te.Message != want {
		t.Errorf("message = %q, want %q", te.Message, want)
	}
}

func TestGetFallsBackToGenericMessage(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"html error page", "<html><body>Proxy authentication required</body></html>"},
		{"json without a message field", `{"$id":"1","typeKey":"SomeException"}`},
		{"empty body", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := errorServer(t, http.StatusForbidden, tt.body)
			client, _ := newTestClient(t, srv.URL)

			_, err := client.Get(t.Context(), "/MyProject/_apis/wit/workitems/123", nil)
			te := assertTFSError(t, err, tfserr.Forbidden)
			if want := "access denied by TFS"; te.Message != want {
				t.Errorf("message = %q, want %q", te.Message, want)
			}
		})
	}
}

func TestGetCollapsesAndBoundsServerMessage(t *testing.T) {
	body := `{"message":"line one\n\tline two   spaced ` + strings.Repeat("x", 400) + `"}`
	srv := errorServer(t, http.StatusInternalServerError, body)
	client, _ := newTestClient(t, srv.URL)

	_, err := client.Get(t.Context(), "/MyProject/_apis/wit/workitems/123", nil)
	te := assertTFSError(t, err, tfserr.Server)

	if strings.ContainsAny(te.Message, "\n\t") {
		t.Errorf("message still contains line breaks or tabs: %q", te.Message)
	}
	if !strings.HasPrefix(te.Message, "line one line two spaced ") {
		t.Errorf("message = %q, want the whitespace collapsed to single spaces", te.Message)
	}
	if got := len([]rune(te.Message)); got > maxMessageRunes+3 {
		t.Errorf("message length = %d runes, want at most %d", got, maxMessageRunes+3)
	}
}

func TestGetUnreachableServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	serverURL := srv.URL
	srv.Close() // nothing listens on that port any more

	client, logger := newTestClient(t, serverURL)
	_, err := client.Get(t.Context(), "/MyProject/_apis/wit/workitems/123", nil)

	te := assertTFSError(t, err, tfserr.Network)
	if want := "cannot reach " + serverURL; te.Message != want {
		t.Errorf("message = %q, want %q", te.Message, want)
	}
	if te.HTTPStatus != 0 {
		t.Errorf("HTTPStatus = %d, want 0", te.HTTPStatus)
	}
	if len(logger.requests) != 1 {
		t.Fatalf("logged %d requests, want 1 — a failed attempt is still one request", len(logger.requests))
	}
	if logger.requests[0].status != 0 {
		t.Errorf("logged status = %d, want 0 for a failed round trip", logger.requests[0].status)
	}
}

func TestGetCanceledContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	client, _ := newTestClient(t, srv.URL)
	_, err := client.Get(ctx, "/MyProject/_apis/wit/workitems/123", nil)

	te := assertTFSError(t, err, tfserr.Network)
	if !strings.Contains(te.Message, "canceled") {
		t.Errorf("message = %q, want it to mention cancellation", te.Message)
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("errors.Is(err, context.Canceled) = false, want true")
	}
}

func TestLogsEveryRequestExactlyOnce(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	client, logger := newTestClient(t, srv.URL)
	for range 2 {
		if _, err := client.Get(t.Context(), "/MyProject/_apis/wit/workitems/123", nil); err != nil {
			t.Fatalf("Get() error = %v, want nil", err)
		}
	}

	if len(logger.requests) != 2 {
		t.Fatalf("logged %d requests, want 2", len(logger.requests))
	}
	entry := logger.requests[0]
	if entry.method != http.MethodGet {
		t.Errorf("method = %q, want %q", entry.method, http.MethodGet)
	}
	if entry.status != http.StatusOK {
		t.Errorf("status = %d, want %d", entry.status, http.StatusOK)
	}
	if !strings.HasPrefix(entry.url, srv.URL+"/DefaultCollection/MyProject/") {
		t.Errorf("url = %q, want it to start with the server and collection", entry.url)
	}
	if strings.Contains(entry.url, testPAT) {
		t.Errorf("url = %q, want it to be free of the PAT", entry.url)
	}
	if entry.dur <= 0 {
		t.Errorf("duration = %v, want a positive duration", entry.dur)
	}
}

func TestNewRejectsBadURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"unparseable", "http://tfs.company.com:pot/tfs"},
		{"missing scheme", "tfs.company.com:8080/tfs"},
		{"unsupported scheme", "ftp://tfs.company.com/tfs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(&config.Config{
				URL:        tt.url,
				Collection: "DefaultCollection",
				PAT:        testPAT,
				APIVersion: "7.2",
			}, log.Noop())
			_ = assertTFSError(t, err, tfserr.Config)
		})
	}
}

func TestNewWarnsWhenVerificationDisabled(t *testing.T) {
	logger := &recordingLogger{}
	if _, err := New(&config.Config{
		URL:                "https://tfs.company.com:8080/tfs",
		Collection:         "DefaultCollection",
		PAT:                testPAT,
		APIVersion:         "7.2",
		InsecureSkipVerify: true,
	}, logger); err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}

	if len(logger.warnings) != 1 || logger.warnings[0] != "TLS verification disabled" {
		t.Errorf("warnings = %v, want exactly one \"TLS verification disabled\"", logger.warnings)
	}
}

func TestNewStaysSilentWhenVerificationIsOn(t *testing.T) {
	logger := &recordingLogger{}
	if _, err := New(&config.Config{
		URL:        "https://tfs.company.com:8080/tfs",
		Collection: "DefaultCollection",
		PAT:        testPAT,
		APIVersion: "7.2",
	}, logger); err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}

	if len(logger.warnings) != 0 {
		t.Errorf("warnings = %v, want none", logger.warnings)
	}
}

func TestNewRejectsBadCABundle(t *testing.T) {
	notPEM := filepath.Join(t.TempDir(), "corp.pem")
	if err := os.WriteFile(notPEM, []byte("this is not a certificate"), 0o600); err != nil {
		t.Fatalf("writing the bundle: %v", err)
	}

	tests := []struct {
		name   string
		bundle string
	}{
		{"missing file", filepath.Join(t.TempDir(), "absent.pem")},
		{"no certificates in file", notPEM},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(&config.Config{
				URL:        "https://tfs.company.com:8080/tfs",
				Collection: "DefaultCollection",
				PAT:        testPAT,
				APIVersion: "7.2",
				CABundle:   tt.bundle,
			}, log.Noop())
			_ = assertTFSError(t, err, tfserr.Config)
		})
	}
}

func TestNewAcceptsCABundle(t *testing.T) {
	// httptest's TLS server certificate is a valid PEM, which is all the
	// bundle loader needs to see.
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	t.Cleanup(srv.Close)

	bundle := filepath.Join(t.TempDir(), "corp.pem")
	encoded := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
	if err := os.WriteFile(bundle, encoded, 0o600); err != nil {
		t.Fatalf("writing the bundle: %v", err)
	}

	if _, err := New(&config.Config{
		URL:        srv.URL,
		Collection: "DefaultCollection",
		PAT:        testPAT,
		APIVersion: "7.2",
		CABundle:   bundle,
	}, log.Noop()); err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}
}

// assertTFSError checks that err is a *tfserr.Error of the wanted category and
// returns it for further assertions.
func assertTFSError(t *testing.T, err error, want tfserr.Category) *tfserr.Error {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want category %q", want)
	}
	var te *tfserr.Error
	if !errors.As(err, &te) {
		t.Fatalf("error = %v, want a *tfserr.Error", err)
	}
	if te.Category != want {
		t.Fatalf("category = %q, want %q", te.Category, want)
	}
	return te
}
