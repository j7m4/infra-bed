package main

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	otelpyroscope "github.com/grafana/otel-profiling-go"
	"github.com/grafana/pyroscope-go"
	"github.com/infra-bed/go-spikes/cmd/handler"
	"github.com/infra-bed/go-spikes/pkg/config"
	"github.com/infra-bed/go-spikes/pkg/logger"
	"github.com/infra-bed/go-spikes/pkg/metrics"
	"github.com/infra-bed/go-spikes/pkg/tracing"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/otel"
)


func initPyroscope() {
	pyroscope.Start(pyroscope.Config{
		ApplicationName: "go-spikes",

		// replace this with the address of pyroscope server
		ServerAddress: "http://pyroscope.observability:4040",

		// you can disable logging by setting this to nil
		Logger: nil, //pyroscope.StandardLogger,

		// Uncomment the following line to enable subset of options; defaults to all profiles
		//ProfileTypes: []pyroscope.ProfileType{
		//	pyroscope.ProfileCPU,
		//	pyroscope.ProfileAllocObjects,
		//	pyroscope.ProfileAllocSpace,
		//	pyroscope.ProfileInuseObjects,
		//	pyroscope.ProfileInuseSpace,
		//},
	})
}

func main() {
	initPyroscope()

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
		// Create empty config file and retry
		os.WriteFile("/tmp/default.yaml", []byte{}, 0644)
		cfgManager, err = config.NewConfigManager("/tmp/default.yaml")
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create default config manager")
		}
	}

	// Store config manager for use by handlers
	handler.SetConfigManager(cfgManager)

	// Register config change callback
	cfgManager.OnChange(func(cfg *config.Config) {
		log.Info().
			Str("logLevel", cfg.Features.LogLevel).
			Bool("profiling", cfg.Features.EnableProfiling).
			Bool("tracing", cfg.Features.EnableTracing).
			Msg("Configuration updated")

		logger.SetLogLevel(cfg.Features.LogLevel)
	})

	// Get initial config
	cfg := cfgManager.Get()

	logger.SetLogLevel(cfg.Features.LogLevel)

	// CROSS-CUTTING START OF otel-metrics CONFIGURATION FOR go-spikes
	// Initialize metrics system
	metrics.RecordApplicationInfo("1.0.0", runtime.Version())
	startTime := time.Now()
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				metrics.UpdateApplicationUptime(time.Since(startTime).Seconds())
			case <-ctx.Done():
				return
			}
		}
	}()
	// CROSS-CUTTING END OF otel-metrics CONFIGURATION FOR go-spikes

	// CROSS-CUTTING START OF otel-tracing CONFIGURATION FOR go-spikes
	// Initialize tracing if enabled
	var shutdownTracer func(context.Context) error
	if cfg.Features.EnableTracing {
		shutdownTracer, err = tracing.InitTracer(ctx, "go-spikes")
		if err != nil {
			log.Error().Err(err).Msg("Failed to initialize tracer")
		} else {
			defer shutdownTracer(ctx)
			// CROSS-CUTTING START OF pyroscope CONFIGURATION FOR go-spikes
			// Wrap tracer provider for Pyroscope integration
			otel.SetTracerProvider(otelpyroscope.NewTracerProvider(otel.GetTracerProvider()))
			// CROSS-CUTTING END OF pyroscope CONFIGURATION FOR go-spikes

			// CROSS-CUTTING START OF otel-logging CONFIGURATION FOR go-spikes
			// Enable OTEL logging when tracing is enabled
			if err := logger.EnableOTEL(ctx); err != nil {
				log.Error().Err(err).Msg("Failed to enable OTEL logging")
			}
			// CROSS-CUTTING END OF otel-logging CONFIGURATION FOR go-spikes
		}
	}
	// CROSS-CUTTING END OF otel-tracing CONFIGURATION FOR go-spikes

	// CROSS-CUTTING START OF pyroscope CONFIGURATION FOR go-spikes
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
	// CROSS-CUTTING END OF pyroscope CONFIGURATION FOR go-spikes

	r := mux.NewRouter()
	r.Use(otelmux.Middleware("go-spikes"))
	r.Use(handler.HTTPMetricsMiddleware)

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

	// CROSS-CUTTING START OF otel-metrics CONFIGURATION FOR go-spikes
	// Create and start metrics server on port 8080
	metricsRouter := mux.NewRouter()
	metricsRouter.Handle("/metrics", promhttp.Handler()).Methods("GET")
	metricsSrv := &http.Server{
		Addr:    ":8080",
		Handler: metricsRouter,
	}

	go func() {
		log.Info().Str("port", "8080").Msg("Starting metrics server")
		if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("Metrics server failed to start")
		}
	}()
	// CROSS-CUTTING END OF otel-metrics CONFIGURATION FOR go-spikes

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

	// Shutdown servers
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
	}
	
	// CROSS-CUTTING START OF otel-metrics CONFIGURATION FOR go-spikes
	if err := metricsSrv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Metrics server forced to shutdown")
	}
	// CROSS-CUTTING END OF otel-metrics CONFIGURATION FOR go-spikes

	// Shutdown OTEL logging
	if err := logger.ShutdownOTEL(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Failed to shutdown OTEL logging")
	}

	log.Info().Msg("Server exited")
}
