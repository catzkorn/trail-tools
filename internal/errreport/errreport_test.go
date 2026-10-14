package errreport

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"regexp"
	"strings"
	"testing"
)

func TestDescribe(t *testing.T) {
	t.Parallel()
	cause := errors.New("rate limited")
	err := describe("Failed to sync activities", cause, []slog.Attr{slog.String("user_id", "u1"), slog.Int("page", 2)})
	if err.Error() != "Failed to sync activities: rate limited (user_id=u1, page=2)" || !errors.Is(err, cause) {
		t.Errorf("got %q, want the message, error and attributes, wrapping the error", err)
	}
	if err := describe("Failed to start", nil, nil); err.Error() != "Failed to start" {
		t.Errorf("got %q without an error or attributes", err)
	}
}

func report() []byte {
	return callerStack(0)
}

func TestCallerStack(t *testing.T) {
	t.Parallel()
	stack := string(report())
	// The format of runtime/debug.Stack, starting at the caller.
	lines := strings.Split(strings.TrimSpace(stack), "\n")
	if lines[0] != "goroutine 1 [running]:" {
		t.Fatalf("stack starts with %q", lines[0])
	}
	if !strings.HasSuffix(lines[1], ".report(...)") {
		t.Errorf("stack's first frame is %q, want the caller", lines[1])
	}
	if !regexp.MustCompile(`^\t/.+/errreport_test\.go:\d+ \+0x[0-9a-f]+$`).MatchString(lines[2]) {
		t.Errorf("unexpected location line %q", lines[2])
	}
	if !strings.HasSuffix(lines[3], ".TestCallerStack(...)") {
		t.Errorf("stack's second frame is %q", lines[3])
	}
}

func TestLocalReporter(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	r := NewLocal(slog.New(slog.NewTextHandler(&buf, nil)))
	r.With(slog.String("user_id", "u1")).Report(context.Background(), "Failed to sync", errors.New("boom"), slog.Int("page", 2))
	if got := buf.String(); !strings.Contains(got, `level=ERROR msg="Failed to sync" user_id=u1 page=2 error=boom`) {
		t.Errorf("logged %q", got)
	}
	if err := r.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}
