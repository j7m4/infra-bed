package main

import (
	"context"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/go-infra-spikes/go-spikes/cmd/handler"
	"github.com/gorilla/mux"
	otelpyroscope "github.com/grafana/otel-profiling-go"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"

	"github.com/go-infra-spikes/go-spikes/pkg/logger"
)

func initTracer(ctx context.Context) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptrace.New(ctx, otlptracegrpc.NewClient(
		otlptracegrpc.WithEndpoint(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")),
		otlptracegrpc.WithInsecure(),
	))
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("go-spikes"),
			semconv.ServiceVersion("1.0.0"),
		)),
	)

	otel.SetTracerProvider(tp)
	return tp, nil
}

func main() {
	ctx := context.Background()

	// Initialize logger
	logger.Init()
	log := logger.Get()

	// Initialize tracing
	tp, err := initTracer(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to initialize tracer")
	} else {
		defer tp.Shutdown(ctx)
		// Wrap tracer provider for Pyroscope integration
		otel.SetTracerProvider(otelpyroscope.NewTracerProvider(tp))
	}

	// Note: We're using Alloy to scrape pprof endpoints instead of pushing directly
	// This allows for better integration with the Grafana stack

	// Start pprof server on separate port
	pprofPort := os.Getenv("PPROF_PORT")
	if pprofPort == "" {
		pprofPort = "6060"
	}
	go func() {
		log.Info().Str("port", pprofPort).Msg("Starting pprof server")
		if err := http.ListenAndServe(":"+pprofPort, nil); err != nil {
			log.Error().Err(err).Msg("pprof server error")
		}
	}()

	r := mux.NewRouter()
	r.Use(otelmux.Middleware("go-spikes"))

	// Health check
	r.HandleFunc("/health", handler.Health).Methods("GET")

	////////////////////////////////////////////////////////////
	// SPIKE ENDPOINTS

	r.HandleFunc("/cpu/fibonacci/{n}", handler.Fibonacci).Methods("GET")
	r.HandleFunc("/kafka/produce", handler.KafkaProduce).Methods("GET")
	r.HandleFunc("/kafka/consume", handler.KafkaConsume).Methods("GET")

	////////////////////////////////////////////////////////////

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Info().Str("port", port).Msg("Starting go-spikes")
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal().Err(err).Msg("Server failed to start")
	}
}
