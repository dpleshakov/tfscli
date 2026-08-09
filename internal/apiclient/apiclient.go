package apiclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/dpleshakov/tfscli/internal/config"
	"github.com/dpleshakov/tfscli/internal/log"
	"github.com/dpleshakov/tfscli/internal/tfserr"
)

// maxMessageRunes bounds how much of a server-supplied error message is
// repeated back to the user, so that an HTML error page or a stack trace
// cannot flood stderr.
const maxMessageRunes = 300

// Client performs TFS REST API calls. It owns the PAT and is the only module
// that sets the Authorization header.
type Client struct {
	http *http.Client
	// base is Config.URL joined with Config.Collection; per-call paths are
	// joined onto it.
	base       *url.URL
	pat        string
	apiVersion string
}

// New builds a Client from cfg. TLS is configured from CABundle (appended to
// the system root pool) and InsecureSkipVerify; when verification is disabled
// the logger receives one warning. Every request made by the returned Client
// is logged exactly once, from the transport below — this package is the only
// caller of LogRequest.
func New(cfg *config.Config, logger log.Logger) (*Client, error) {
	base, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, &tfserr.Error{
			Category: tfserr.Config,
			Message:  fmt.Sprintf("url %q is not a valid URL", cfg.URL),
			Cause:    err,
		}
	}
	if base.Scheme != "http" && base.Scheme != "https" {
		return nil, &tfserr.Error{
			Category: tfserr.Config,
			Message:  fmt.Sprintf("url %q must start with http:// or https://", cfg.URL),
		}
	}

	tlsConfig, err := newTLSConfig(cfg)
	if err != nil {
		return nil, err
	}
	if cfg.InsecureSkipVerify {
		logger.Warn("TLS verification disabled")
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = tlsConfig

	return &Client{
		http:       &http.Client{Transport: &loggingTransport{base: transport, logger: logger}},
		base:       base.JoinPath(cfg.Collection),
		pat:        cfg.PAT,
		apiVersion: cfg.APIVersion,
	}, nil
}

// Get performs a GET request against path, which is appended to the base URL
// (server URL plus collection). The api-version parameter is added here, so
// callers pass only their own query parameters. On a non-2xx response or a
// transport failure the error is a *tfserr.Error carrying the matching
// category.
func (c *Client) Get(ctx context.Context, path string, query url.Values) ([]byte, error) {
	u := c.base.JoinPath(path)
	q := maps.Clone(query)
	if q == nil {
		q = url.Values{}
	}
	q.Set("api-version", c.apiVersion)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, &tfserr.Error{
			Category: tfserr.Config,
			Message:  fmt.Sprintf("cannot build a request for %s", u.Redacted()),
			Cause:    err,
		}
	}
	// TFS expects the PAT as the password of an empty user name; this is the
	// single place where it is attached to a request.
	req.SetBasicAuth("", c.pat)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, transportError(u, err)
	}
	// The body is read in full below; a failure to close it afterwards says
	// nothing about the result and has nowhere useful to go.
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &tfserr.Error{
			Category:   tfserr.Server,
			Message:    "cannot read the response from TFS",
			HTTPStatus: resp.StatusCode,
			Cause:      err,
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, classify(resp.StatusCode, body)
	}
	return body, nil
}

// loggingTransport logs one line per request and is the single call site of
// LogRequest. It only observes the request; the round trip itself is
// delegated unchanged.
type loggingTransport struct {
	base   http.RoundTripper
	logger log.Logger
}

// RoundTrip logs the request and its outcome, and returns what the base
// transport returned.
func (t *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := t.base.RoundTrip(req)
	// A failed round trip has no status; it is logged as 0 so that the
	// attempt still shows up under --verbose.
	status := 0
	if resp != nil {
		status = resp.StatusCode
	}
	t.logger.LogRequest(req.Method, req.URL.Redacted(), status, time.Since(start))
	return resp, err
}

func newTLSConfig(cfg *config.Config) (*tls.Config, error) {
	if !cfg.InsecureSkipVerify && cfg.CABundle == "" {
		return nil, nil
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		// Set from the config file only, so that disabling verification is
		// always a written-down decision.
		InsecureSkipVerify: cfg.InsecureSkipVerify, //nolint:gosec // documented opt-in for servers behind an internal CA
	}
	if cfg.CABundle == "" {
		return tlsConfig, nil
	}

	pem, err := os.ReadFile(cfg.CABundle)
	if err != nil {
		return nil, &tfserr.Error{
			Category: tfserr.Config,
			Message:  fmt.Sprintf("cannot read the CA bundle at %s", cfg.CABundle),
			Cause:    err,
		}
	}
	pool, err := x509.SystemCertPool()
	if err != nil {
		// Falling back to an empty pool would silently drop every system
		// root, so this is reported instead.
		return nil, &tfserr.Error{
			Category: tfserr.Config,
			Message:  "cannot read the system certificate pool",
			Cause:    err,
		}
	}
	if !pool.AppendCertsFromPEM(pem) {
		return nil, &tfserr.Error{
			Category: tfserr.Config,
			Message:  fmt.Sprintf("the CA bundle at %s contains no certificates", cfg.CABundle),
		}
	}
	tlsConfig.RootCAs = pool
	return tlsConfig, nil
}

// transportError maps a failed round trip to category network.
func transportError(u *url.URL, err error) error {
	server := u.Scheme + "://" + u.Host
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return &tfserr.Error{
			Category: tfserr.Network,
			Message:  fmt.Sprintf("request to %s timed out", server),
			Cause:    err,
		}
	case errors.Is(err, context.Canceled):
		return &tfserr.Error{
			Category: tfserr.Network,
			Message:  fmt.Sprintf("request to %s was canceled", server),
			Cause:    err,
		}
	default:
		return &tfserr.Error{
			Category: tfserr.Network,
			Message:  fmt.Sprintf("cannot reach %s", server),
			Cause:    err,
		}
	}
}

// classify maps a non-2xx response to a stable error category, preferring the
// message from the TFS error JSON over the generic wording when the body
// carries one.
func classify(status int, body []byte) error {
	category, generic := categoryFor(status)
	message := generic
	if fromServer := serverMessage(body); fromServer != "" {
		message = fromServer
	}
	return &tfserr.Error{
		Category:   category,
		Message:    message,
		HTTPStatus: status,
	}
}

func categoryFor(status int) (tfserr.Category, string) {
	switch {
	case status == http.StatusUnauthorized:
		return tfserr.Auth, "PAT is invalid or expired"
	case status == http.StatusForbidden:
		return tfserr.Forbidden, "access denied by TFS"
	case status == http.StatusNotFound:
		return tfserr.NotFound, "resource not found"
	case status >= 500:
		return tfserr.Server, "TFS returned a server error"
	case status >= 400:
		// Remaining 4xx responses mean TFS rejected what we sent — a bad API
		// version, an unknown field, a malformed id. All of these come from
		// configuration or arguments, so they land in the config category.
		return tfserr.Config, "TFS rejected the request"
	default:
		return tfserr.Server, "TFS returned an unexpected response"
	}
}

// serverMessage extracts the "message" field of a TFS error response. A body
// that is not TFS error JSON — an HTML error page from a proxy, say — yields
// an empty string so that the generic wording is used instead.
func serverMessage(body []byte) string {
	var payload struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	return sanitize(payload.Message)
}

// sanitize collapses a server-supplied message into a single bounded line.
func sanitize(s string) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return ' '
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
	s = strings.Join(strings.Fields(s), " ")

	if runes := []rune(s); len(runes) > maxMessageRunes {
		s = strings.TrimSpace(string(runes[:maxMessageRunes])) + "..."
	}
	return s
}
