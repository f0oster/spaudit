package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"
)

// Config holds logging configuration
type Config struct {
	Level  string `env:"LOG_LEVEL" default:"info"`
	Format string `env:"LOG_FORMAT" default:"json"`
	Output string `env:"LOG_OUTPUT" default:"stdout"`
}

// DefaultConfig returns the default logging configuration
func DefaultConfig() *Config {
	return &Config{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}
}

// Logger wraps slog.Logger with additional context methods
type Logger struct {
	*slog.Logger
}

// NewLogger creates a new structured logger from configuration
func NewLogger(cfg *Config) *Logger {
	var writer io.Writer = os.Stdout

	// Configure output destination
	switch strings.ToLower(cfg.Output) {
	case "stderr":
		writer = os.Stderr
	case "stdout", "":
		writer = os.Stdout
	default:
		// Default to stdout for unrecognized output targets
		writer = os.Stdout
	}

	// Configure log level
	var level slog.Level
	switch strings.ToLower(cfg.Level) {
	case "debug":
		level = slog.LevelDebug
	case "info", "":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Configure handler based on format
	var handler slog.Handler
	handlerOpts := &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize timestamp format
			if a.Key == slog.TimeKey {
				return slog.String("timestamp", a.Value.Time().Format(time.RFC3339))
			}
			return a
		},
	}

	switch strings.ToLower(cfg.Format) {
	case "text", "console":
		handler = slog.NewTextHandler(writer, handlerOpts)
	case "json", "":
		handler = slog.NewJSONHandler(writer, handlerOpts)
	default:
		handler = slog.NewJSONHandler(writer, handlerOpts)
	}

	return &Logger{
		Logger: slog.New(handler),
	}
}

// WithComponent adds component context to logger
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		Logger: l.Logger.With("component", component),
	}
}

// WithContext adds request context to logger (if available)
func (l *Logger) WithContext(ctx context.Context) *Logger {
	// Extract common context values if available
	if requestID, ok := ctx.Value("request_id").(string); ok {
		return &Logger{
			Logger: l.Logger.With("request_id", requestID),
		}
	}
	return l
}

// Audit logs audit-specific events with standard fields
func (l *Logger) Audit(msg string, siteURL string, attrs ...slog.Attr) {
	args := []any{"site_url", siteURL}
	for _, attr := range attrs {
		args = append(args, attr.Key, attr.Value)
	}
	l.Logger.Info(msg, args...)
}

// AuditError logs audit errors with standard fields
func (l *Logger) AuditError(msg string, err error, siteURL string, attrs ...slog.Attr) {
	args := []any{"site_url", siteURL, "error", err.Error()}
	for _, attr := range attrs {
		args = append(args, attr.Key, attr.Value)
	}
	l.Logger.Error(msg, args...)
}

// Performance logs performance metrics
func (l *Logger) Performance(operation string, duration time.Duration, attrs ...slog.Attr) {
	args := []any{"operation", operation, "duration_ms", duration.Milliseconds()}
	for _, attr := range attrs {
		args = append(args, attr.Key, attr.Value)
	}
	l.Logger.Info("performance", args...)
}

// SharePoint logs SharePoint-specific events
func (l *Logger) SharePoint(msg string, args ...any) {
	finalArgs := []any{"subsystem", "sharepoint"}
	finalArgs = append(finalArgs, args...)
	l.Logger.Info(msg, finalArgs...)
}

// Database logs database-specific events
func (l *Logger) Database(msg string, args ...any) {
	finalArgs := []any{"subsystem", "database"}
	finalArgs = append(finalArgs, args...)
	l.Logger.Debug(msg, finalArgs...)
}

// Security logs security-related events
func (l *Logger) Security(msg string, args ...any) {
	finalArgs := []any{"subsystem", "security"}
	finalArgs = append(finalArgs, args...)
	l.Logger.Info(msg, finalArgs...)
}

var defaultLogger *Logger

// SetDefault sets the default logger instance
func SetDefault(logger *Logger) {
	defaultLogger = logger
}

// Default returns the default logger instance
func Default() *Logger {
	if defaultLogger == nil {
		defaultLogger = NewLogger(DefaultConfig())
	}
	return defaultLogger
}

// Convenience functions using default logger
func Info(msg string, args ...any) {
	Default().Info(msg, args...)
}

func Debug(msg string, args ...any) {
	Default().Debug(msg, args...)
}

func Warn(msg string, args ...any) {
	Default().Warn(msg, args...)
}

func Error(msg string, args ...any) {
	Default().Error(msg, args...)
}

func Audit(msg string, siteURL string, attrs ...slog.Attr) {
	Default().Audit(msg, siteURL, attrs...)
}

func AuditError(msg string, err error, siteURL string, attrs ...slog.Attr) {
	Default().AuditError(msg, err, siteURL, attrs...)
}

func Performance(operation string, duration time.Duration, attrs ...slog.Attr) {
	Default().Performance(operation, duration, attrs...)
}
