package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/dpleshakov/tfscli/internal/tfserr"
)

// Config is the effective configuration after merging the credential, the
// config file, environment variables, and command-line flags.
type Config struct {
	// URL and PAT come only from the credential (auth.json or TFSCLI_AUTH),
	// never from the config file or a flag, so that the token cannot be
	// paired with a server it was not stored for.
	URL string `json:"-"`
	PAT string `json:"-"`

	Collection string `json:"collection"`
	Project    string `json:"project"`
	// APIVersion is sent as api-version when set. Empty means no version is
	// sent and the server answers at the version it chooses.
	APIVersion string `json:"apiVersion"`
	// APIVersionSource names the setting APIVersion came from, as the user
	// would refer to it: "--api-version", "TFSCLI_API_VERSION", or
	// "apiVersion" in the config file with its path. It is empty when no
	// version is set, and lets an error about a refused version name the
	// setting to change.
	APIVersionSource string `json:"-"`

	// InsecureSkipVerify disables TLS certificate verification. Config file
	// only: there is deliberately no environment variable and no flag for it,
	// so that turning verification off is always a written-down decision.
	InsecureSkipVerify bool `json:"insecureSkipVerify"`
	// CABundle is the path to a PEM bundle appended to the system root pool.
	// Config file only, for the same reason.
	CABundle string `json:"caBundle"`
}

// Overrides carries command-line flag values. A nil field means the flag was
// not set and must not override anything; the caller fills only the flags the
// user actually passed.
type Overrides struct {
	Collection *string
	Project    *string
	APIVersion *string
}

// DefaultPath returns the path of the default config file,
// $XDG_CONFIG_HOME/tfscli/config.json, falling back to
// ~/.config/tfscli/config.json. The XDG Base Directory rules apply on every
// OS, Windows included: one location across platforms is deliberate.
func DefaultPath() (string, error) {
	dir, err := xdgDir("XDG_CONFIG_HOME", ".config")
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "tfscli", "config.json"), nil
}

// xdgDir returns the directory named by the XDG variable env, or the
// fallback directory under the home directory when the variable is unset,
// empty, or a relative path, which the XDG specification declares invalid.
func xdgDir(env string, fallback ...string) (string, error) {
	if dir := os.Getenv(env); filepath.IsAbs(dir) {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", &tfserr.Error{
			Category: tfserr.Config,
			Message:  "cannot determine the home directory",
			Cause:    err,
		}
	}
	return filepath.Join(append([]string{home}, fallback...)...), nil
}

// Load resolves the credential (TFSCLI_AUTH, or the auth file at authPath),
// reads the config file at path, overlays environment variables, overlays
// explicitly-set flag values, and validates the
// fields every command needs. The config file is optional: the environment
// and flags can supply everything it holds.
//
// Project is not validated here; commands that need it call RequireProject.
func Load(path, authPath string, ov Overrides) (*Config, error) {
	auth, err := LoadAuth(authPath)
	if err != nil {
		return nil, err
	}

	cfg, err := LoadFile(path)
	if err != nil {
		return nil, err
	}
	cfg.URL = auth.URL
	cfg.PAT = auth.PAT

	applyEnv(cfg)
	applyOverrides(cfg, ov)
	// An empty --api-version clears the version, and with it the source.
	if cfg.APIVersion == "" {
		cfg.APIVersionSource = ""
	}

	if cfg.Collection == "" {
		return nil, &tfserr.Error{
			Category: tfserr.Config,
			Message: fmt.Sprintf("collection is not set (pass --collection, set TFSCLI_COLLECTION, or add \"collection\" to %s)",
				path),
		}
	}
	return cfg, nil
}

// RequireProject reports a config error when no project was supplied by flag,
// environment, or config file.
func (c *Config) RequireProject() error {
	if c.Project == "" {
		return &tfserr.Error{
			Category: tfserr.Config,
			Message:  "project is not set (pass -p, set TFSCLI_PROJECT, or add \"project\" to the config file)",
		}
	}
	return nil
}

// LoadFile reads the config file at path, without the credential, the
// environment, or flags. An absent file yields an empty configuration and no
// error. It serves on its own the commands that run
// before a credential exists.
func LoadFile(path string) (*Config, error) {
	cfg := &Config{}
	data, err := os.ReadFile(path) //nolint:gosec // the path is the caller's own config file, by design
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return nil, &tfserr.Error{
			Category: tfserr.Config,
			Message:  fmt.Sprintf("cannot read config file at %s", path),
			Cause:    err,
		}
	default:
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, &tfserr.Error{
				Category: tfserr.Config,
				Message:  fmt.Sprintf("config file at %s is not valid JSON", path),
				Cause:    err,
			}
		}
	}

	if cfg.APIVersion != "" {
		cfg.APIVersionSource = fmt.Sprintf("%q in %s", "apiVersion", path)
	}
	return cfg, nil
}

func applyEnv(cfg *Config) {
	setFromEnv(&cfg.Collection, "TFSCLI_COLLECTION")
	setFromEnv(&cfg.Project, "TFSCLI_PROJECT")
	if setFromEnv(&cfg.APIVersion, "TFSCLI_API_VERSION") {
		cfg.APIVersionSource = "TFSCLI_API_VERSION"
	}
}

// setFromEnv overrides dst when the variable is set to a non-empty value, and
// reports whether it did. An empty variable counts as unset rather than as an
// override with "".
func setFromEnv(dst *string, name string) bool {
	v := os.Getenv(name)
	if v == "" {
		return false
	}
	*dst = v
	return true
}

func applyOverrides(cfg *Config, ov Overrides) {
	setFromFlag(&cfg.Collection, ov.Collection)
	setFromFlag(&cfg.Project, ov.Project)
	if setFromFlag(&cfg.APIVersion, ov.APIVersion) {
		cfg.APIVersionSource = "--api-version"
	}
}

// setFromFlag overrides dst when the flag was set, and reports whether it did.
func setFromFlag(dst *string, flag *string) bool {
	if flag == nil {
		return false
	}
	*dst = *flag
	return true
}
