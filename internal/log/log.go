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
}

// Noop returns a Logger that discards everything. It is the default when
// verbose output is not requested.
func Noop() Logger { return noop{} }

type noop struct{}

func (noop) LogRequest(method, url string, status int, dur time.Duration) {}

func (noop) Warn(msg string) {}

// New returns a Logger writing one line per event to w. The CLI passes
// os.Stderr when --verbose / TFSCLI_VERBOSE=1 is set.
func New(w io.Writer) Logger { return &writerLogger{w: w} }

type writerLogger struct {
	w io.Writer
}

func (l *writerLogger) LogRequest(method, url string, status int, dur time.Duration) {
	fmt.Fprintf(l.w, "%s %s %d %s\n", method, url, status, dur)
}

func (l *writerLogger) Warn(msg string) {
	fmt.Fprintf(l.w, "[warn] %s\n", msg)
}
