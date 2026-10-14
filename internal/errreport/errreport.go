// Package errreport reports errors to Google Cloud Error Reporting, which
// groups and counts them and can notify about new ones. Errors are reported
// instead of logged: the logs say what happened, and Error Reporting what
// went wrong.
package errreport

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"slices"
	"strings"

	"cloud.google.com/go/errorreporting"
	"google.golang.org/api/option"
)

// Reporter reports errors.
type Reporter struct {
	// Nil for a local reporter.
	client *errorreporting.Client
	// Logs errors when there's no client, and failures to report them.
	log *slog.Logger
	// Added to every report, like slog.Logger.With.
	attrs []slog.Attr
}

// New returns a Reporter that reports errors to Error Reporting in a
// project, from a service at a version, e.g. a Cloud Run service and
// revision. Close it to send the errors not yet sent. The options configure
// the client, e.g. to trace its calls.
func New(ctx context.Context, project, service, version string, log *slog.Logger, opts ...option.ClientOption) (*Reporter, error) {
	client, err := errorreporting.NewClient(ctx, project, errorreporting.Config{
		ServiceName:    service,
		ServiceVersion: version,
		OnError: func(err error) {
			log.WarnContext(ctx, "Failed to report an error to Error Reporting", slog.String("error", err.Error()))
		},
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Error Reporting client: %w", err)
	}
	return &Reporter{client: client, log: log}, nil
}

// NewLocal returns a Reporter that logs errors, for local development and
// tests.
func NewLocal(log *slog.Logger) *Reporter {
	return &Reporter{log: log}
}

// With returns a Reporter that adds attributes to every report, e.g. the ID
// of the user a background job works for.
func (r *Reporter) With(attrs ...slog.Attr) *Reporter {
	return &Reporter{client: r.client, log: r.log, attrs: append(slices.Clip(r.attrs), attrs...)}
}

// Report reports an error described by msg, e.g. "Failed to sync
// activities", with attributes for context, like the ID of the user. err may
// be nil when msg says it all.
func (r *Reporter) Report(ctx context.Context, msg string, err error, attrs ...slog.Attr) {
	attrs = append(slices.Clip(r.attrs), attrs...)
	if r.client == nil {
		if err != nil {
			attrs = append(attrs, slog.String("error", err.Error()))
		}
		r.log.LogAttrs(ctx, slog.LevelError, msg, attrs...)
		return
	}
	r.client.Report(errorreporting.Entry{
		Error: describe(msg, err, attrs),
		// The stack of the caller, so errors are grouped by where they
		// were reported, not by this function.
		Stack: callerStack(2),
	})
}

// Close sends the errors not yet sent.
func (r *Reporter) Close() error {
	if r.client == nil {
		return nil
	}
	return r.client.Close()
}

// describe returns an error with msg, the error's message and the
// attributes, e.g. "Failed to sync activities: rate limited (user_id=1)".
func describe(msg string, err error, attrs []slog.Attr) error {
	var b strings.Builder
	b.WriteString(msg)
	if err != nil {
		b.WriteString(": ")
		b.WriteString(err.Error())
	}
	for i, a := range attrs {
		if i == 0 {
			b.WriteString(" (")
		} else {
			b.WriteString(", ")
		}
		b.WriteString(a.String())
		if i == len(attrs)-1 {
			b.WriteString(")")
		}
	}
	if err == nil {
		return errors.New(b.String())
	}
	return reported{msg: b.String(), err: err}
}

// reported is an error with a description, which wraps the original error.
type reported struct {
	msg string
	err error
}

func (e reported) Error() string { return e.msg }
func (e reported) Unwrap() error { return e.err }

// callerStack returns the stack of the caller skip frames up from the
// caller of callerStack, in the format of runtime/debug.Stack, which Error
// Reporting parses.
func callerStack(skip int) []byte {
	pcs := make([]uintptr, 64)
	n := runtime.Callers(skip+2, pcs)
	frames := runtime.CallersFrames(pcs[:n])
	var b strings.Builder
	b.WriteString("goroutine 1 [running]:\n")
	for {
		frame, more := frames.Next()
		fmt.Fprintf(&b, "%s(...)\n\t%s:%d +0x%x\n", frame.Function, frame.File, frame.Line, frame.PC-frame.Entry)
		if !more {
			break
		}
	}
	return []byte(b.String())
}
