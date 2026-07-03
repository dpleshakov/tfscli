package tfserr

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestPrintWithHTTPStatus(t *testing.T) {
	tests := []struct {
		category Category
		message  string
		status   int
		want     string
	}{
		{Auth, "PAT is invalid or expired", 401, "Error [auth]: PAT is invalid or expired (HTTP 401)\n"},
		{NotFound, "work item 99999 not found in project MyProject", 404, "Error [not_found]: work item 99999 not found in project MyProject (HTTP 404)\n"},
		{Forbidden, "no access to project MyProject", 403, "Error [forbidden]: no access to project MyProject (HTTP 403)\n"},
		{Server, "TFS returned an internal error", 500, "Error [server]: TFS returned an internal error (HTTP 500)\n"},
		{Config, "config value rejected by server", 400, "Error [config]: config value rejected by server (HTTP 400)\n"},
		{Network, "proxy refused the connection", 502, "Error [network]: proxy refused the connection (HTTP 502)\n"},
	}
	for _, tt := range tests {
		t.Run(string(tt.category), func(t *testing.T) {
			var sb strings.Builder
			Print(&Error{Category: tt.category, Message: tt.message, HTTPStatus: tt.status}, &sb)
			if got := sb.String(); got != tt.want {
				t.Errorf("Print() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPrintWithoutHTTPStatus(t *testing.T) {
	tests := []struct {
		category Category
		message  string
		want     string
	}{
		{Auth, "PAT is not set", "Error [auth]: PAT is not set\n"},
		{NotFound, "work item 99999 not found", "Error [not_found]: work item 99999 not found\n"},
		{Forbidden, "no access to project MyProject", "Error [forbidden]: no access to project MyProject\n"},
		{Server, "TFS returned an invalid response", "Error [server]: TFS returned an invalid response\n"},
		{Config, "config file not found at ~/.tfscli/config.json", "Error [config]: config file not found at ~/.tfscli/config.json\n"},
		{Network, "cannot reach https://tfs.company.com:8080", "Error [network]: cannot reach https://tfs.company.com:8080\n"},
	}
	for _, tt := range tests {
		t.Run(string(tt.category), func(t *testing.T) {
			var sb strings.Builder
			Print(&Error{Category: tt.category, Message: tt.message}, &sb)
			if got := sb.String(); got != tt.want {
				t.Errorf("Print() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPrintWrappedError(t *testing.T) {
	inner := &Error{Category: Network, Message: "cannot reach server"}
	wrapped := fmt.Errorf("running workitem get: %w", inner)

	var sb strings.Builder
	Print(wrapped, &sb)
	if got, want := sb.String(), "Error [network]: cannot reach server\n"; got != want {
		t.Errorf("Print() = %q, want %q", got, want)
	}
}

func TestPrintUntypedError(t *testing.T) {
	var sb strings.Builder
	Print(errors.New("something broke"), &sb)
	if got, want := sb.String(), "Error: something broke\n"; got != want {
		t.Errorf("Print() = %q, want %q", got, want)
	}
}

func TestErrorMessage(t *testing.T) {
	withStatus := &Error{Category: Auth, Message: "PAT is invalid", HTTPStatus: 401}
	if got, want := withStatus.Error(), "[auth] PAT is invalid (HTTP 401)"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
	withoutStatus := &Error{Category: Config, Message: "config file not found"}
	if got, want := withoutStatus.Error(), "[config] config file not found"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestUnwrap(t *testing.T) {
	cause := errors.New("connection refused")
	err := &Error{Category: Network, Message: "cannot reach server", Cause: cause}
	if !errors.Is(err, cause) {
		t.Errorf("errors.Is(err, cause) = false, want true")
	}
}

func TestExitCode(t *testing.T) {
	if got := ExitCode(nil); got != 0 {
		t.Errorf("ExitCode(nil) = %d, want 0", got)
	}
	for _, category := range []Category{Auth, NotFound, Forbidden, Server, Config, Network} {
		err := &Error{Category: category, Message: "msg"}
		if got := ExitCode(err); got != 1 {
			t.Errorf("ExitCode(%s) = %d, want 1", category, got)
		}
	}
	if got := ExitCode(errors.New("untyped")); got != 1 {
		t.Errorf("ExitCode(untyped) = %d, want 1", got)
	}
}
