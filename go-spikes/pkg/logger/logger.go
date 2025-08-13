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
var Level zerolog.Level = zerolog.InfoLevel

func Init() {
	zerolog.TimeFieldFormat = time.RFC3339

	defaultLevel := zerolog.InfoLevel

	Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).
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

func SetDebugLevel() {
	setLevel(zerolog.DebugLevel)
}

func SetInfoLevel() {
	setLevel(zerolog.InfoLevel)
}

func setLevel(level zerolog.Level) {
	Level = level
	zerolog.SetGlobalLevel(level)
}
