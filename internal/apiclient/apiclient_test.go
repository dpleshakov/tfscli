package apiclient

import (
	"context"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
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
	infos    []string
}

func (l *recordingLogger) LogRequest(method, url string, status int, dur time.Duration) {
	l.requests = append(l.requests, requestEntry{method, url, status, dur})
}

func (l *recordingLogger) Warn(msg string) { l.warnings = append(l.warnings, msg) }

func (l *recordingLogger) Info(msg string) { l.infos = append(l.infos, msg) }

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
	if _, err := client.Get(t.Context(), testLocation, "/MyProject/_apis/wit/workitems/123", nil); err != nil {
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
	if _, err := client.Get(t.Context(), testLocation, "/MyProject/_apis/wit/workitems/123", query); err != nil {
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

func TestGetEscapesCollectionAndKeepsEscapedPath(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
	}))
	t.Cleanup(srv.Close)

	client, err := New(&config.Config{
		URL:        srv.URL,
		Collection: "100% Collection",
		PAT:        testPAT,
	}, log.Noop())
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}
	if _, err := client.Get(t.Context(), testLocation, "100%25%20Done/_apis/wit/workitems/123", nil); err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}

	if want := "/100% Collection/100% Done/_apis/wit/workitems/123"; gotPath != want {
		t.Errorf("path = %q, want %q", gotPath, want)
	}
}

func TestGetWithoutLocationSendsNoVersion(t *testing.T) {
	var methods []string
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		gotQuery = r.URL.Query()
	}))
	t.Cleanup(srv.Close)

	client, err := New(&config.Config{
		URL:        srv.URL,
		Collection: "DefaultCollection",
		PAT:        testPAT,
	}, &recordingLogger{})
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}
	if _, err := client.Get(t.Context(), "", "_apis/connectionData", nil); err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}

	if want := []string{http.MethodGet}; !slices.Equal(methods, want) {
		t.Errorf("requests = %v, want %v — no OPTIONS without a location", methods, want)
	}
	if _, ok := gotQuery["api-version"]; ok {
		t.Errorf("query = %v, want no api-version", gotQuery)
	}
}

func TestGetDoesNotMutateCallerQuery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	t.Cleanup(srv.Close)

	client, _ := newTestClient(t, srv.URL)
	query := url.Values{"fields": {"System.Title"}}
	if _, err := client.Get(t.Context(), testLocation, "/MyProject/_apis/wit/workitems/123", query); err != nil {
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
	got, err := client.Get(t.Context(), testLocation, "/MyProject/_apis/wit/workitems/123", nil)
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
		{http.StatusUnauthorized, tfserr.Auth, "the server did not accept the PAT (it may be invalid, expired, or revoked; if it is valid, IIS Basic Authentication may be enabled on the server, which only an administrator can turn off)"},
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

			_, err := client.Get(t.Context(), testLocation, "/MyProject/_apis/wit/workitems/123", nil)
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

	_, err := client.Get(t.Context(), testLocation, "/MyProject/_apis/wit/workitems/99999", nil)
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

			_, err := client.Get(t.Context(), testLocation, "/MyProject/_apis/wit/workitems/123", nil)
			te := assertTFSError(t, err, tfserr.Forbidden)
			if want := "access denied by TFS"; te.Message != want {
				t.Errorf("message = %q, want %q", te.Message, want)
			}
		})
	}
}

const outOfRangeMessage = "The requested REST API version of 7.2 is out of range for this server. " +
	"The latest REST API version this server supports is 7.1."

func TestGetNamesTheSourceOfARefusedVersion(t *testing.T) {
	body := `{"$id":"1","message":"` + outOfRangeMessage + `","typeKey":"VssVersionOutOfRangeException"}`
	srv := errorServer(t, http.StatusBadRequest, body)
	client, err := New(&config.Config{
		URL:              srv.URL,
		Collection:       "DefaultCollection",
		PAT:              testPAT,
		APIVersion:       "7.2",
		APIVersionSource: "TFSCLI_API_VERSION",
	}, log.Noop())
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}

	_, err = client.Get(t.Context(), testLocation, "/MyProject/_apis/wit/workitems/123", nil)
	te := assertTFSError(t, err, tfserr.Config)
	want := outOfRangeMessage + ` (api-version "7.2" is set by TFSCLI_API_VERSION; ` +
		"remove it to let the server choose the version, or set one the server supports)"
	if te.Message != want {
		t.Errorf("message = %q, want %q", te.Message, want)
	}
}

func TestGetLeavesOtherRejectionsAlone(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		apiVersion string
		want       string
	}{
		{
			name:       "another typeKey",
			body:       `{"message":"TF51535: Cannot find field System.Titel.","typeKey":"WorkItemFieldInvalidException"}`,
			apiVersion: "7.2",
			want:       "TF51535: Cannot find field System.Titel.",
		},
		{
			name:       "no typeKey",
			body:       `{"message":"` + outOfRangeMessage + `"}`,
			apiVersion: "7.2",
			want:       outOfRangeMessage,
		},
		{
			name: "version error with no version configured",
			body: `{"message":"` + outOfRangeMessage + `","typeKey":"VssVersionOutOfRangeException"}`,
			want: outOfRangeMessage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := errorServer(t, http.StatusBadRequest, tt.body)
			source := ""
			if tt.apiVersion != "" {
				source = "--api-version"
			}
			client, err := New(&config.Config{
				URL:              srv.URL,
				Collection:       "DefaultCollection",
				PAT:              testPAT,
				APIVersion:       tt.apiVersion,
				APIVersionSource: source,
			}, log.Noop())
			if err != nil {
				t.Fatalf("New() error = %v, want nil", err)
			}

			_, err = client.Get(t.Context(), testLocation, "/MyProject/_apis/wit/workitems/123", nil)
			te := assertTFSError(t, err, tfserr.Config)
			if te.Message != tt.want {
				t.Errorf("message = %q, want %q", te.Message, tt.want)
			}
		})
	}
}

func TestGetCollapsesAndBoundsServerMessage(t *testing.T) {
	body := `{"message":"line one\n\tline two   spaced ` + strings.Repeat("x", 400) + `"}`
	srv := errorServer(t, http.StatusInternalServerError, body)
	client, _ := newTestClient(t, srv.URL)

	_, err := client.Get(t.Context(), testLocation, "/MyProject/_apis/wit/workitems/123", nil)
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
	_, err := client.Get(t.Context(), testLocation, "/MyProject/_apis/wit/workitems/123", nil)

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
	_, err := client.Get(ctx, testLocation, "/MyProject/_apis/wit/workitems/123", nil)

	te := assertTFSError(t, err, tfserr.Network)
	if !strings.Contains(te.Message, "canceled") {
		t.Errorf("message = %q, want it to mention cancellation", te.Message)
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("errors.Is(err, context.Canceled) = false, want true")
	}
}

func TestLogsEveryRequestExactlyOnce(t *testing.T) {
	// The monotonic clock on Windows advances in steps of up to 15.6ms, so a
	// local round trip can measure 0s; a handler that waits longer than one
	// step makes the duration measurable on every platform.
	const delay = 20 * time.Millisecond
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(delay)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	client, logger := newTestClient(t, srv.URL)
	for range 2 {
		if _, err := client.Get(t.Context(), testLocation, "/MyProject/_apis/wit/workitems/123", nil); err != nil {
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
	if entry.dur < delay {
		t.Errorf("duration = %v, want at least %v", entry.dur, delay)
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

// postRecord is what a test server saw of one request.
type postRecord struct {
	method      string
	path        string
	query       url.Values
	contentType string
	auth        string
	body        string
}

func recordingServer(t *testing.T, reply string) (*httptest.Server, *postRecord) {
	t.Helper()
	rec := &postRecord{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		*rec = postRecord{
			method:      r.Method,
			path:        r.URL.Path,
			query:       r.URL.Query(),
			contentType: r.Header.Get("Content-Type"),
			auth:        r.Header.Get("Authorization"),
			body:        string(body),
		}
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(srv.Close)
	return srv, rec
}

func TestPostSendsJSONBody(t *testing.T) {
	srv, rec := recordingServer(t, `{"count":0,"value":[]}`)
	client, _ := newTestClient(t, srv.URL)

	body := struct {
		IDs    []int    `json:"ids"`
		Fields []string `json:"fields,omitempty"`
	}{IDs: []int{297, 299}}
	got, err := client.Post(t.Context(), testLocation, "/MyProject/_apis/wit/workitemsbatch", nil, body)
	if err != nil {
		t.Fatalf("Post() error = %v, want nil", err)
	}

	if rec.method != http.MethodPost {
		t.Errorf("method = %q, want POST", rec.method)
	}
	if want := "/DefaultCollection/MyProject/_apis/wit/workitemsbatch"; rec.path != want {
		t.Errorf("path = %q, want %q", rec.path, want)
	}
	if rec.contentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", rec.contentType)
	}
	if want := `{"ids":[297,299]}`; rec.body != want {
		t.Errorf("body = %q, want %q", rec.body, want)
	}
	if want := "Basic " + base64.StdEncoding.EncodeToString([]byte(":"+testPAT)); rec.auth != want {
		t.Errorf("Authorization = %q, want %q", rec.auth, want)
	}
	if want := `{"count":0,"value":[]}`; string(got) != want {
		t.Errorf("returned body = %q, want %q", got, want)
	}
}

func TestPostCarriesQueryAndAPIVersion(t *testing.T) {
	srv, rec := recordingServer(t, "")
	client, _ := newTestClient(t, srv.URL)

	query := url.Values{"$top": {"50"}}
	if _, err := client.Post(t.Context(), testLocation, "/MyProject/_apis/wit/wiql", query, map[string]string{}); err != nil {
		t.Fatalf("Post() error = %v, want nil", err)
	}

	if want := "7.2"; rec.query.Get("api-version") != want {
		t.Errorf("api-version = %q, want %q", rec.query.Get("api-version"), want)
	}
	if want := "50"; rec.query.Get("$top") != want {
		t.Errorf("$top = %q, want %q", rec.query.Get("$top"), want)
	}
	if _, ok := query["api-version"]; ok {
		t.Errorf("caller query gained an api-version key: %v", query)
	}
}

func TestGetSendsNoContentType(t *testing.T) {
	srv, rec := recordingServer(t, "")
	client, _ := newTestClient(t, srv.URL)

	if _, err := client.Get(t.Context(), testLocation, "/MyProject/_apis/wit/workitems/123", nil); err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}

	if rec.contentType != "" {
		t.Errorf("Content-Type = %q, want none for a request without a body", rec.contentType)
	}
}

func TestPostClassifiesResponseStatus(t *testing.T) {
	const body = `{"$id":"1","message":"TF401232: Work item 99999 does not exist.","typeKey":"WorkItemNotFoundException"}`
	srv := errorServer(t, http.StatusNotFound, body)
	client, _ := newTestClient(t, srv.URL)

	_, err := client.Post(t.Context(), testLocation, "/MyProject/_apis/wit/workitemsbatch", nil, map[string]string{})
	te := assertTFSError(t, err, tfserr.NotFound)
	if te.HTTPStatus != http.StatusNotFound {
		t.Errorf("HTTPStatus = %d, want %d", te.HTTPStatus, http.StatusNotFound)
	}
	if want := "TF401232: Work item 99999 does not exist."; te.Message != want {
		t.Errorf("message = %q, want %q", te.Message, want)
	}
}

func TestPostRejectsUnencodableBody(t *testing.T) {
	srv, rec := recordingServer(t, "")
	client, _ := newTestClient(t, srv.URL)

	_, err := client.Post(t.Context(), testLocation, "/MyProject/_apis/wit/workitemsbatch", nil, make(chan int))

	_ = assertTFSError(t, err, tfserr.Config)
	if rec.method != "" {
		t.Errorf("a %s request was sent, want none", rec.method)
	}
}
