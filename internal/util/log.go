package util

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// Logger is a small facade used across the project.
// It gives you printf-style methods and a With(...) for fields.
type Logger interface {
	// printf-style
	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any) // logs and os.Exit(1)

	// structured chaining
	With(args ...any) Logger

	// (optional) expose underlying slog for advanced use
	Slog() *slog.Logger
}

type slogLogger struct {
	l *slog.Logger
}

// NewLogger creates a slog-backed logger.
// Controlled by env:
//
//	LOG_LEVEL  = debug|info|warn|error   (default: info)
//	LOG_FORMAT = json|text               (default: json)
//	LOG_SOURCE = true|false              (default: false) - include source file:line
func NewLogger() Logger {
	level := parseLevel(os.Getenv("LOG_LEVEL"))
	addSource := strings.EqualFold(os.Getenv("LOG_SOURCE"), "true")

	opts := &slog.HandlerOptions{Level: level, AddSource: addSource}

	var h slog.Handler
	switch strings.ToLower(os.Getenv("LOG_FORMAT")) {
	case "text":
		h = slog.NewTextHandler(os.Stdout, opts)
	default:
		h = slog.NewJSONHandler(os.Stdout, opts)
	}

	return &slogLogger{l: slog.New(h)}
}

// NewTestLogger returns a text logger at debug level for tests.
func NewTestLogger() Logger {
	opts := &slog.HandlerOptions{Level: slog.LevelDebug, AddSource: false}
	return &slogLogger{l: slog.New(slog.NewTextHandler(os.Stdout, opts))}
}

// --- Logger implementation ---

func (s *slogLogger) Debugf(format string, args ...any) { s.l.Debug(fmt.Sprintf(format, args...)) }
func (s *slogLogger) Infof(format string, args ...any)  { s.l.Info(fmt.Sprintf(format, args...)) }
func (s *slogLogger) Warnf(format string, args ...any)  { s.l.Warn(fmt.Sprintf(format, args...)) }
func (s *slogLogger) Errorf(format string, args ...any) { s.l.Error(fmt.Sprintf(format, args...)) }
func (s *slogLogger) Fatalf(format string, args ...any) {
	s.l.Error(fmt.Sprintf(format, args...))
	os.Exit(1)
}

func (s *slogLogger) With(args ...any) Logger {
	return &slogLogger{l: s.l.With(args...)}
}

func (s *slogLogger) Slog() *slog.Logger { return s.l }

// --- helpers ---

func parseLevel(v string) slog.Leveler {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
