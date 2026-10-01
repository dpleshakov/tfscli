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

// DefaultAPIVersion is used when neither the config file, the environment, nor
// a flag specifies an API version.
const DefaultAPIVersion = "7.2"

// Config is the effective configuration after merging the config file,
// environment variables, and command-line flags.
type Config struct {
	URL        string `json:"url"`
	Collection string `json:"collection"`
	PAT        string `json:"pat"`
	Project    string `json:"project"`
	APIVersion string `json:"apiVersion"`

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
	URL        *string
	Collection *string
	PAT        *string
	Project    *string
	APIVersion *string
}

// DefaultPath returns the path of the default config file,
// $XDG_CONFIG_HOME/tfscli/config.json, falling back to
// ~/.config/tfscli/config.json. The XDG Base Directory rules apply on every
// OS, Windows included: one location across platforms is deliberate.
func DefaultPath() (string, error) {
	dir, err := configHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "tfscli", "config.json"), nil
}

// configHome returns $XDG_CONFIG_HOME, or ~/.config when the variable is
// unset, empty, or a relative path, which the XDG specification declares
// invalid.
func configHome() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); filepath.IsAbs(dir) {
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
	return filepath.Join(home, ".config"), nil
}

// Load reads the config file at path, overlays environment variables, overlays
// explicitly-set flag values, applies built-in defaults, and validates the
// fields every command needs. A missing config file is not an error in itself:
// the environment and flags may supply everything. It only surfaces — as
// "config file not found at <path>" — when something required is in fact
// missing, since that is then the most useful thing to tell the user.
//
// Project is not validated here; commands that need it call RequireProject.
func Load(path string, ov Overrides) (*Config, error) {
	cfg, fileFound, err := loadFile(path)
	if err != nil {
		return nil, err
	}

	applyEnv(cfg)
	applyOverrides(cfg, ov)

	if cfg.APIVersion == "" {
		cfg.APIVersion = DefaultAPIVersion
	}

	if err := cfg.validate(path, fileFound); err != nil {
		return nil, err
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

// loadFile reads and parses the config file. It reports whether the file
// exists; an absent file yields an empty config and no error.
func loadFile(path string) (*Config, bool, error) {
	data, err := os.ReadFile(path) //nolint:gosec // the path is the caller's own config file, by design
	if errors.Is(err, fs.ErrNotExist) {
		return &Config{}, false, nil
	}
	if err != nil {
		return nil, false, &tfserr.Error{
			Category: tfserr.Config,
			Message:  fmt.Sprintf("cannot read config file at %s", path),
			Cause:    err,
		}
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, false, &tfserr.Error{
			Category: tfserr.Config,
			Message:  fmt.Sprintf("config file at %s is not valid JSON", path),
			Cause:    err,
		}
	}
	return &cfg, true, nil
}

func applyEnv(cfg *Config) {
	setFromEnv(&cfg.URL, "TFSCLI_URL")
	setFromEnv(&cfg.Collection, "TFSCLI_COLLECTION")
	setFromEnv(&cfg.PAT, "TFSCLI_PAT")
	setFromEnv(&cfg.Project, "TFSCLI_PROJECT")
	setFromEnv(&cfg.APIVersion, "TFSCLI_API_VERSION")
}

// setFromEnv overrides dst when the variable is set to a non-empty value. An
// empty variable counts as unset rather than as an override with "".
func setFromEnv(dst *string, name string) {
	if v := os.Getenv(name); v != "" {
		*dst = v
	}
}

func applyOverrides(cfg *Config, ov Overrides) {
	setFromFlag(&cfg.URL, ov.URL)
	setFromFlag(&cfg.Collection, ov.Collection)
	setFromFlag(&cfg.PAT, ov.PAT)
	setFromFlag(&cfg.Project, ov.Project)
	setFromFlag(&cfg.APIVersion, ov.APIVersion)
}

func setFromFlag(dst *string, flag *string) {
	if flag != nil {
		*dst = *flag
	}
}

func (c *Config) validate(path string, fileFound bool) error {
	required := []struct {
		value string
		name  string
		env   string
		flag  string
	}{
		{c.URL, "url", "TFSCLI_URL", "--url"},
		{c.Collection, "collection", "TFSCLI_COLLECTION", "--collection"},
		{c.PAT, "pat", "TFSCLI_PAT", "--pat"},
	}

	for _, f := range required {
		if f.value != "" {
			continue
		}
		if !fileFound {
			return &tfserr.Error{
				Category: tfserr.Config,
				Message:  fmt.Sprintf("config file not found at %s", path),
			}
		}
		return &tfserr.Error{
			Category: tfserr.Config,
			Message: fmt.Sprintf("%s is not set (add %q to the config file, set %s, or pass %s)",
				f.name, f.name, f.env, f.flag),
		}
	}
	return nil
}
