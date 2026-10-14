package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"runtime"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
)

// baseTransport is the default transport, before TraceDefaultTransport
// replaces it.
var baseTransport = http.DefaultTransport

// NewClient returns an HTTP client whose requests are traced, with a
// timeout, or none if it's 0. Pass it to everything that makes requests.
func NewClient(timeout time.Duration) *http.Client {
	return &http.Client{Transport: Transport(nil), Timeout: timeout}
}

// Transport returns a transport that traces the requests it sends with
// base, or the default transport if base is nil.
func Transport(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = baseTransport
	}
	return otelhttp.NewTransport(base, otelhttp.WithSpanNameFormatter(clientSpanName))
}

// clientSpanName names a request after its method and host, e.g. "GET
// api.open-meteo.com". Paths would make too many span names, e.g. with IDs.
func clientSpanName(_ string, r *http.Request) string {
	if r.URL == nil || r.URL.Host == "" {
		return r.Method
	}
	return r.Method + " " + r.URL.Host
}

// WithClient returns a context with an HTTP client for libraries that take
// theirs from the context, like golang.org/x/oauth2 and the Google API
// clients created with google.DefaultClient.
func WithClient(ctx context.Context, client *http.Client) context.Context {
	return context.WithValue(ctx, oauth2.HTTPClient, client)
}

// GRPCDialOption traces the calls of a gRPC client, like the Google Cloud
// client libraries.
func GRPCDialOption() grpc.DialOption {
	return grpc.WithStatsHandler(otelgrpc.NewClientHandler())
}

// TraceDefaultTransport traces the requests sent with http.DefaultClient
// and http.DefaultTransport, by code that wasn't given a client. If log is
// set, it also logs where such requests come from, so they can be given
// one: use this while developing.
func TraceDefaultTransport(log *slog.Logger) {
	traced := Transport(nil)
	if log != nil {
		traced = reportingTransport{traced: traced, log: log}
	}
	http.DefaultTransport = traced
	http.DefaultClient = &http.Client{Transport: traced}
}

// reportingTransport logs the caller of each request.
type reportingTransport struct {
	traced http.RoundTripper
	log    *slog.Logger
}

func (t reportingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	t.log.WarnContext(r.Context(), "A request used the default HTTP client; give the caller a client from telemetry.NewClient",
		slog.String("url", r.URL.Redacted()), slog.String("caller", caller()))
	return t.traced.RoundTrip(r)
}

// caller returns the first function outside net/http and this package on
// the stack.
func caller() string {
	pcs := make([]uintptr, 32)
	frames := runtime.CallersFrames(pcs[:runtime.Callers(3, pcs)])
	for {
		frame, more := frames.Next()
		if !strings.HasPrefix(frame.Function, "net/http.") && !strings.Contains(frame.Function, "/internal/telemetry.") {
			return fmt.Sprintf("%s (%s:%d)", frame.Function, frame.File, frame.Line)
		}
		if !more {
			return "unknown"
		}
	}
}

// Handler traces the requests to an HTTP server, except for static
// assets, which would only add noise.
func Handler(h http.Handler) http.Handler {
	return otelhttp.NewHandler(h, "",
		otelhttp.WithSpanNameFormatter(serverSpanName),
		otelhttp.WithFilter(func(r *http.Request) bool { return !isStaticAsset(r.URL.Path) }),
	)
}

// Routes names the server spans of the requests a mux serves after the
// route they matched, e.g. "GET /routes/". Put it right around the mux:
// handlers in between may pass on a copy of the request, whose route the
// handler from Handler doesn't see.
func Routes(mux http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r)
		if r.Pattern == "" {
			return
		}
		span := trace.SpanFromContext(r.Context())
		span.SetName(serverSpanName("", r))
		_, route, ok := strings.Cut(r.Pattern, " ")
		if !ok {
			route = r.Pattern
		}
		span.SetAttributes(semconv.HTTPRoute(route))
	})
}

// connectProcedure matches the paths of Connect RPCs, e.g.
// /trips.v1.TripReportService/GetTripReport.
var connectProcedure = regexp.MustCompile(`^/[a-z0-9_]+(\.[a-z0-9_]+)*\.v[0-9]+\.[A-Za-z0-9_]+/[A-Za-z0-9_]+$`)

// serverSpanName names a request after its RPC, or its method and the
// route it matched once that's known, e.g. "GET /routes/".
func serverSpanName(_ string, r *http.Request) string {
	if connectProcedure.MatchString(r.URL.Path) {
		return r.URL.Path
	}
	if r.Pattern != "" {
		// Patterns can include the method, e.g. "GET /photos/".
		if _, pattern, ok := strings.Cut(r.Pattern, " "); ok {
			return r.Method + " " + pattern
		}
		return r.Method + " " + r.Pattern
	}
	return r.Method
}

// staticAssets are the paths of static assets, or their prefixes ending in
// a slash.
var staticAssets = []string{"/index.js", "/index.js.map", "/index.css", "/favicon.svg", "/fonts/", "/img/"}

func isStaticAsset(path string) bool {
	for _, asset := range staticAssets {
		if path == asset || (strings.HasSuffix(asset, "/") && strings.HasPrefix(path, asset)) {
			return true
		}
	}
	return false
}
