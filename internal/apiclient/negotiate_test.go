package apiclient

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"slices"
	"testing"

	"github.com/dpleshakov/tfscli/internal/config"
	"github.com/dpleshakov/tfscli/internal/tfserr"
)

const (
	testLocation = "908509b6-4248-4475-a1cd-829139ba419f"

	// optionsBody is a response to OPTIONS on _apis listing a released
	// location, the one under test, and a preview-only one.
	optionsBody = `{"count":3,"value":[
		{"id":"72c7ddf8-2cdc-4f60-90cd-ab71c14a399b","area":"wit","resourceName":"workItems","minVersion":"1.0","maxVersion":"7.1","releasedVersion":"7.1","resourceVersion":3},
		{"id":"908509B6-4248-4475-A1CD-829139BA419F","area":"wit","resourceName":"workItemsBatch","minVersion":"5.0","maxVersion":"7.1","releasedVersion":"7.0","resourceVersion":1},
		{"id":"1a9c53f7-f243-4447-b110-35ef023636e4","area":"wit","resourceName":"wiql","minVersion":"1.0","maxVersion":"7.1","releasedVersion":"0.0","resourceVersion":2}
	]}`
)

func TestChooseVersion(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		location string
		want     string
	}{
		{
			name:     "released location, id in another case",
			body:     optionsBody,
			location: testLocation,
			want:     "7.0",
		},
		{
			name:     "preview-only location",
			body:     optionsBody,
			location: "1a9c53f7-f243-4447-b110-35ef023636e4",
			want:     "7.1-preview.2",
		},
		{
			name:     "preview-only location without a resource version",
			body:     `{"count":1,"value":[{"id":"` + testLocation + `","maxVersion":"5.0","releasedVersion":"0.0"}]}`,
			location: testLocation,
			want:     "5.0-preview",
		},
		{
			name:     "missing released version",
			body:     `{"count":1,"value":[{"id":"` + testLocation + `","maxVersion":"5.0","resourceVersion":1}]}`,
			location: testLocation,
			want:     "5.0-preview.1",
		},
		{
			name:     "byte order mark",
			body:     "\xef\xbb\xbf" + optionsBody,
			location: testLocation,
			want:     "7.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, reason := chooseVersion([]byte(tt.body), tt.location)
			if got != tt.want || reason != "" {
				t.Errorf("chooseVersion() = %q, %q; want %q, \"\"", got, reason, tt.want)
			}
		})
	}
}

func TestChooseVersionFailures(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		reason string
	}{
		{
			name:   "location not listed",
			body:   `{"count":1,"value":[{"id":"72c7ddf8-2cdc-4f60-90cd-ab71c14a399b","maxVersion":"7.1","releasedVersion":"7.1"}]}`,
			reason: "the server does not list this resource among its API versions",
		},
		{
			name:   "no version for the location",
			body:   `{"count":1,"value":[{"id":"` + testLocation + `","releasedVersion":"0.0"}]}`,
			reason: "the server lists no version for this resource",
		},
		{
			name:   "html page",
			body:   "<html><body>Sign in</body></html>",
			reason: "the server's list of API versions could not be read",
		},
		{
			name:   "json without a value array",
			body:   `{"message":"something else"}`,
			reason: "the server's list of API versions could not be read",
		},
		{
			name:   "empty body",
			body:   "",
			reason: "the server's list of API versions could not be read",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, reason := chooseVersion([]byte(tt.body), testLocation)
			if got != "" || reason != tt.reason {
				t.Errorf("chooseVersion() = %q, %q; want \"\", %q", got, reason, tt.reason)
			}
		})
	}
}

func TestIsReleased(t *testing.T) {
	tests := map[string]bool{
		"":    false,
		"0.0": false,
		"0":   false,
		"1.0": true,
		"7.1": true,
		"0.1": true,
	}
	for v, want := range tests {
		if got := isReleased(v); got != want {
			t.Errorf("isReleased(%q) = %v, want %v", v, got, want)
		}
	}
}

// negotiation is what a negotiationServer saw.
type negotiation struct {
	// requests lists each request as method and path, in order.
	requests []string
	// query is the query of the last request other than OPTIONS.
	query url.Values
}

// negotiationServer answers OPTIONS with optionsStatus and optionsReply, and
// every other request with an empty JSON object.
func negotiationServer(t *testing.T, optionsStatus int, optionsReply string) (*httptest.Server, *negotiation) {
	t.Helper()
	rec := &negotiation{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.requests = append(rec.requests, r.Method+" "+r.URL.Path)
		if r.Method == http.MethodOptions {
			w.WriteHeader(optionsStatus)
			_, _ = w.Write([]byte(optionsReply))
			return
		}
		rec.query = r.URL.Query()
		_, _ = w.Write([]byte("{}"))
	}))
	t.Cleanup(srv.Close)
	return srv, rec
}

// newUnversionedClient builds a Client with no API version configured, so
// that it negotiates one.
func newUnversionedClient(t *testing.T, serverURL string) (*Client, *recordingLogger) {
	t.Helper()
	logger := &recordingLogger{}
	client, err := New(&config.Config{
		URL:        serverURL,
		Collection: "DefaultCollection",
		PAT:        testPAT,
	}, logger)
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}
	return client, logger
}

func TestNegotiatesVersion(t *testing.T) {
	tests := []struct {
		name string
		send func(*Client) error
		want []string
	}{
		{
			name: "GET",
			send: func(c *Client) error {
				_, err := c.Get(t.Context(), testLocation, "MyProject/_apis/wit/workitems", nil)
				return err
			},
			want: []string{"OPTIONS /DefaultCollection/_apis", "GET /DefaultCollection/MyProject/_apis/wit/workitems"},
		},
		{
			name: "POST",
			send: func(c *Client) error {
				_, err := c.Post(t.Context(), testLocation, "MyProject/_apis/wit/workitemsbatch", nil, map[string]string{})
				return err
			},
			want: []string{"OPTIONS /DefaultCollection/_apis", "POST /DefaultCollection/MyProject/_apis/wit/workitemsbatch"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, rec := negotiationServer(t, http.StatusOK, optionsBody)
			client, logger := newUnversionedClient(t, srv.URL)

			if err := tt.send(client); err != nil {
				t.Fatalf("request error = %v, want nil", err)
			}

			if !slices.Equal(rec.requests, tt.want) {
				t.Errorf("requests = %v, want %v", rec.requests, tt.want)
			}
			if got := rec.query.Get("api-version"); got != "7.0" {
				t.Errorf("api-version = %q, want the negotiated %q", got, "7.0")
			}
			if len(logger.requests) != 2 {
				t.Errorf("logged %d requests, want 2 — OPTIONS is logged like any other", len(logger.requests))
			}
			if len(logger.infos) != 0 {
				t.Errorf("infos = %v, want none after a successful negotiation", logger.infos)
			}
		})
	}
}

func TestConfiguredVersionSkipsNegotiation(t *testing.T) {
	srv, rec := negotiationServer(t, http.StatusOK, optionsBody)
	client, _ := newTestClient(t, srv.URL)

	if _, err := client.Post(t.Context(), testLocation, "MyProject/_apis/wit/workitemsbatch", nil, map[string]string{}); err != nil {
		t.Fatalf("Post() error = %v, want nil", err)
	}

	if want := []string{"POST /DefaultCollection/MyProject/_apis/wit/workitemsbatch"}; !slices.Equal(rec.requests, want) {
		t.Errorf("requests = %v, want %v", rec.requests, want)
	}
	if got := rec.query.Get("api-version"); got != "7.2" {
		t.Errorf("api-version = %q, want the configured %q", got, "7.2")
	}
}

func TestNegotiationFailsOnUnauthorized(t *testing.T) {
	srv, rec := negotiationServer(t, http.StatusUnauthorized, "")
	client, _ := newUnversionedClient(t, srv.URL)

	_, err := client.Get(t.Context(), testLocation, "MyProject/_apis/wit/workitems", nil)

	te := assertTFSError(t, err, tfserr.Auth)
	if te.HTTPStatus != http.StatusUnauthorized {
		t.Errorf("HTTPStatus = %d, want %d", te.HTTPStatus, http.StatusUnauthorized)
	}
	if want := []string{"OPTIONS /DefaultCollection/_apis"}; !slices.Equal(rec.requests, want) {
		t.Errorf("requests = %v, want %v — the request itself is not sent", rec.requests, want)
	}
}

func TestNegotiationFailsOnTransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	serverURL := srv.URL
	srv.Close()
	client, logger := newUnversionedClient(t, serverURL)

	_, err := client.Get(t.Context(), testLocation, "MyProject/_apis/wit/workitems", nil)

	_ = assertTFSError(t, err, tfserr.Network)
	if len(logger.requests) != 1 || logger.requests[0].method != http.MethodOptions {
		t.Errorf("logged requests = %v, want the OPTIONS request alone", logger.requests)
	}
}

func TestNegotiationFallsBack(t *testing.T) {
	tests := []struct {
		name   string
		status int
		reply  string
		info   string
	}{
		{"bad request", http.StatusBadRequest, "", "api-version not negotiated (OPTIONS returned HTTP 400); sending the request without it"},
		{"forbidden", http.StatusForbidden, "", "api-version not negotiated (OPTIONS returned HTTP 403); sending the request without it"},
		{"not found", http.StatusNotFound, "", "api-version not negotiated (OPTIONS returned HTTP 404); sending the request without it"},
		{"method not allowed", http.StatusMethodNotAllowed, "", "api-version not negotiated (OPTIONS returned HTTP 405); sending the request without it"},
		{"server error", http.StatusInternalServerError, "", "api-version not negotiated (OPTIONS returned HTTP 500); sending the request without it"},
		{"unparseable", http.StatusOK, "<html>Sign in</html>", "api-version not negotiated (the server's list of API versions could not be read); sending the request without it"},
		{"location missing", http.StatusOK, `{"count":0,"value":[]}`, "api-version not negotiated (the server does not list this resource among its API versions); sending the request without it"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, rec := negotiationServer(t, tt.status, tt.reply)
			client, logger := newUnversionedClient(t, srv.URL)

			if _, err := client.Post(t.Context(), testLocation, "MyProject/_apis/wit/workitemsbatch", nil, map[string]string{}); err != nil {
				t.Fatalf("Post() error = %v, want nil", err)
			}

			if len(rec.requests) != 2 {
				t.Fatalf("requests = %v, want OPTIONS and then the request", rec.requests)
			}
			if _, ok := rec.query["api-version"]; ok {
				t.Errorf("query = %v, want no api-version after a fallback", rec.query)
			}
			if want := []string{tt.info}; !slices.Equal(logger.infos, want) {
				t.Errorf("infos = %q, want %q", logger.infos, want)
			}
		})
	}
}

const missingVersionReply = `{"$id":"1","message":"No api-version was supplied for the \"POST\" request.","typeKey":"VssVersionNotSpecifiedException"}`

// refusingServer answers OPTIONS with optionsStatus and every other request
// with HTTP 400 and reply.
func refusingServer(t *testing.T, optionsStatus int, reply string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(optionsStatus)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestMissingVersionAfterFallbackNamesTheSettings(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	cfgPath := filepath.Join(cfgHome, "tfscli", "config.json")
	step := ` (the API version could not be negotiated: OPTIONS returned HTTP 405; ` +
		`set one with "apiVersion" in ` + cfgPath + `, TFSCLI_API_VERSION, or --api-version, ` +
		`no higher than the server supports (see "REST API version" in the tfscli README))`

	tests := []struct {
		name  string
		reply string
		want  string
	}{
		{
			name:  "typeKey",
			reply: missingVersionReply,
			want:  `No api-version was supplied for the "POST" request.` + step,
		},
		{
			name:  "message without a typeKey",
			reply: `{"message":"No api-version was supplied for the \"POST\" request."}`,
			want:  `No api-version was supplied for the "POST" request.` + step,
		},
		{
			name:  "another refusal",
			reply: `{"message":"TF51005: The query references a field that does not exist.","typeKey":"QueryException"}`,
			want:  "TF51005: The query references a field that does not exist.",
		},
		{
			name:  "message of another error without a typeKey",
			reply: `{"message":"Some other problem."}`,
			want:  "Some other problem.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := refusingServer(t, http.StatusMethodNotAllowed, tt.reply)
			client, _ := newUnversionedClient(t, srv.URL)

			_, err := client.Post(t.Context(), testLocation, "_apis/wit/wiql", nil, map[string]string{})

			te := assertTFSError(t, err, tfserr.Config)
			if te.Message != tt.want {
				t.Errorf("message = %q, want %q", te.Message, tt.want)
			}
		})
	}
}

func TestMissingVersionWithoutFallbackKeepsTheMessage(t *testing.T) {
	tests := []struct {
		name     string
		location string
	}{
		// No location: the request was meant to go without a version.
		{"no location", ""},
		// A wrong typeKey on a negotiated request is not a missing version.
		{"negotiated", testLocation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodOptions {
					_, _ = w.Write([]byte(optionsBody))
					return
				}
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(missingVersionReply))
			}))
			t.Cleanup(srv.Close)
			client, _ := newUnversionedClient(t, srv.URL)

			_, err := client.Post(t.Context(), tt.location, "_apis/wit/wiql", nil, map[string]string{})

			te := assertTFSError(t, err, tfserr.Config)
			if want := `No api-version was supplied for the "POST" request.`; te.Message != want {
				t.Errorf("message = %q, want %q", te.Message, want)
			}
		})
	}
}
