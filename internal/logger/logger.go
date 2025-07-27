package logger

import "sync"

// Level represents a level of logging. If the level set in the logger is higher than it,
// the message will not be logged.
type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	PanicLevel
	FatalLevel
)

var (
	globalLogger Logger
	initOnce     sync.Once
)

type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)
	Panic(msg string, fields ...Field)
	With(fields ...Field) Logger
	Sync() error
	SetLevel(level Level)
}

// Field represents a json field in a log message
type Field struct {
	Key   string
	Value interface{}
}
