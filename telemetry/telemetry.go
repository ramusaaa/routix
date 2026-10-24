package telemetry

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/ramusaaa/routix"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type Config struct {
	Registry       *prometheus.Registry
	TracerProvider trace.TracerProvider
	Propagator     propagation.TextMapPropagator
}

type Telemetry struct {
	Registry   *prometheus.Registry
	tracer     trace.Tracer
	propagator propagation.TextMapPropagator
	requests   *prometheus.CounterVec
	duration   *prometheus.HistogramVec
	active     prometheus.Gauge
}

func New(config Config) (*Telemetry, error) {
	if config.Registry == nil {
		config.Registry = prometheus.NewRegistry()
	}
	if config.TracerProvider == nil {
		config.TracerProvider = otel.GetTracerProvider()
	}
	if config.Propagator == nil {
		config.Propagator = propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	}
	labels := []string{"method", "route", "status"}
	t := &Telemetry{Registry: config.Registry, tracer: config.TracerProvider.Tracer("github.com/ramusaaa/routix"), propagator: config.Propagator,
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "http_server_requests_total", Help: "Completed HTTP requests."}, labels),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "http_server_request_duration_seconds", Help: "HTTP request duration in seconds.", Buckets: prometheus.DefBuckets}, labels),
		active:   prometheus.NewGauge(prometheus.GaugeOpts{Name: "http_server_active_requests", Help: "Active HTTP requests."})}
	registered := []prometheus.Collector{}
	for _, collector := range []prometheus.Collector{t.requests, t.duration, t.active} {
		if err := t.Registry.Register(collector); err != nil {
			for _, previous := range registered {
				t.Registry.Unregister(previous)
			}
			return nil, fmt.Errorf("register HTTP metrics: %w", err)
		}
		registered = append(registered, collector)
	}
	return t, nil
}

func (t *Telemetry) Handler() http.Handler {
	return promhttp.HandlerFor(t.Registry, promhttp.HandlerOpts{EnableOpenMetrics: true})
}

func (t *Telemetry) Attach(router *routix.Router, path string) {
	if path == "" {
		path = "/metrics"
	}
	router.Use(t.Middleware())
	handler := t.Handler()
	router.GET(path, func(c *routix.Context) error { handler.ServeHTTP(c.Response, c.Request); return nil })
}

func (t *Telemetry) Middleware() routix.Middleware {
	return func(next routix.Handler) routix.Handler {
		return func(c *routix.Context) (err error) {
			start := time.Now()
			route := c.RoutePath()
			if route == "" {
				route = "unmatched"
			}
			method := c.Request.Method
			switch method {
			case "GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "CONNECT", "TRACE":
			default:
				method = "_OTHER"
			}
			parent := t.propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))
			ctx, span := t.tracer.Start(parent, method+" "+route, trace.WithSpanKind(trace.SpanKindServer), trace.WithAttributes(attribute.String("http.request.method", method), attribute.String("http.route", route)))
			request := c.Request
			c.Request = c.Request.WithContext(ctx)
			t.active.Inc()
			defer func() {
				recovered := recover()
				status := c.Status()
				if recovered != nil {
					status = http.StatusInternalServerError
					span.RecordError(fmt.Errorf("handler panic"))
				}
				if err != nil {
					span.RecordError(err)
					if status < 400 {
						status = http.StatusInternalServerError
						var responseError *routix.Error
						if errors.As(err, &responseError) {
							status = responseError.Code
						}
					}
				}
				if status >= 500 {
					span.SetStatus(codes.Error, http.StatusText(status))
				}
				span.SetAttributes(attribute.Int("http.response.status_code", status))
				span.End()
				t.requests.WithLabelValues(method, route, strconv.Itoa(status)).Inc()
				t.duration.WithLabelValues(method, route, strconv.Itoa(status)).Observe(time.Since(start).Seconds())
				t.active.Dec()
				c.Request = request
				if recovered != nil {
					panic(recovered)
				}
			}()
			return next(c)
		}
	}
}
