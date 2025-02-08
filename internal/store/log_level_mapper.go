package store

import (
	"context"

	"github.com/jackc/pgx/v5/tracelog"
)

// logLevelMapper maps from the default pgx log level to
// a lower priority level. It maps:
// tracelog.LogLevelTrace, tracelog.LogLevelDebug, tracelog.LogLevelWarn
// and tracelog.LogLevelError to themselves, and maps
// tracelog.LogLevelInfo to traceLogLevel.LevelDebug.
type logLevelMapper struct {
	wrapped tracelog.Logger
}

func (l *logLevelMapper) Log(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]any) {
	if level == tracelog.LogLevelInfo {
		level = tracelog.LogLevelDebug
	}
	l.wrapped.Log(ctx, level, msg, data)
}
