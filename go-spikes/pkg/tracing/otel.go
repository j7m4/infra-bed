package tracing

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	tracer     trace.Tracer
	tracerName = "github.com/infra-bed/go-spikes"
)

// InitTracer initializes OpenTelemetry tracing with improved configuration
func InitTracer(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	// Get OTLP endpoint from environment or use default
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "alloy.observability:4317"
	}

	// Create OTLP trace exporter
	exporter, err := otlptrace.New(
		ctx,
		otlptracegrpc.NewClient(
			otlptracegrpc.WithEndpoint(endpoint),
			otlptracegrpc.WithInsecure(),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating OTLP trace exporter: %w", err)
	}

	// Get service version from environment or use default
	version := os.Getenv("SERVICE_VERSION")
	if version == "" {
		version = "1.0.0"
	}

	// Create resource with comprehensive service information
	resource := sdkresource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
		semconv.ServiceVersion(version),
		semconv.DeploymentEnvironment(getEnvironment()),
		// Additional useful attributes
		attribute.String("service.instance.id", getInstanceID()),
		attribute.String("go.version", runtime.Version()),
		attribute.String("go.arch", runtime.GOARCH),
		attribute.String("go.os", runtime.GOOS),
	)

	// Create trace provider with improved configuration
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			// Configure batch processor for better performance
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
			sdktrace.WithMaxQueueSize(2048),
		),
		sdktrace.WithResource(resource),
		sdktrace.WithSampler(createSampler()),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Set global propagator for distributed tracing
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	// Initialize package-level tracer
	tracer = otel.GetTracerProvider().Tracer(
		tracerName,
		trace.WithInstrumentationVersion("1.0.0"),
	)

	// Return shutdown function
	return tp.Shutdown, nil
}

// GetTracer returns a tracer for the given component
func GetTracer(component string) trace.Tracer {
	return otel.GetTracerProvider().Tracer(
		tracerName,
		trace.WithInstrumentationVersion("1.0.0"),
		trace.WithInstrumentationAttributes(
			attribute.String("component", component),
		),
	)
}

// StartSpan starts a new span with the given name and options
func StartSpan(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if tracer == nil {
		// Return a no-op span if tracer is not initialized
		return ctx, trace.SpanFromContext(ctx)
	}
	return tracer.Start(ctx, spanName, opts...)
}

// StartSpanWithAttributes starts a new span with the given name and attributes
func StartSpanWithAttributes(ctx context.Context, spanName string, attrs []attribute.KeyValue, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	opts = append(opts, trace.WithAttributes(attrs...))
	return StartSpan(ctx, spanName, opts...)
}

// RecordError records an error in the current span with additional context
func RecordError(span trace.Span, err error, description string, attrs ...attribute.KeyValue) {
	if span == nil || err == nil {
		return
	}

	// Add common error attributes
	errorAttrs := []attribute.KeyValue{
		attribute.String("error.type", fmt.Sprintf("%T", err)),
		attribute.String("error.message", err.Error()),
	}

	if description != "" {
		errorAttrs = append(errorAttrs, attribute.String("error.description", description))
	}

	// Add any additional attributes provided
	errorAttrs = append(errorAttrs, attrs...)

	span.RecordError(err, trace.WithAttributes(errorAttrs...))
	span.SetStatus(codes.Error, description)
}

// SetSpanAttributes sets multiple attributes on a span
func SetSpanAttributes(span trace.Span, attrs ...attribute.KeyValue) {
	if span == nil {
		return
	}
	span.SetAttributes(attrs...)
}

// AddSpanEvent adds an event to the span with attributes
func AddSpanEvent(span trace.Span, name string, attrs ...attribute.KeyValue) {
	if span == nil {
		return
	}
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// WithSpanKind returns a SpanStartOption that sets the span kind
func WithSpanKind(kind trace.SpanKind) trace.SpanStartOption {
	return trace.WithSpanKind(kind)
}

// Common attribute helpers for consistent span labeling
func HTTPAttributes(method, url, userAgent string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("http.method", method),
		attribute.String("http.url", url),
		attribute.String("http.user_agent", userAgent),
	}
}

func HTTPStatusAttributes(statusCode int) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.Int("http.status_code", statusCode),
	}
}

func KafkaAttributes(topic, partition, operation string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("messaging.system", "kafka"),
		attribute.String("messaging.destination.name", topic),
		attribute.String("messaging.kafka.partition", partition),
		attribute.String("messaging.operation", operation),
	}
}

func DatabaseAttributes(dbSystem, dbName, operation string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("db.system", dbSystem),
		attribute.String("db.name", dbName),
		attribute.String("db.operation", operation),
	}
}

func FibonacciAttributes(n int) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("operation.type", "fibonacci"),
		attribute.Int("fibonacci.input", n),
	}
}

// getEnvironment returns the current environment
func getEnvironment() string {
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = os.Getenv("ENV")
	}
	if env == "" {
		env = "development"
	}
	return env
}

// getInstanceID returns a unique instance identifier
func getInstanceID() string {
	instanceID := os.Getenv("INSTANCE_ID")
	if instanceID == "" {
		instanceID = os.Getenv("HOSTNAME")
	}
	if instanceID == "" {
		instanceID = os.Getenv("POD_NAME")
	}
	if instanceID == "" {
		instanceID = "unknown"
	}
	return instanceID
}

// createSampler creates an appropriate sampler based on environment
func createSampler() sdktrace.Sampler {
	samplingRate := os.Getenv("OTEL_TRACE_SAMPLING_RATE")

	switch samplingRate {
	case "0":
		return sdktrace.NeverSample()
	case "1", "":
		return sdktrace.AlwaysSample()
	default:
		// For production environments, use parent-based sampling with ratio
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(0.1)) // 10% sampling
	}
}
