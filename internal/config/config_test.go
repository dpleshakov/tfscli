package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dpleshakov/tfscli/internal/tfserr"
)

// envNames lists every variable the package reads, so that tests can start
// from a known-empty environment regardless of the developer's own settings.
var envNames = []string{
	"TFSCLI_URL",
	"TFSCLI_COLLECTION",
	"TFSCLI_PAT",
	"TFSCLI_PROJECT",
	"TFSCLI_API_VERSION",
}

func clearEnv(t *testing.T) {
	t.Helper()
	for _, name := range envNames {
		t.Setenv(name, "")
	}
}

// writeConfig writes body to a config file inside a fresh temp directory and
// returns its path.
func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("writing config file: %v", err)
	}
	return path
}

// missingConfig returns a path inside a fresh temp directory where no file
// exists.
func missingConfig(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "config.json")
}

const fullConfig = `{
  "url": "https://file.example.com/tfs",
  "collection": "FileCollection",
  "pat": "file-pat",
  "project": "FileProject",
  "apiVersion": "6.0"
}`

func TestLoadPrecedence(t *testing.T) {
	tests := []struct {
		name      string
		file      string
		env       map[string]string
		overrides Overrides
		want      Config
	}{
		{
			name: "file only",
			file: fullConfig,
			want: Config{
				URL:        "https://file.example.com/tfs",
				Collection: "FileCollection",
				PAT:        "file-pat",
				Project:    "FileProject",
				APIVersion: "6.0",
			},
		},
		{
			name: "env overrides file",
			file: fullConfig,
			env: map[string]string{
				"TFSCLI_URL":         "https://env.example.com/tfs",
				"TFSCLI_COLLECTION":  "EnvCollection",
				"TFSCLI_PAT":         "env-pat",
				"TFSCLI_PROJECT":     "EnvProject",
				"TFSCLI_API_VERSION": "7.0",
			},
			want: Config{
				URL:        "https://env.example.com/tfs",
				Collection: "EnvCollection",
				PAT:        "env-pat",
				Project:    "EnvProject",
				APIVersion: "7.0",
			},
		},
		{
			name: "flag overrides env and file",
			file: fullConfig,
			env: map[string]string{
				"TFSCLI_URL":         "https://env.example.com/tfs",
				"TFSCLI_COLLECTION":  "EnvCollection",
				"TFSCLI_PAT":         "env-pat",
				"TFSCLI_PROJECT":     "EnvProject",
				"TFSCLI_API_VERSION": "7.0",
			},
			overrides: Overrides{
				URL:        new("https://flag.example.com/tfs"),
				Collection: new("FlagCollection"),
				PAT:        new("flag-pat"),
				Project:    new("FlagProject"),
				APIVersion: new("7.1"),
			},
			want: Config{
				URL:        "https://flag.example.com/tfs",
				Collection: "FlagCollection",
				PAT:        "flag-pat",
				Project:    "FlagProject",
				APIVersion: "7.1",
			},
		},
		{
			name: "api version falls back to the built-in default",
			file: `{"url": "https://file.example.com/tfs", "collection": "FileCollection", "pat": "file-pat"}`,
			want: Config{
				URL:        "https://file.example.com/tfs",
				Collection: "FileCollection",
				PAT:        "file-pat",
				APIVersion: DefaultAPIVersion,
			},
		},
		{
			name: "each source wins for its own field",
			file: fullConfig,
			env: map[string]string{
				"TFSCLI_COLLECTION": "EnvCollection",
				"TFSCLI_PROJECT":    "EnvProject",
			},
			overrides: Overrides{Project: new("FlagProject")},
			want: Config{
				URL:        "https://file.example.com/tfs",
				Collection: "EnvCollection",
				PAT:        "file-pat",
				Project:    "FlagProject",
				APIVersion: "6.0",
			},
		},
		{
			name: "environment alone is enough without a config file",
			env: map[string]string{
				"TFSCLI_URL":        "https://env.example.com/tfs",
				"TFSCLI_COLLECTION": "EnvCollection",
				"TFSCLI_PAT":        "env-pat",
			},
			want: Config{
				URL:        "https://env.example.com/tfs",
				Collection: "EnvCollection",
				PAT:        "env-pat",
				APIVersion: DefaultAPIVersion,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			for name, value := range tt.env {
				t.Setenv(name, value)
			}

			path := missingConfig(t)
			if tt.file != "" {
				path = writeConfig(t, tt.file)
			}

			got, err := Load(path, tt.overrides)
			if err != nil {
				t.Fatalf("Load() error = %v, want nil", err)
			}
			if *got != tt.want {
				t.Errorf("Load() = %+v, want %+v", *got, tt.want)
			}
		})
	}
}

func TestLoadEmptyEnvDoesNotOverride(t *testing.T) {
	clearEnv(t)
	path := writeConfig(t, fullConfig)

	got, err := Load(path, Overrides{})
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if got.URL != "https://file.example.com/tfs" {
		t.Errorf("URL = %q, want the config file value", got.URL)
	}
}

func TestLoadReadsTLSFieldsFromFile(t *testing.T) {
	clearEnv(t)
	path := writeConfig(t, `{
	  "url": "https://file.example.com/tfs",
	  "collection": "FileCollection",
	  "pat": "file-pat",
	  "insecureSkipVerify": true,
	  "caBundle": "C:/certs/corp.pem"
	}`)

	got, err := Load(path, Overrides{})
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if !got.InsecureSkipVerify {
		t.Errorf("InsecureSkipVerify = false, want true")
	}
	if want := "C:/certs/corp.pem"; got.CABundle != want {
		t.Errorf("CABundle = %q, want %q", got.CABundle, want)
	}
}

func TestLoadMissingFile(t *testing.T) {
	clearEnv(t)
	path := missingConfig(t)

	_, err := Load(path, Overrides{})
	assertConfigError(t, err, "config file not found at "+path)
}

func TestLoadMalformedJSON(t *testing.T) {
	clearEnv(t)
	path := writeConfig(t, `{"url": "https://file.example.com/tfs",`)

	_, err := Load(path, Overrides{})
	assertConfigError(t, err, "is not valid JSON")
}

func TestLoadMissingRequiredField(t *testing.T) {
	tests := []struct {
		name string
		file string
		want string
	}{
		{
			name: "url",
			file: `{"collection": "FileCollection", "pat": "file-pat"}`,
			want: `url is not set (add "url" to the config file, set TFSCLI_URL, or pass --url)`,
		},
		{
			name: "collection",
			file: `{"url": "https://file.example.com/tfs", "pat": "file-pat"}`,
			want: `collection is not set (add "collection" to the config file, set TFSCLI_COLLECTION, or pass --collection)`,
		},
		{
			name: "pat",
			file: `{"url": "https://file.example.com/tfs", "collection": "FileCollection"}`,
			want: `pat is not set (add "pat" to the config file, set TFSCLI_PAT, or pass --pat)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			path := writeConfig(t, tt.file)

			_, err := Load(path, Overrides{})
			assertConfigError(t, err, tt.want)
		})
	}
}

func TestRequireProject(t *testing.T) {
	withProject := &Config{Project: "MyProject"}
	if err := withProject.RequireProject(); err != nil {
		t.Errorf("RequireProject() = %v, want nil", err)
	}

	withoutProject := &Config{}
	assertConfigError(t, withoutProject.RequireProject(), "project is not set")
}

func TestDefaultPath(t *testing.T) {
	xdg := t.TempDir()
	home := t.TempDir()

	tests := []struct {
		name string
		xdg  string
		want string
	}{
		{"XDG_CONFIG_HOME set", xdg, filepath.Join(xdg, "tfscli", "config.json")},
		{"XDG_CONFIG_HOME unset", "", filepath.Join(home, ".config", "tfscli", "config.json")},
		{"XDG_CONFIG_HOME relative", filepath.Join("relative", "dir"), filepath.Join(home, ".config", "tfscli", "config.json")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", tt.xdg)
			// os.UserHomeDir reads HOME on Unix and USERPROFILE on Windows.
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)

			path, err := DefaultPath()
			if err != nil {
				t.Fatalf("DefaultPath() error = %v, want nil", err)
			}
			if path != tt.want {
				t.Errorf("DefaultPath() = %q, want %q", path, tt.want)
			}
		})
	}
}

// assertConfigError checks that err is a *tfserr.Error of category config whose
// message contains want.
func assertConfigError(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want a config error containing %q", want)
	}
	var te *tfserr.Error
	if !errors.As(err, &te) {
		t.Fatalf("error = %v, want a *tfserr.Error", err)
	}
	if te.Category != tfserr.Config {
		t.Errorf("category = %q, want %q", te.Category, tfserr.Config)
	}
	if !strings.Contains(te.Message, want) {
		t.Errorf("message = %q, want it to contain %q", te.Message, want)
	}
}
