package telemetry

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/catzkorn/trail-tools/internal/errreport"
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
// api.open-meteo.com". Many paths have IDs or names, like Strava's
// activities, elevation tiles and stored objects, which would make a span
// name for each. The span's url.full attribute has the whole URL.
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

// errDefaultClient is reported for requests sent with the default client.
var errDefaultClient = errors.New("give the caller a client from telemetry.NewClient")

// TraceDefaultTransport traces the requests sent with http.DefaultClient
// and http.DefaultTransport, by code that wasn't given a client, and reports
// them, so the code can be given one. The report's stack trace shows where
// the request came from.
func TraceDefaultTransport(report *errreport.Reporter) {
	traced := reportingTransport{traced: Transport(nil), report: report}
	http.DefaultTransport = traced
	http.DefaultClient = &http.Client{Transport: traced}
}

// reportingTransport reports each request.
type reportingTransport struct {
	traced http.RoundTripper
	report *errreport.Reporter
}

func (t reportingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	t.report.Report(r.Context(), "A request used the default HTTP client", errDefaultClient, slog.String("host", r.URL.Host))
	return t.traced.RoundTrip(r)
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

// isProcedure reports whether a path is a Connect RPC's: a service's full
// name and a method, e.g. /trips.v1.TripReportService/GetTripReport. Method
// names start with a capital letter, unlike the paths of /.well-known/.
func isProcedure(path string) bool {
	service, method, ok := strings.Cut(strings.TrimPrefix(path, "/"), "/")
	return ok && strings.Contains(service, ".") && !strings.HasPrefix(service, ".") &&
		method != "" && !strings.Contains(method, "/") && unicode.IsUpper(rune(method[0]))
}

// serverSpanName names a request after its RPC, or its method and the
// route it matched once that's known, e.g. "GET /routes/".
func serverSpanName(_ string, r *http.Request) string {
	if isProcedure(r.URL.Path) {
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
