package logger

import (
	"context"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
)

var Logger zerolog.Logger
var Level zerolog.Level = zerolog.InfoLevel
var lokiWriter *LokiWriter

func Init() {
	zerolog.TimeFieldFormat = time.RFC3339

	defaultLevel := zerolog.InfoLevel

	// Create console writer
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}

	var writers []io.Writer
	writers = append(writers, consoleWriter)

	// Create multi-writer output
	multi := zerolog.MultiLevelWriter(writers...)

	Logger = zerolog.New(multi).
		Level(Level).
		With().
		Timestamp().
		Caller().
		Logger()

	log.Logger = Logger

	Logger.Info().Str("level", defaultLevel.String()).Msg("Logger initialized")
}

func Get() *zerolog.Logger {
	return &Logger
}

// WithContext returns a logger with trace context if available
func WithContext(ctx context.Context) zerolog.Logger {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		return Logger.Level(Level).With().
			Str("trace_id", spanCtx.TraceID().String()).
			Str("span_id", spanCtx.SpanID().String()).
			Logger()
	}
	return Logger.Level(Level)
}

// Ctx returns a zerolog context logger with trace information
func Ctx(ctx context.Context) *zerolog.Logger {
	l := WithContext(ctx)
	return &l
}

func SetLogLevel(level string) {
	switch strings.ToLower(level) {
	case "trace":
		setLevel(zerolog.TraceLevel)
	case "debug":
		setLevel(zerolog.DebugLevel)
	case "info":
		setLevel(zerolog.InfoLevel)
	case "warn":
		setLevel(zerolog.WarnLevel)
	case "error":
		setLevel(zerolog.ErrorLevel)
	case "fatal":
		setLevel(zerolog.FatalLevel)
	case "panic":
		setLevel(zerolog.PanicLevel)
	default:
		Logger.Warn().Str("level", level).Msg("Unknown log level, defaulting to INFO")
		setLevel(zerolog.InfoLevel)
	}
}

func setLevel(level zerolog.Level) {
	Level = level
	zerolog.SetGlobalLevel(level)
}

// EnableOTEL enables Loki logging for Grafana integration
func EnableOTEL(ctx context.Context) error {
	if lokiWriter != nil {
		return nil
	}

	// Configure Loki endpoint
	lokiURL := os.Getenv("LOKI_URL")
	if lokiURL == "" {
		lokiURL = "http://lgtm.observability:3100/loki/api/v1/push"
	}

	// Create Loki writer with labels
	labels := map[string]string{
		"app":     "go-spikes",
		"job":     "go-spikes",
		"version": "1.0.0",
	}

	lokiWriter = NewLokiWriter(lokiURL, labels)

	// Configure Loki log level (default to INFO, can be overridden by env var)
	lokiLogLevel := zerolog.InfoLevel
	if envLevel := os.Getenv("LOKI_LOG_LEVEL"); envLevel != "" {
		switch envLevel {
		case "debug":
			lokiLogLevel = zerolog.DebugLevel
		case "info":
			lokiLogLevel = zerolog.InfoLevel
		case "warn":
			lokiLogLevel = zerolog.WarnLevel
		case "error":
			lokiLogLevel = zerolog.ErrorLevel
		}
	}
	lokiWriter.SetMinLevel(lokiLogLevel)

	// Create multi-writer for both console and Loki
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	multi := NewMultiWriter(consoleWriter, lokiWriter)

	// Recreate logger with multi-writer
	Logger = zerolog.New(multi).
		Level(Level).
		With().
		Timestamp().
		Caller().
		Logger()

	log.Logger = Logger

	Logger.Info().
		Str("loki_url", lokiURL).
		Str("loki_min_level", lokiLogLevel.String()).
		Msg("Loki logging enabled")
	return nil
}

// ShutdownOTEL properly shuts down the Loki logger
func ShutdownOTEL(ctx context.Context) error {
	if lokiWriter != nil {
		return lokiWriter.Close()
	}
	return nil
}

// SetLokiLogLevel sets the minimum log level for Loki export
func SetLokiLogLevel(level zerolog.Level) {
	if lokiWriter != nil {
		lokiWriter.SetMinLevel(level)
		Logger.Info().Str("level", level.String()).Msg("Loki log level updated")
	}
}
