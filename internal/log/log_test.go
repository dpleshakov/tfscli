package log

import (
	"io"
	"strings"
	"testing"
	"time"
)

func TestWriterLoggerLogRequest(t *testing.T) {
	var sb strings.Builder
	l := New(&sb)

	l.LogRequest("GET", "https://tfs.company.com:8080/DefaultCollection/MyProject/_apis/wit/workitems/12345?api-version=7.2", 200, 245*time.Millisecond)

	want := "GET https://tfs.company.com:8080/DefaultCollection/MyProject/_apis/wit/workitems/12345?api-version=7.2 200 245ms\n"
	if got := sb.String(); got != want {
		t.Errorf("LogRequest() wrote %q, want %q", got, want)
	}
}

func TestWriterLoggerLogRequestOneLinePerCall(t *testing.T) {
	var sb strings.Builder
	l := New(&sb)

	l.LogRequest("GET", "https://tfs.company.com/coll/_apis/a", 200, time.Second)
	l.LogRequest("GET", "https://tfs.company.com/coll/_apis/b", 404, 30*time.Millisecond)

	got := sb.String()
	lines := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("LogRequest() x2 wrote %d lines, want 2: %q", len(lines), got)
	}
	if want := "GET https://tfs.company.com/coll/_apis/a 200 1s"; lines[0] != want {
		t.Errorf("first line = %q, want %q", lines[0], want)
	}
	if want := "GET https://tfs.company.com/coll/_apis/b 404 30ms"; lines[1] != want {
		t.Errorf("second line = %q, want %q", lines[1], want)
	}
}

func TestWriterLoggerWarn(t *testing.T) {
	var sb strings.Builder
	l := New(&sb)

	l.Warn("TLS verification disabled")

	if got, want := sb.String(), "[warn] TLS verification disabled\n"; got != want {
		t.Errorf("Warn() wrote %q, want %q", got, want)
	}
}

func TestWriterLoggerInfo(t *testing.T) {
	var sb strings.Builder
	l := New(&sb)

	l.Info("api-version not negotiated")

	if got, want := sb.String(), "[info] api-version not negotiated\n"; got != want {
		t.Errorf("Info() wrote %q, want %q", got, want)
	}
}

var (
	_ Logger = Noop()
	_ Logger = New(io.Discard)
)

func TestNoopDoesNotPanic(t *testing.T) {
	// The noop logger holds no writer, so it writes nothing by construction;
	// there is no output to observe. This test only guards that the calls are
	// safe to make.
	l := Noop()

	l.LogRequest("GET", "https://tfs.company.com/coll/_apis/a", 200, time.Second)
	l.Warn("TLS verification disabled")
	l.Info("api-version not negotiated")
}
