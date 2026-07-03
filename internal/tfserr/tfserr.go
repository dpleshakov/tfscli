package tfserr

import (
	"errors"
	"fmt"
	"io"
)

// Category is a machine-readable error category. The set of categories is a
// stable contract: existing values are never removed or renamed; new ones may
// be added.
type Category string

const (
	Auth      Category = "auth"
	NotFound  Category = "not_found"
	Forbidden Category = "forbidden"
	Server    Category = "server"
	Config    Category = "config"
	Network   Category = "network"
)

// Error is the typed error carried between modules and printed to stderr.
type Error struct {
	Category   Category
	Message    string
	HTTPStatus int // 0 when the error did not originate from an HTTP response
	Cause      error
}

func (e *Error) Error() string {
	if e.HTTPStatus != 0 {
		return fmt.Sprintf("[%s] %s (HTTP %d)", e.Category, e.Message, e.HTTPStatus)
	}
	return fmt.Sprintf("[%s] %s", e.Category, e.Message)
}

func (e *Error) Unwrap() error { return e.Cause }

// Print writes err to w in the contract format
// "Error [category]: message (HTTP status)", omitting the status part when
// the error carries no HTTP status. Errors that are not (and do not wrap) an
// *Error are printed as "Error: message".
func Print(err error, w io.Writer) {
	var te *Error
	if errors.As(err, &te) {
		if te.HTTPStatus != 0 {
			fmt.Fprintf(w, "Error [%s]: %s (HTTP %d)\n", te.Category, te.Message, te.HTTPStatus)
		} else {
			fmt.Fprintf(w, "Error [%s]: %s\n", te.Category, te.Message)
		}
		return
	}
	fmt.Fprintf(w, "Error: %s\n", err)
}

// ExitCode maps err to a process exit code: 0 for nil, non-zero otherwise
// (v1: 1 for every category).
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	return 1
}
