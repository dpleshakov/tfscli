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
	"TFSCLI_AUTH",
	"TFSCLI_PROJECT",
	"TFSCLI_API_VERSION",
}

func clearEnv(t *testing.T) {
	t.Helper()
	for _, name := range envNames {
		t.Setenv(name, "")
	}
}

// writeFile writes body to a file with the given name inside a fresh temp
// directory and returns its path.
func writeFile(t *testing.T, name, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
	return path
}

// writeConfig writes body to a config file and returns its path.
func writeConfig(t *testing.T, body string) string {
	t.Helper()
	return writeFile(t, "config.json", body)
}

// writeAuth writes body to an auth file and returns its path.
func writeAuth(t *testing.T, body string) string {
	t.Helper()
	return writeFile(t, "auth.json", body)
}

// missingFile returns a path inside a fresh temp directory where no file
// exists.
func missingFile(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(t.TempDir(), name)
}

const fullConfig = `{
  "project": "FileProject",
  "apiVersion": "6.0"
}`

// fileSource stands for the API version source of the config file in a
// table, whose path is known only once the test has written the file.
const fileSource = "<config file>"

const fileAuth = `{"url": "https://file.example.com/tfs", "collection": "AuthCollection", "pat": "file-pat"}`

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
				URL:              "https://file.example.com/tfs",
				Collection:       "AuthCollection",
				PAT:              "file-pat",
				Project:          "FileProject",
				APIVersion:       "6.0",
				APIVersionSource: fileSource,
			},
		},
		{
			name: "env overrides file",
			file: fullConfig,
			env: map[string]string{
				"TFSCLI_PROJECT":     "EnvProject",
				"TFSCLI_API_VERSION": "7.0",
			},
			want: Config{
				URL:              "https://file.example.com/tfs",
				Collection:       "AuthCollection",
				PAT:              "file-pat",
				Project:          "EnvProject",
				APIVersion:       "7.0",
				APIVersionSource: "TFSCLI_API_VERSION",
			},
		},
		{
			name: "flag overrides env and file",
			file: fullConfig,
			env: map[string]string{
				"TFSCLI_PROJECT":     "EnvProject",
				"TFSCLI_API_VERSION": "7.0",
			},
			overrides: Overrides{
				Project:    new("FlagProject"),
				APIVersion: new("7.1"),
			},
			want: Config{
				URL:              "https://file.example.com/tfs",
				Collection:       "AuthCollection",
				PAT:              "file-pat",
				Project:          "FlagProject",
				APIVersion:       "7.1",
				APIVersionSource: "--api-version",
			},
		},
		{
			name: "no api version is set by default",
			file: `{"project": "FileProject"}`,
			want: Config{
				URL:        "https://file.example.com/tfs",
				Collection: "AuthCollection",
				PAT:        "file-pat",
				Project:    "FileProject",
			},
		},
		{
			name:      "an empty api version flag clears the configured version",
			file:      fullConfig,
			overrides: Overrides{APIVersion: new("")},
			want: Config{
				URL:        "https://file.example.com/tfs",
				Collection: "AuthCollection",
				PAT:        "file-pat",
				Project:    "FileProject",
			},
		},
		{
			name:      "each source wins for its own field",
			file:      fullConfig,
			env:       map[string]string{"TFSCLI_PROJECT": "EnvProject"},
			overrides: Overrides{Project: new("FlagProject")},
			want: Config{
				URL:              "https://file.example.com/tfs",
				Collection:       "AuthCollection",
				PAT:              "file-pat",
				Project:          "FlagProject",
				APIVersion:       "6.0",
				APIVersionSource: fileSource,
			},
		},
		{
			name: "no config file is needed",
			want: Config{
				URL:        "https://file.example.com/tfs",
				Collection: "AuthCollection",
				PAT:        "file-pat",
			},
		},
		{
			name: "credential fields in the config file and the environment are ignored",
			file: `{"url": "https://attacker.example", "collection": "FileCollection", "pat": "config-pat"}`,
			env:  map[string]string{"TFSCLI_COLLECTION": "EnvCollection"},
			want: Config{
				URL:        "https://file.example.com/tfs",
				Collection: "AuthCollection",
				PAT:        "file-pat",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			for name, value := range tt.env {
				t.Setenv(name, value)
			}

			path := missingFile(t, "config.json")
			if tt.file != "" {
				path = writeConfig(t, tt.file)
			}

			got, err := Load(path, writeAuth(t, fileAuth), tt.overrides)
			if err != nil {
				t.Fatalf("Load() error = %v, want nil", err)
			}
			if tt.want.APIVersionSource == fileSource {
				tt.want.APIVersionSource = `"apiVersion" in ` + path
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

	got, err := Load(path, writeAuth(t, fileAuth), Overrides{})
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if got.Project != "FileProject" {
		t.Errorf("Project = %q, want the config file value", got.Project)
	}
}

func TestLoadReadsTLSFieldsFromFile(t *testing.T) {
	clearEnv(t)
	path := writeConfig(t, `{
	  "insecureSkipVerify": true,
	  "caBundle": "C:/certs/corp.pem"
	}`)

	got, err := Load(path, writeAuth(t, fileAuth), Overrides{})
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

func TestLoadMalformedJSON(t *testing.T) {
	clearEnv(t)
	path := writeConfig(t, `{"project": "FileProject",`)

	_, err := Load(path, writeAuth(t, fileAuth), Overrides{})
	assertConfigError(t, err, "config file at "+path+" is not valid JSON")
}

func TestLoadReportsCredentialErrorsFirst(t *testing.T) {
	clearEnv(t)
	authPath := missingFile(t, "auth.json")

	_, err := Load(missingFile(t, "config.json"), authPath, Overrides{})
	assertConfigError(t, err, "not logged in")
}

func TestLoadFileAbsent(t *testing.T) {
	got, err := LoadFile(missingFile(t, "config.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v, want nil", err)
	}
	if want := (Config{}); *got != want {
		t.Errorf("LoadFile() = %+v, want %+v", *got, want)
	}
}

func TestLoadFileUnreadable(t *testing.T) {
	// A directory in place of the file cannot be read as one.
	path := t.TempDir()

	_, err := LoadFile(path)
	assertConfigError(t, err, "cannot read config file at "+path)
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
