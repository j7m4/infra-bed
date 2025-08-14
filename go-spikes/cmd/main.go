package main

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	otelpyroscope "github.com/grafana/otel-profiling-go"
	"github.com/infra-bed/go-spikes/cmd/handler"
	"github.com/infra-bed/go-spikes/pkg/config"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"

	"github.com/infra-bed/go-spikes/pkg/logger"
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

	// Initialize configuration
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "/etc/config/config.yaml"
	}

	cfgManager, err := config.NewConfigManager(configPath)
	if err != nil {
		log.Warn().Err(err).Str("path", configPath).Msg("Failed to load config, using defaults")
		// Create config manager with defaults if file doesn't exist
		cfgManager, _ = config.NewConfigManager("/tmp/nonexistent.yaml")
	}

	// Store config manager for use by handlers
	handler.SetConfigManager(cfgManager)

	// Register config change callback
	cfgManager.OnChange(func(cfg *config.Config) {
		log.Info().
			Bool("debug", cfg.Features.EnableDebugLogging).
			Bool("profiling", cfg.Features.EnableProfiling).
			Bool("tracing", cfg.Features.EnableTracing).
			Msg("Configuration updated")

		// Update logger level based on config
		if cfg.Features.EnableDebugLogging {
			logger.SetDebugLevel()
		} else {
			logger.SetInfoLevel()
		}
	})

	// Get initial config
	cfg := cfgManager.Get()

	// Enable debug logging if set
	if cfg.Features.EnableDebugLogging {
		logger.SetDebugLevel()
	}

	// Initialize tracing if enabled
	var tp *sdktrace.TracerProvider
	if cfg.Features.EnableTracing {
		tp, err = initTracer(ctx)
		if err != nil {
			log.Error().Err(err).Msg("Failed to initialize tracer")
		} else {
			defer tp.Shutdown(ctx)
			// Wrap tracer provider for Pyroscope integration
			otel.SetTracerProvider(otelpyroscope.NewTracerProvider(tp))
			
			// Enable OTEL logging when tracing is enabled
			if err := logger.EnableOTEL(ctx); err != nil {
				log.Error().Err(err).Msg("Failed to enable OTEL logging")
			}
		}
	}

	// Note: We're using Alloy to scrape pprof endpoints instead of pushing directly
	// This allows for better integration with the Grafana stack

	// Start pprof server on separate port if profiling is enabled
	if cfg.Features.EnableProfiling {
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
	}

	r := mux.NewRouter()
	r.Use(otelmux.Middleware("go-spikes"))

	// Health check
	r.HandleFunc("/health", handler.Health).Methods("GET")

	////////////////////////////////////////////////////////////
	// SPIKE ENDPOINTS

	r.HandleFunc("/cpu/fibonacci/{n}", handler.Fibonacci).Methods("GET")
	r.HandleFunc("/kafka/entity-repo", handler.EntityRepoTest).Methods("GET")
	r.HandleFunc("/config", handler.GetConfig).Methods("GET")
	r.HandleFunc("/config/feature/{feature}", handler.CheckFeature).Methods("GET")

	////////////////////////////////////////////////////////////

	// Use port from config or environment
	port := fmt.Sprintf("%d", cfg.Server.Port)
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}

	// Create server with config timeouts
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	log.Info().
		Str("port", port).
		Dur("readTimeout", cfg.Server.ReadTimeout).
		Dur("writeTimeout", cfg.Server.WriteTimeout).
		Dur("idleTimeout", cfg.Server.IdleTimeout).
		Msg("Starting go-spikes")

	// Start server in a goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed to start")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Shutdown server
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
	}

	// Shutdown OTEL logging
	if err := logger.ShutdownOTEL(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Failed to shutdown OTEL logging")
	}

	log.Info().Msg("Server exited")
}
