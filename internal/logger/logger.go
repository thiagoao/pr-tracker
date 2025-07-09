package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"fc-pr-tracker/internal/config"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Init initializes the logger with the given configuration
func Init(cfg *config.Config) {
	var handlers []slog.Handler

	// Create log directory if it doesn't exist
	logDir := filepath.Dir(cfg.Log.File)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		// Use fmt instead of slog since logger isn't initialized yet
		fmt.Printf("Failed to create log directory: %v\n", err)
		return
	}

	// File handler with rotation
	fileHandler := slog.NewJSONHandler(&lumberjack.Logger{
		Filename:   cfg.Log.File,
		MaxSize:    cfg.Log.MaxSizeMB,
		MaxBackups: cfg.Log.MaxBackups,
		MaxAge:     cfg.Log.MaxAgeDays,
		Compress:   cfg.Log.Compress,
	}, &slog.HandlerOptions{
		Level: getLogLevel(cfg.Log.Level),
	})

	handlers = append(handlers, fileHandler)

	// Stdout handler if enabled
	if cfg.Log.Stdout {
		stdoutHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: getLogLevel(cfg.Log.Level),
		})
		handlers = append(handlers, stdoutHandler)
	}

	// Create multi-handler logger
	var logger *slog.Logger
	if len(handlers) > 1 {
		// Use a multi-handler approach for both file and stdout
		logger = slog.New(handlers[0])
		// Set up a custom handler that writes to both
		multiHandler := &MultiHandler{
			handlers: handlers,
		}
		logger = slog.New(multiHandler)
	} else {
		// Use single handler
		logger = slog.New(handlers[0])
	}
	slog.SetDefault(logger)
}

// MultiHandler writes to multiple handlers
type MultiHandler struct {
	handlers []slog.Handler
}

// Enabled returns true if any handler is enabled for the given level
func (h *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

// Handle writes the record to all handlers
func (h *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	var lastErr error
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, r.Level) {
			if err := handler.Handle(ctx, r); err != nil {
				lastErr = err
			}
		}
	}
	return lastErr
}

// WithAttrs returns a new handler with the given attributes
func (h *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithAttrs(attrs)
	}
	return &MultiHandler{handlers: handlers}
}

// WithGroup returns a new handler with the given group
func (h *MultiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithGroup(name)
	}
	return &MultiHandler{handlers: handlers}
}

// getLogLevel converts string level to slog.Level
func getLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
