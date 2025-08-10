package logger

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
)

var Logger zerolog.Logger

func Init() {
	zerolog.TimeFieldFormat = time.RFC3339
	
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	
	level, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	
	Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).
		Level(level).
		With().
		Timestamp().
		Caller().
		Logger()
	
	log.Logger = Logger
	
	Logger.Info().Str("level", level.String()).Msg("Logger initialized")
}

func Get() *zerolog.Logger {
	return &Logger
}

// WithContext returns a logger with trace context if available
func WithContext(ctx context.Context) zerolog.Logger {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		return Logger.With().
			Str("trace_id", spanCtx.TraceID().String()).
			Str("span_id", spanCtx.SpanID().String()).
			Logger()
	}
	return Logger
}

// Ctx returns a zerolog context logger with trace information
func Ctx(ctx context.Context) *zerolog.Logger {
	l := WithContext(ctx)
	return &l
}