package logger

import (
	"io"
	"log/slog"
	"os"
)

var (
	// Default is the default logger instance used throughout the application
	Default *slog.Logger
)

// Level represents the logging level
type Level = slog.Level

const (
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
)

// Init initializes the default logger with the specified level and output
// level: the minimum log level to output (LevelDebug, LevelInfo, LevelWarn, LevelError)
// output: where to write logs (typically os.Stdout or os.Stderr)
func Init(level Level, output io.Writer) {
	opts := &slog.HandlerOptions{
		Level: level,
	}

	handler := slog.NewTextHandler(output, opts)
	Default = slog.New(handler)
}

// InitJSON initializes the default logger with JSON output
func InitJSON(level Level, output io.Writer) {
	opts := &slog.HandlerOptions{
		Level: level,
	}

	handler := slog.NewJSONHandler(output, opts)
	Default = slog.New(handler)
}

// ParseLevel converts a string level to slog.Level
func ParseLevel(level string) Level {
	switch level {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

func init() {
	// Initialize with default settings: Info level, text output to stdout
	Init(LevelInfo, os.Stdout)
}

// Convenience functions that wrap the Default logger

// Debug logs a debug message
func Debug(msg string, args ...any) {
	Default.Debug(msg, args...)
}

// Info logs an info message
func Info(msg string, args ...any) {
	Default.Info(msg, args...)
}

// Warn logs a warning message
func Warn(msg string, args ...any) {
	Default.Warn(msg, args...)
}

// Error logs an error message
func Error(msg string, args ...any) {
	Default.Error(msg, args...)
}

// With returns a new logger with the given attributes
func With(args ...any) *slog.Logger {
	return Default.With(args...)
}
