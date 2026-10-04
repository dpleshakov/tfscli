package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadAuthSources(t *testing.T) {
	tests := []struct {
		name string
		env  string
		file string
		want Auth
	}{
		{
			name: "auth file",
			file: `{"url": "https://file.example.com/tfs", "collection": "FileCollection", "pat": "file-pat"}`,
			want: Auth{URL: "https://file.example.com/tfs", Collection: "FileCollection", PAT: "file-pat"},
		},
		{
			name: "TFSCLI_AUTH without an auth file",
			env:  `{"url": "https://env.example.com/tfs", "collection": "EnvCollection", "pat": "env-pat"}`,
			want: Auth{URL: "https://env.example.com/tfs", Collection: "EnvCollection", PAT: "env-pat"},
		},
		{
			name: "TFSCLI_AUTH takes precedence over the auth file",
			env:  `{"url": "https://env.example.com/tfs", "collection": "EnvCollection", "pat": "env-pat"}`,
			file: `{"url": "https://file.example.com/tfs", "collection": "FileCollection", "pat": "file-pat"}`,
			want: Auth{URL: "https://env.example.com/tfs", Collection: "EnvCollection", PAT: "env-pat"},
		},
		{
			name: "TFSCLI_AUTH is not merged with the auth file",
			env:  `{"url": "https://env.example.com/tfs", "collection": "EnvCollection", "pat": "env-pat"}`,
			file: `this file is not even JSON`,
			want: Auth{URL: "https://env.example.com/tfs", Collection: "EnvCollection", PAT: "env-pat"},
		},
		{
			name: "the url is normalised",
			file: `{"url": "HTTPS://TFS.Example.com:8080/tfs/", "collection": "FileCollection", "pat": "file-pat"}`,
			want: Auth{URL: "https://tfs.example.com:8080/tfs", Collection: "FileCollection", PAT: "file-pat"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			t.Setenv(AuthEnv, tt.env)

			path := missingFile(t, "auth.json")
			if tt.file != "" {
				path = writeAuth(t, tt.file)
			}

			got, err := LoadAuth(path)
			if err != nil {
				t.Fatalf("LoadAuth() error = %v, want nil", err)
			}
			if *got != tt.want {
				t.Errorf("LoadAuth() = %+v, want %+v", *got, tt.want)
			}
		})
	}
}

func TestLoadAuthMissing(t *testing.T) {
	clearEnv(t)
	path := missingFile(t, "auth.json")

	_, err := LoadAuth(path)
	assertConfigError(t, err,
		`not logged in: no credential at `+path+` (run "tfscli auth login", or set TFSCLI_AUTH)`)
}

func TestLoadAuthUnreadable(t *testing.T) {
	clearEnv(t)
	// A directory in place of the file cannot be read as one.
	path := t.TempDir()

	_, err := LoadAuth(path)
	assertConfigError(t, err, "cannot read the auth file at "+path)
}

func TestLoadAuthMalformed(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"not JSON", `{"url": "https://tfs.example.com",`, "is not valid JSON"},
		{"no url", `{"collection": "DefaultCollection", "pat": "secret"}`, `has no "url"`},
		{"no collection", `{"url": "https://tfs.example.com", "pat": "secret"}`, `has no "collection"`},
		{"no pat", `{"url": "https://tfs.example.com", "collection": "DefaultCollection"}`, `has no "pat"`},
		{"bad url", `{"url": "tfs.example.com/tfs", "collection": "DefaultCollection", "pat": "secret"}`, "must start with http:// or https://"},
	}

	for _, tt := range tests {
		t.Run("TFSCLI_AUTH "+tt.name, func(t *testing.T) {
			clearEnv(t)
			t.Setenv(AuthEnv, tt.body)

			_, err := LoadAuth(missingFile(t, "auth.json"))
			assertConfigError(t, err, "TFSCLI_AUTH")
			assertConfigError(t, err, tt.want)
		})
		t.Run("auth file "+tt.name, func(t *testing.T) {
			clearEnv(t)
			path := writeAuth(t, tt.body)

			_, err := LoadAuth(path)
			assertConfigError(t, err, "the auth file at "+path)
			assertConfigError(t, err, tt.want)
		})
	}
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"https://tfs.example.com/tfs", "https://tfs.example.com/tfs"},
		{"HTTPS://TFS.EXAMPLE.COM:8080/tfs/", "https://tfs.example.com:8080/tfs"},
		{"http://tfs.example.com/", "http://tfs.example.com"},
		{"  https://tfs.example.com//  ", "https://tfs.example.com"},
		{"https://tfs.example.com/Tfs/DefaultCollection", "https://tfs.example.com/Tfs/DefaultCollection"},
	}
	for _, tt := range tests {
		got, err := NormalizeURL(tt.raw)
		if err != nil {
			t.Errorf("NormalizeURL(%q) error = %v, want nil", tt.raw, err)
			continue
		}
		if got != tt.want {
			t.Errorf("NormalizeURL(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestNormalizeURLRejects(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"http://tfs.example.com:pot/tfs", "is not a valid URL"},
		{"tfs.example.com:8080/tfs", "must start with http:// or https://"},
		{"ftp://tfs.example.com/tfs", "must start with http:// or https://"},
		{"https:///tfs", "has no host"},
	}
	for _, tt := range tests {
		_, err := NormalizeURL(tt.raw)
		assertConfigError(t, err, tt.want)
	}
}

func TestDefaultAuthPath(t *testing.T) {
	xdg := t.TempDir()
	home := t.TempDir()

	tests := []struct {
		name string
		xdg  string
		want string
	}{
		{"XDG_DATA_HOME set", xdg, filepath.Join(xdg, "tfscli", "auth.json")},
		{"XDG_DATA_HOME unset", "", filepath.Join(home, ".local", "share", "tfscli", "auth.json")},
		{"XDG_DATA_HOME relative", filepath.Join("relative", "dir"), filepath.Join(home, ".local", "share", "tfscli", "auth.json")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("XDG_DATA_HOME", tt.xdg)
			// os.UserHomeDir reads HOME on Unix and USERPROFILE on Windows.
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)

			path, err := DefaultAuthPath()
			if err != nil {
				t.Fatalf("DefaultAuthPath() error = %v, want nil", err)
			}
			if path != tt.want {
				t.Errorf("DefaultAuthPath() = %q, want %q", path, tt.want)
			}
		})
	}
}

func TestSaveAuthRoundTrip(t *testing.T) {
	clearEnv(t)
	path := filepath.Join(t.TempDir(), "nested", "tfscli", "auth.json")
	want := Auth{URL: "https://tfs.example.com/tfs", Collection: "DefaultCollection", PAT: "secret"}

	if err := SaveAuth(path, &want); err != nil {
		t.Fatalf("SaveAuth() error = %v, want nil", err)
	}
	// A second save replaces the first.
	if err := SaveAuth(path, &want); err != nil {
		t.Fatalf("second SaveAuth() error = %v, want nil", err)
	}

	got, err := LoadAuth(path)
	if err != nil {
		t.Fatalf("LoadAuth() error = %v, want nil", err)
	}
	if *got != want {
		t.Errorf("LoadAuth() = %+v, want %+v", *got, want)
	}

	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatalf("reading the directory: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("directory holds %d entries, want only auth.json — no temporary file left behind", len(entries))
	}

	if runtime.GOOS == "windows" {
		return
	}
	for p, mode := range map[string]os.FileMode{path: 0o600, filepath.Dir(path): 0o700} {
		info, err := os.Stat(p)
		if err != nil {
			t.Fatalf("stat %s: %v", p, err)
		}
		if got := info.Mode().Perm(); got != mode {
			t.Errorf("mode of %s = %v, want %v", p, got, mode)
		}
	}
}

func TestSaveAuthReportsWriteFailure(t *testing.T) {
	// A file where the directory should be makes the directory impossible.
	blocker := writeFile(t, "tfscli", "not a directory")
	path := filepath.Join(blocker, "auth.json")

	err := SaveAuth(path, &Auth{URL: "https://tfs.example.com", PAT: "secret"})
	assertConfigError(t, err, "cannot write the auth file at "+path)
}
