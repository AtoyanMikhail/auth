package logger

import (
	"io"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type loggerImpl struct {
	zapLogger *zap.Logger
	level     zap.AtomicLevel
	encoder   zapcore.Encoder
	teedCore  zapcore.Core
}

// New creates a new logger instance that writes logs to the provided io.Writer interfaces.
func New(writers ...io.Writer) Logger {
	// Конфигурация по умолчанию
	level := zap.NewAtomicLevelAt(zap.InfoLevel)
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoder := zapcore.NewJSONEncoder(encoderConfig)

	cores := make([]zapcore.Core, 0, len(writers))
	for _, w := range writers {
		core := zapcore.NewCore(
			encoder,
			zapcore.AddSync(w),
			level,
		)
		cores = append(cores, core)
	}

	teedCore := zapcore.NewTee(cores...)

	return &loggerImpl{
		zapLogger: zap.New(teedCore, zap.AddCaller(), zap.AddCallerSkip(1)),
		level:     level,
		encoder:   encoder,
		teedCore:  teedCore,
	}
}

// Initialize sets up the global logger instance with the specified writers. Thread-safe.
func Initialize(writers ...io.Writer) {
	initOnce.Do(func() {
		globalLogger = New(writers...)
	})
}

// Global returns the global logger instance, initializing it to stdout if not already set.
func Global() Logger {
	if globalLogger == nil {
		Initialize(os.Stdout)
	}
	return globalLogger
}

// String returns a string field for structured logging.
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int returns an int field for structured logging.
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Error returns an error field for structured logging.
func Error(err error) Field {
	return Field{Key: "error", Value: err.Error()}
}

// Any returns a generic field for structured logging.
func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// Debug logs a message at DebugLevel.
func (l *loggerImpl) Debug(msg string, fields ...Field) {
	l.zapLogger.Debug(msg, convertFields(fields)...)
}

// Info logs a message at InfoLevel.
func (l *loggerImpl) Info(msg string, fields ...Field) {
	l.zapLogger.Info(msg, convertFields(fields)...)
}

// Warn logs a message at WarnLevel.
func (l *loggerImpl) Warn(msg string, fields ...Field) {
	l.zapLogger.Warn(msg, convertFields(fields)...)
}

// Error logs a message at ErrorLevel.
func (l *loggerImpl) Error(msg string, fields ...Field) {
	l.zapLogger.Error(msg, convertFields(fields)...)
}

// Fatal logs a message at FatalLevel and then calls os.Exit(1).
func (l *loggerImpl) Fatal(msg string, fields ...Field) {
	l.zapLogger.Fatal(msg, convertFields(fields)...)
}

// Panic logs a message at PanicLevel and then panics.
func (l *loggerImpl) Panic(msg string, fields ...Field) {
	l.zapLogger.Panic(msg, convertFields(fields)...)
}

// With returns a new logger instance with additional structured fields.
func (l *loggerImpl) With(fields ...Field) Logger {
	return &loggerImpl{
		zapLogger: l.zapLogger.With(convertFields(fields)...),
		level:     l.level,
		encoder:   l.encoder,
		teedCore:  l.teedCore,
	}
}

// Sync flushes any buffered log entries.
func (l *loggerImpl) Sync() error {
	return l.zapLogger.Sync()
}

// SetLevel dynamically sets the logging level for this logger instance.
func (l *loggerImpl) SetLevel(level Level) {
	l.level.SetLevel(toZapLevel(level))
}

func convertFields(fields []Field) []zap.Field {
	zapFields := make([]zap.Field, len(fields))
	for i, f := range fields {
		zapFields[i] = zap.Any(f.Key, f.Value)
	}
	return zapFields
}

func toZapLevel(level Level) zapcore.Level {
	switch level {
	case DebugLevel:
		return zap.DebugLevel
	case InfoLevel:
		return zap.InfoLevel
	case WarnLevel:
		return zap.WarnLevel
	case ErrorLevel:
		return zap.ErrorLevel
	case PanicLevel:
		return zap.PanicLevel
	case FatalLevel:
		return zap.FatalLevel
	default:
		return zap.InfoLevel
	}
}
