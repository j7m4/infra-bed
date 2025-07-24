package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	otelpyroscope "github.com/grafana/otel-profiling-go"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"

	"github.com/go-infra-spikes/go-spikes/pkg/fibonacci"
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

type Response struct {
	Message    string      `json:"message"`
	Duration   string      `json:"duration,omitempty"`
	Result     interface{} `json:"result,omitempty"`
	Iterations int         `json:"iterations,omitempty"`
}

func main() {
	ctx := context.Background()

	// Initialize tracing
	tp, err := initTracer(ctx)
	if err != nil {
		log.Printf("Failed to initialize tracer: %v", err)
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
		log.Printf("Starting pprof server on port %s", pprofPort)
		log.Println(http.ListenAndServe(":"+pprofPort, nil))
	}()

	r := mux.NewRouter()
	r.Use(otelmux.Middleware("go-spikes"))

	// Health check
	r.HandleFunc("/health", healthHandler).Methods("GET")

	////////////////////////////////////////////////////////////
	// SPIKE ENDPOINTS

	r.HandleFunc("/cpu/fibonacci/{n}", fibonacciHandler).Methods("GET")

	////////////////////////////////////////////////////////////

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting go-spikes on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(Response{Message: "healthy"})
}

func fibonacciHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	n, err := strconv.Atoi(vars["n"])
	if err != nil || n < 1 || n > 45 {
		http.Error(w, "Invalid number (1-45)", http.StatusBadRequest)
		return
	}

	start := time.Now()
	result := fibonacci.DoFibonacci(n)
	duration := time.Since(start)

	json.NewEncoder(w).Encode(Response{
		Message:  fmt.Sprintf("Fibonacci of %d", n),
		Duration: duration.String(),
		Result:   result,
	})
}
