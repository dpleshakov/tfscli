package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/dpleshakov/tfscli/internal/tfserr"
)

// AuthEnv names the environment variable that carries the whole credential
// as JSON, for environments where `tfscli auth login` cannot run. When it is
// set, the auth file is not read.
const AuthEnv = "TFSCLI_AUTH"

// Auth is the credential: the server URL, the collection, and the PAT issued
// for that collection. A PAT is issued for one collection, and requests
// outside it are refused, so the three always come from the same source and
// the token is never sent anywhere other than where it was stored for.
type Auth struct {
	URL        string `json:"url"`
	Collection string `json:"collection"`
	PAT        string `json:"pat"`
}

// DefaultAuthPath returns the path of the auth file,
// $XDG_DATA_HOME/tfscli/auth.json, falling back to
// ~/.local/share/tfscli/auth.json, on every OS.
func DefaultAuthPath() (string, error) {
	dir, err := xdgDir("XDG_DATA_HOME", ".local", "share")
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "tfscli", "auth.json"), nil
}

// LoadAuth returns the credential from TFSCLI_AUTH when that variable is set
// to a non-empty value, and from the auth file at path otherwise. The URL is
// returned normalised. Every field is required, and every failure is a config
// error naming the source.
func LoadAuth(path string) (*Auth, error) {
	if v := os.Getenv(AuthEnv); v != "" {
		return parseAuth([]byte(v), AuthEnv)
	}

	data, err := os.ReadFile(path) //nolint:gosec // the path is the caller's own auth file, by design
	if errors.Is(err, fs.ErrNotExist) {
		return nil, &tfserr.Error{
			Category: tfserr.Config,
			Message: fmt.Sprintf("not logged in: no credential at %s (run \"tfscli auth login\", or set %s)",
				path, AuthEnv),
		}
	}
	if err != nil {
		return nil, &tfserr.Error{
			Category: tfserr.Config,
			Message:  fmt.Sprintf("cannot read the auth file at %s", path),
			Cause:    err,
		}
	}
	return parseAuth(data, "the auth file at "+path)
}

// parseAuth decodes and validates a credential; source names where it came
// from for the error messages.
func parseAuth(data []byte, source string) (*Auth, error) {
	var auth Auth
	if err := json.Unmarshal(data, &auth); err != nil {
		return nil, &tfserr.Error{
			Category: tfserr.Config,
			Message:  fmt.Sprintf("%s is not valid JSON", source),
			Cause:    err,
		}
	}

	for _, f := range []struct{ name, value string }{
		{"url", auth.URL}, {"collection", auth.Collection}, {"pat", auth.PAT},
	} {
		if f.value == "" {
			return nil, &tfserr.Error{
				Category: tfserr.Config,
				Message:  fmt.Sprintf("%s has no %q", source, f.name),
			}
		}
	}

	normalized, err := NormalizeURL(auth.URL)
	if err != nil {
		var te *tfserr.Error
		if errors.As(err, &te) {
			te.Message = source + ": " + te.Message
		}
		return nil, err
	}
	auth.URL = normalized
	return &auth, nil
}

// SaveAuth writes the credential to the auth file at path, replacing any
// previous one. The file gets mode 0600 and a directory created for it 0700;
// the content is written to a temporary file first and renamed into place, so
// that a failure leaves the previous credential intact. On Windows the modes
// are not enforced and the file inherits the ACL of the user profile.
func SaveAuth(path string, auth *Auth) error {
	data, err := json.MarshalIndent(auth, "", "  ")
	if err != nil {
		return saveError(path, err)
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return saveError(path, err)
	}
	// CreateTemp creates the file with mode 0600.
	tmp, err := os.CreateTemp(dir, "auth-*.json")
	if err != nil {
		return saveError(path, err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()

	_, err = tmp.Write(data)
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(tmp.Name(), path)
	}
	if err != nil {
		return saveError(path, err)
	}
	return nil
}

func saveError(path string, err error) error {
	return &tfserr.Error{
		Category: tfserr.Config,
		Message:  fmt.Sprintf("cannot write the auth file at %s", path),
		Cause:    err,
	}
}

// NormalizeURL validates a server URL and returns it in the form it is
// stored in: lower-case scheme and host, no trailing slash. Only http and
// https URLs with a host are accepted.
func NormalizeURL(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", &tfserr.Error{
			Category: tfserr.Config,
			Message:  fmt.Sprintf("url %q is not a valid URL", raw),
			Cause:    err,
		}
	}
	// url.Parse already lower-cases the scheme.
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", &tfserr.Error{
			Category: tfserr.Config,
			Message:  fmt.Sprintf("url %q must start with http:// or https://", raw),
		}
	}
	if u.Host == "" {
		return "", &tfserr.Error{
			Category: tfserr.Config,
			Message:  fmt.Sprintf("url %q has no host", raw),
		}
	}

	u.Host = strings.ToLower(u.Host)
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawPath = ""
	return u.String(), nil
}
