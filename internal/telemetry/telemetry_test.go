package telemetry

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/johanbrandhorst/trail-tools/internal/errreport"
	"go.opentelemetry.io/otel/attribute"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestServerSpanName(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		method, path, pattern, want string
	}{
		{"POST", "/trips.v1.TripReportService/GetTripReport", "/trips.v1.TripReportService/", "/trips.v1.TripReportService/GetTripReport"},
		{"GET", "/routes/abc.fit", "/routes/", "GET /routes/"},
		{"GET", "/photos/abc/large.jpg", "GET /photos/", "GET /photos/"},
		{"GET", "/u/jane", "", "GET"},
	} {
		r := httptest.NewRequest(tc.method, tc.path, nil)
		r.Pattern = tc.pattern
		if got := serverSpanName("", r); got != tc.want {
			t.Errorf("serverSpanName(%s %s, %q) = %q, want %q", tc.method, tc.path, tc.pattern, got, tc.want)
		}
	}
}

func TestIsProcedure(t *testing.T) {
	t.Parallel()
	for path, want := range map[string]bool{
		"/trips.v1.TripReportService/GetTripReport": true,
		"/users.v1.UserService/GetCurrentUser":      true,
		"/.well-known/oauth-authorization-server":   false,
		"/trips.v1.TripReportService/":              false,
		"/trips.v1.TripReportService/Get/Extra":     false,
		"/routes/abc.fit":                           false,
		"/index.js":                                 false,
		"/":                                         false,
	} {
		if got := isProcedure(path); got != want {
			t.Errorf("isProcedure(%q) = %v, want %v", path, got, want)
		}
	}
}

// TestTraceDefaultTransport checks that requests with the default client
// are traced and reported.
func TestTraceDefaultTransport(t *testing.T) {
	// Replaces the default transport, so not parallel.
	prevTransport, prevClient := http.DefaultTransport, http.DefaultClient
	t.Cleanup(func() {
		http.DefaultTransport, http.DefaultClient = prevTransport, prevClient
	})
	var buf bytes.Buffer
	TraceDefaultTransport(errreport.NewLocal(slog.New(slog.NewTextHandler(&buf, nil))))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	t.Cleanup(srv.Close)
	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if got := buf.String(); !strings.Contains(got, `msg="A request used the default HTTP client"`) {
		t.Errorf("the request wasn't reported: %q", got)
	}
}

func TestIsStaticAsset(t *testing.T) {
	t.Parallel()
	for path, want := range map[string]bool{
		"/index.js":             true,
		"/fonts/lexend.woff2":   true,
		"/img/og.png":           true,
		"/":                     false,
		"/routes/abc.gpx":       false,
		"/index.json":           false,
		"/users.v1.UserService": false,
	} {
		if got := isStaticAsset(path); got != want {
			t.Errorf("isStaticAsset(%q) = %v, want %v", path, got, want)
		}
	}
}

// TestTracing checks that a request with a traced client to a traced server
// makes a client and a server span in the same trace.
func TestTracing(t *testing.T) {
	// Sets the global trace provider and propagator, so not parallel.
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	prevProvider, prevPropagator := otel.GetTracerProvider(), otel.GetTextMapPropagator()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() {
		otel.SetTracerProvider(prevProvider)
		otel.SetTextMapPropagator(prevPropagator)
	})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /routes/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "ok")
	})
	mux.HandleFunc("GET /index.js", func(w http.ResponseWriter, r *http.Request) {})
	// A middleware passing on a copy of the request, like the live reload.
	copying := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Routes(mux).ServeHTTP(w, r.WithContext(r.Context()))
	})
	srv := httptest.NewServer(Handler(copying))
	t.Cleanup(srv.Close)

	client := NewClient(0)
	for _, path := range []string{"/routes/abc.gpx", "/index.js"} {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
	if err := tp.ForceFlush(context.Background()); err != nil {
		t.Fatal(err)
	}

	var clientSpans, serverSpans []tracetest.SpanStub
	var names []string
	for _, s := range exporter.GetSpans() {
		names = append(names, s.Name)
		switch s.SpanKind {
		case trace.SpanKindClient:
			clientSpans = append(clientSpans, s)
		case trace.SpanKindServer:
			serverSpans = append(serverSpans, s)
		}
	}
	// Static assets aren't traced on the server, only by the client.
	if len(clientSpans) != 2 || len(serverSpans) != 1 {
		t.Fatalf("got spans %v, want 2 client spans and 1 server span", names)
	}
	host := srv.Listener.Addr().String()
	for _, s := range clientSpans {
		if s.Name != "GET "+host {
			t.Errorf("client span %q, want it named after the host", s.Name)
		}
		// The span has the URL, with its path.
		if !slices.ContainsFunc(s.Attributes, func(a attribute.KeyValue) bool {
			return a.Key == "url.full" && strings.HasPrefix(a.Value.AsString(), srv.URL+"/")
		}) {
			t.Errorf("client span %q has no url.full: %v", s.Name, s.Attributes)
		}
	}
	server := serverSpans[0]
	if server.Name != "GET /routes/" {
		t.Errorf("server span %q, want it named after the route", server.Name)
	}
	// The trace context propagates from the client to the server.
	if !slices.ContainsFunc(clientSpans, func(c tracetest.SpanStub) bool {
		return server.Parent.SpanID() == c.SpanContext.SpanID() && server.SpanContext.TraceID() == c.SpanContext.TraceID()
	}) {
		t.Errorf("the server span isn't a child of a client span")
	}
}

func TestInitWithoutEndpoint(t *testing.T) {
	t.Parallel()
	shutdown, err := Init(context.Background(), &Config{})
	if err != nil {
		t.Fatalf("Init without an endpoint: %v", err)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Errorf("shutdown: %v", err)
	}
	if _, err := Init(context.Background(), &Config{Endpoint: LocalEndpoint, SampleRate: 2}); err == nil {
		t.Error("Init accepted a sample rate of 2")
	}
}
