// Package log provides the Logger interface with noop (default) and
// stderr (--verbose) implementations. The interface accepts only scalar
// request fields, so the PAT and Authorization header can never leak
// through it.
package log
