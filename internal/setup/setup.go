// Package setup sets up what the commands share: flags from the
// environment, logging, error reporting, tracing and the elevation model.
package setup

import (
	"cmp"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/catzkorn/trail-tools/internal/blob"
	"github.com/catzkorn/trail-tools/internal/elevation"
	"github.com/catzkorn/trail-tools/internal/errreport"
	"github.com/catzkorn/trail-tools/internal/telemetry"
	"gitlab.com/greyxor/slogor"
	"google.golang.org/api/option"
)

// Reporter returns a reporter sending errors to Error Reporting in a
// project, from the Cloud Run service or job it runs in, or logging them if
// the project is empty.
func Reporter(ctx context.Context, log *slog.Logger, project string) (*errreport.Reporter, error) {
	if project == "" {
		return errreport.NewLocal(log), nil
	}
	return errreport.New(ctx, project, ServiceName(), ServiceVersion(), log,
		option.WithGRPCDialOption(telemetry.GRPCDialOption()))
}

// ServiceName returns the name of the Cloud Run service or job the app runs
// as, or "trail-tools" outside of Cloud Run.
func ServiceName() string {
	return cmp.Or(os.Getenv("K_SERVICE"), os.Getenv("CLOUD_RUN_JOB"), "trail-tools")
}

// ServiceVersion returns the Cloud Run revision or job execution the app
// runs as, if any.
func ServiceVersion() string {
	return cmp.Or(os.Getenv("K_REVISION"), os.Getenv("CLOUD_RUN_EXECUTION"))
}

// tracingShutdownTimeout bounds sending the remaining traces on exit.
const tracingShutdownTimeout = 5 * time.Second

// Tracing exports traces to an OTLP endpoint, and traces requests made
// with the default HTTP client. Outside of Google Cloud, such requests are
// logged too, so their callers can be given a traced client. The returned
// function sends the remaining traces.
func Tracing(ctx context.Context, log *slog.Logger, endpoint, project string, sampleRate float64) (func(), error) {
	shutdown, err := telemetry.Init(ctx, &telemetry.Config{
		Endpoint:       endpoint,
		GCPProject:     project,
		ServiceName:    ServiceName(),
		ServiceVersion: ServiceVersion(),
		SampleRate:     sampleRate,
	})
	if err != nil {
		return nil, err
	}
	if project == "" {
		telemetry.TraceDefaultTransport(log)
	} else {
		telemetry.TraceDefaultTransport(nil)
	}
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), tracingShutdownTimeout)
		defer cancel()
		if err := shutdown(ctx); err != nil {
			log.Warn("Failed to send the remaining traces", slog.String("error", err.Error()))
		}
	}, nil
}

// envPrefix is the prefix of environment variables that set flags.
const envPrefix = "TRAIL_TOOLS_"

// FlagsFromEnv sets flags that weren't set on the command line from
// environment variables named TRAIL_TOOLS_<FLAG>, e.g. TRAIL_TOOLS_DATABASE_URL
// for -database-url. This allows passing secrets through the environment.
func FlagsFromEnv(fs *flag.FlagSet) error {
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	var err error
	fs.VisitAll(func(f *flag.Flag) {
		if set[f.Name] || err != nil {
			return
		}
		name := envPrefix + strings.ToUpper(strings.ReplaceAll(f.Name, "-", "_"))
		if value, ok := os.LookupEnv(name); ok {
			if setErr := f.Value.Set(value); setErr != nil {
				err = fmt.Errorf("invalid value for %s: %w", name, setErr)
			}
		}
	})
	return err
}

// DEM creates a DEM that downloads tiles from baseURL into cacheDir, and
// caches them in the Cloud Storage bucket, if set. On Cloud Run, the cache
// directory is in memory and lost when an instance or job stops.
func DEM(ctx context.Context, log *slog.Logger, baseURL, cacheDir, bucket string) (*elevation.DEM, error) {
	dem := elevation.NewDEM(baseURL, cacheDir, telemetry.NewClient(0))
	if bucket != "" {
		tiles, err := blob.NewGCS(ctx, bucket)
		if err != nil {
			return nil, fmt.Errorf("failed to create the elevation tile bucket client: %w", err)
		}
		dem.SetTileStore(tiles, log)
		log.InfoContext(ctx, "Caching elevation tiles in Cloud Storage", slog.String("bucket", bucket))
	}
	return dem, nil
}

// Logger returns a logger logging at level in format: "text" for
// terminals, or "json" for Cloud Logging.
func Logger(level, format string) (*slog.Logger, error) {
	var slogLevel slog.Level
	if err := slogLevel.UnmarshalText([]byte(level)); err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}
	switch format {
	case "text":
		return slog.New(slogor.NewHandler(os.Stdout, slogor.SetTimeFormat(time.Stamp), slogor.SetLevel(slogLevel))), nil
	case "json":
		return slog.New(jsonLogHandler(slogLevel)), nil
	default:
		return nil, fmt.Errorf("invalid log format %q, expected text or json", format)
	}
}

// jsonLogHandler returns a handler logging JSON lines in the format of
// Google Cloud Logging's structured logs.
func jsonLogHandler(level slog.Level) slog.Handler {
	return slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if len(groups) > 0 {
				return a
			}
			switch a.Key {
			case slog.MessageKey:
				a.Key = "message"
			case slog.LevelKey:
				a.Key = "severity"
				if lvl, ok := a.Value.Any().(slog.Level); ok && lvl == slog.LevelWarn {
					a.Value = slog.StringValue("WARNING")
				}
			}
			return a
		},
	})
}
