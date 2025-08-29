package logger

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger
var Sugar *zap.SugaredLogger

// ZapLogger wraps zap.Logger to provide zerolog-compatible API
type ZapLogger struct {
	*zap.Logger
}

// ZapEvent represents a log event with zerolog-like chaining
type ZapEvent struct {
	logger *zap.Logger
	level  zapcore.Level
	fields []zap.Field
	msg    string
}

// NewZapEvent creates a new log event
func NewZapEvent(logger *zap.Logger, level zapcore.Level) *ZapEvent {
	return &ZapEvent{
		logger: logger,
		level:  level,
		fields: make([]zap.Field, 0),
	}
}

// Err adds an error field
func (e *ZapEvent) Err(err error) *ZapEvent {
	e.fields = append(e.fields, zap.Error(err))
	return e
}

// Str adds a string field
func (e *ZapEvent) Str(key, val string) *ZapEvent {
	e.fields = append(e.fields, zap.String(key, val))
	return e
}

// Bool adds a boolean field
func (e *ZapEvent) Bool(key string, val bool) *ZapEvent {
	e.fields = append(e.fields, zap.Bool(key, val))
	return e
}

// Dur adds a duration field
func (e *ZapEvent) Dur(key string, val interface{}) *ZapEvent {
	switch v := val.(type) {
	case fmt.Stringer:
		e.fields = append(e.fields, zap.String(key, v.String()))
	default:
		e.fields = append(e.fields, zap.Any(key, v))
	}
	return e
}

// Interface adds an interface field
func (e *ZapEvent) Interface(key string, val interface{}) *ZapEvent {
	e.fields = append(e.fields, zap.Any(key, val))
	return e
}

// Int adds an int field
func (e *ZapEvent) Int(key string, val int) *ZapEvent {
	e.fields = append(e.fields, zap.Int(key, val))
	return e
}

// Int32 adds an int32 field
func (e *ZapEvent) Int32(key string, val int32) *ZapEvent {
	e.fields = append(e.fields, zap.Int32(key, val))
	return e
}

// Int64 adds an int64 field
func (e *ZapEvent) Int64(key string, val int64) *ZapEvent {
	e.fields = append(e.fields, zap.Int64(key, val))
	return e
}

// Any adds an any field
func (e *ZapEvent) Any(key string, val interface{}) *ZapEvent {
	e.fields = append(e.fields, zap.Any(key, val))
	return e
}

// Msg logs the message with accumulated fields
func (e *ZapEvent) Msg(msg string) {
	switch e.level {
	case zapcore.DebugLevel:
		e.logger.Debug(msg, e.fields...)
	case zapcore.InfoLevel:
		e.logger.Info(msg, e.fields...)
	case zapcore.WarnLevel:
		e.logger.Warn(msg, e.fields...)
	case zapcore.ErrorLevel:
		e.logger.Error(msg, e.fields...)
	case zapcore.FatalLevel:
		e.logger.Fatal(msg, e.fields...)
	}
}

// Info returns an info-level log event
func (l *ZapLogger) Info() *ZapEvent {
	return NewZapEvent(l.Logger, zapcore.InfoLevel)
}

// Warn returns a warn-level log event
func (l *ZapLogger) Warn() *ZapEvent {
	return NewZapEvent(l.Logger, zapcore.WarnLevel)
}

// Error returns an error-level log event
func (l *ZapLogger) Error() *ZapEvent {
	return NewZapEvent(l.Logger, zapcore.ErrorLevel)
}

// Fatal returns a fatal-level log event
func (l *ZapLogger) Fatal() *ZapEvent {
	return NewZapEvent(l.Logger, zapcore.FatalLevel)
}

// Debug returns a debug-level log event
func (l *ZapLogger) Debug() *ZapEvent {
	return NewZapEvent(l.Logger, zapcore.DebugLevel)
}

// Trace returns a debug-level log event (mapped to debug since zap doesn't have trace)
func (l *ZapLogger) Trace() *ZapEvent {
	return NewZapEvent(l.Logger, zapcore.DebugLevel)
}

func Init() {
	// Create zap config for JSON logging to stdout
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}
	
	// Set default level to info
	config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	
	// Enable caller information
	config.Development = false
	config.DisableCaller = false
	config.DisableStacktrace = false
	
	// Build logger
	var err error
	Logger, err = config.Build()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}
	
	Sugar = Logger.Sugar()
	
	Logger.Info("Logger initialized", zap.String("level", "info"))
}

func Get() *ZapLogger {
	return &ZapLogger{Logger: Logger}
}

// WithContext returns a logger with trace context if available
func WithContext(ctx context.Context) *ZapLogger {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		logger := Logger.With(
			zap.String("trace_id", spanCtx.TraceID().String()),
			zap.String("span_id", spanCtx.SpanID().String()),
		)
		return &ZapLogger{Logger: logger}
	}
	return &ZapLogger{Logger: Logger}
}

// Ctx returns a logger with trace information from context
func Ctx(ctx context.Context) *ZapLogger {
	return WithContext(ctx)
}

func SetLogLevel(level string) {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zap.DebugLevel
	case "info":
		zapLevel = zap.InfoLevel
	case "warn":
		zapLevel = zap.WarnLevel
	case "error":
		zapLevel = zap.ErrorLevel
	case "fatal":
		zapLevel = zap.FatalLevel
	case "panic":
		zapLevel = zap.PanicLevel
	default:
		Logger.Warn("Unknown log level, defaulting to INFO", zap.String("level", level))
		zapLevel = zap.InfoLevel
	}
	
	// Rebuild logger with new level
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}
	config.Level = zap.NewAtomicLevelAt(zapLevel)
	config.Development = false
	config.DisableCaller = false
	config.DisableStacktrace = false
	
	var err error
	Logger, err = config.Build()
	if err != nil {
		Logger.Error("Failed to rebuild logger with new level", zap.Error(err))
		return
	}
	
	Sugar = Logger.Sugar()
	Logger.Info("Log level updated", zap.String("level", level))
}

// EnableOTEL is kept for compatibility but simplified for STDIO-only logging
func EnableOTEL(ctx context.Context) error {
	Logger.Info("OTEL logging enabled (writing to STDIO)")
	return nil
}

// ShutdownOTEL properly shuts down the logger
func ShutdownOTEL(ctx context.Context) error {
	if Logger != nil {
		return Logger.Sync()
	}
	return nil
}

// Shutdown syncs the logger
func Shutdown() error {
	if Logger != nil {
		return Logger.Sync()
	}
	return nil
}
