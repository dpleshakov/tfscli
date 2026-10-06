package log

import (
	"fmt"
	"io"
	"time"
)

// Logger records diagnostic events. Implementations receive only scalar
// request fields — never headers or bodies — so credentials cannot pass
// through this interface.
type Logger interface {
	LogRequest(method, url string, status int, dur time.Duration)
	Warn(msg string)
	Info(msg string)
}

// Noop returns a Logger that discards everything. It is the default when
// verbose output is not requested.
func Noop() Logger { return noop{} }

type noop struct{}

// LogRequest discards the event.
func (noop) LogRequest(method, url string, status int, dur time.Duration) {}

// Warn discards the message.
func (noop) Warn(msg string) {}

// Info discards the message.
func (noop) Info(msg string) {}

// New returns a Logger writing one line per event to w. The CLI passes
// os.Stderr when --verbose / TFSCLI_VERBOSE=1 is set.
func New(w io.Writer) Logger { return &writerLogger{w: w} }

type writerLogger struct {
	w io.Writer
}

// LogRequest writes one line describing a finished round trip. A write that
// fails is ignored: diagnostics must never displace the command's own result.
func (l *writerLogger) LogRequest(method, url string, status int, dur time.Duration) {
	_, _ = fmt.Fprintf(l.w, "%s %s %d %s\n", method, url, status, dur)
}

// Warn writes one line describing a condition worth reporting but not fatal.
func (l *writerLogger) Warn(msg string) {
	_, _ = fmt.Fprintf(l.w, "[warn] %s\n", msg)
}

// Info writes one line describing what tfscli did on its own, such as a
// fallback, that a reader of the request lines would otherwise have to infer.
func (l *writerLogger) Info(msg string) {
	_, _ = fmt.Fprintf(l.w, "[info] %s\n", msg)
}
