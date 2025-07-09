package logger

import (
	"context"
	"fc-pr-tracker/internal/config"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Handler fake para simular erro
type errorHandler struct{}

func (e *errorHandler) Enabled(ctx context.Context, level slog.Level) bool { return true }
func (e *errorHandler) Handle(ctx context.Context, r slog.Record) error {
	return fmt.Errorf("erro fake")
}
func (e *errorHandler) WithAttrs(attrs []slog.Attr) slog.Handler { return e }
func (e *errorHandler) WithGroup(name string) slog.Handler       { return e }

// Handler fake que sempre retorna false em Enabled
type disabledHandler struct{}

func (d *disabledHandler) Enabled(ctx context.Context, level slog.Level) bool { return false }
func (d *disabledHandler) Handle(ctx context.Context, r slog.Record) error    { return nil }
func (d *disabledHandler) WithAttrs(attrs []slog.Attr) slog.Handler           { return d }
func (d *disabledHandler) WithGroup(name string) slog.Handler                 { return d }

func TestGetLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected slog.Level
	}{
		{"debug level", "debug", slog.LevelDebug},
		{"info level", "info", slog.LevelInfo},
		{"warn level", "warn", slog.LevelWarn},
		{"error level", "error", slog.LevelError},
		{"unknown level", "unknown", slog.LevelInfo},
		{"empty level", "", slog.LevelInfo},
		{"uppercase debug", "DEBUG", slog.LevelInfo},
		{"mixed case info", "Info", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getLogLevel(tt.level)
			if result != tt.expected {
				t.Errorf("Expected level %v for input '%s', got %v", tt.expected, tt.level, result)
			}
		})
	}
}

func TestMultiHandler_Enabled(t *testing.T) {
	// Create test handlers
	fileHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	stdoutHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})

	multiHandler := &MultiHandler{
		handlers: []slog.Handler{fileHandler, stdoutHandler},
	}

	// Test that handler is enabled when any handler is enabled
	if !multiHandler.Enabled(nil, slog.LevelInfo) {
		t.Error("Expected handler to be enabled for Info level")
	}

	if !multiHandler.Enabled(nil, slog.LevelDebug) {
		t.Error("Expected handler to be enabled for Debug level")
	}

	if !multiHandler.Enabled(nil, slog.LevelError) {
		t.Error("Expected handler to be enabled for Error level")
	}
}

func TestMultiHandler_WithAttrs(t *testing.T) {
	fileHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	stdoutHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})

	multiHandler := &MultiHandler{
		handlers: []slog.Handler{fileHandler, stdoutHandler},
	}

	// Test WithAttrs returns a new handler
	newHandler := multiHandler.WithAttrs([]slog.Attr{slog.String("test", "value")})
	if newHandler == nil {
		t.Error("Expected WithAttrs to return a new handler")
	}

	// Verify it's a different handler
	if newHandler == multiHandler {
		t.Error("Expected WithAttrs to return a new handler instance")
	}
}

func TestMultiHandler_WithGroup(t *testing.T) {
	fileHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	stdoutHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})

	multiHandler := &MultiHandler{
		handlers: []slog.Handler{fileHandler, stdoutHandler},
	}

	// Test WithGroup returns a new handler
	newHandler := multiHandler.WithGroup("testgroup")
	if newHandler == nil {
		t.Error("Expected WithGroup to return a new handler")
	}

	// Verify it's a different handler
	if newHandler == multiHandler {
		t.Error("Expected WithGroup to return a new handler instance")
	}
}

func TestInit_FileOnly(t *testing.T) {
	// Create temporary config
	cfg := &config.Config{
		Log: struct {
			File       string `yaml:"file"`
			Level      string `yaml:"level"`
			Format     string `yaml:"format"`
			MaxSizeMB  int    `yaml:"max_size_mb"`
			MaxBackups int    `yaml:"max_backups"`
			MaxAgeDays int    `yaml:"max_age_days"`
			Compress   bool   `yaml:"compress"`
			Stdout     bool   `yaml:"stdout"`
		}{
			File:       "test_logs/test.log",
			Level:      "info",
			MaxSizeMB:  100,
			MaxBackups: 3,
			MaxAgeDays: 30,
			Compress:   false,
			Stdout:     false,
		},
	}

	// Clean up after test
	defer func() {
		os.RemoveAll("test_logs")
	}()

	// Initialize logger
	Init(cfg)

	// Verify logger is set
	if slog.Default() == nil {
		t.Error("Expected logger to be initialized")
	}

	// Test that we can log
	slog.Info("Test message")
	slog.Error("Test error")

	// Verify log file was created
	if _, err := os.Stat("test_logs/test.log"); os.IsNotExist(err) {
		t.Error("Expected log file to be created")
	}
}

func TestInit_FileAndStdout(t *testing.T) {
	// Create temporary config
	cfg := &config.Config{
		Log: struct {
			File       string `yaml:"file"`
			Level      string `yaml:"level"`
			Format     string `yaml:"format"`
			MaxSizeMB  int    `yaml:"max_size_mb"`
			MaxBackups int    `yaml:"max_backups"`
			MaxAgeDays int    `yaml:"max_age_days"`
			Compress   bool   `yaml:"compress"`
			Stdout     bool   `yaml:"stdout"`
		}{
			File:       "test_logs/test_stdout.log",
			Level:      "debug",
			MaxSizeMB:  100,
			MaxBackups: 3,
			MaxAgeDays: 30,
			Compress:   false,
			Stdout:     true,
		},
	}

	// Clean up after test
	defer func() {
		os.RemoveAll("test_logs")
	}()

	// Initialize logger
	Init(cfg)

	// Verify logger is set
	if slog.Default() == nil {
		t.Error("Expected logger to be initialized")
	}

	// Test that we can log
	slog.Debug("Test debug message")
	slog.Info("Test info message")
	slog.Error("Test error message")

	// Verify log file was created
	if _, err := os.Stat("test_logs/test_stdout.log"); os.IsNotExist(err) {
		t.Error("Expected log file to be created")
	}
}

func TestInit_InvalidLogDirectory(t *testing.T) {
	// Create config with invalid log directory
	cfg := &config.Config{
		Log: struct {
			File       string `yaml:"file"`
			Level      string `yaml:"level"`
			Format     string `yaml:"format"`
			MaxSizeMB  int    `yaml:"max_size_mb"`
			MaxBackups int    `yaml:"max_backups"`
			MaxAgeDays int    `yaml:"max_age_days"`
			Compress   bool   `yaml:"compress"`
			Stdout     bool   `yaml:"stdout"`
		}{
			File:       "/invalid/path/test.log",
			Level:      "info",
			MaxSizeMB:  100,
			MaxBackups: 3,
			MaxAgeDays: 30,
			Compress:   false,
			Stdout:     false,
		},
	}

	// This should not panic and should handle the error gracefully
	Init(cfg)

	// Verify logger is still set (should fall back to default)
	if slog.Default() == nil {
		t.Error("Expected logger to be set even with invalid directory")
	}
}

func TestInit_DifferentLogLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error"}

	for _, level := range levels {
		t.Run("level_"+level, func(t *testing.T) {
			cfg := &config.Config{
				Log: struct {
					File       string `yaml:"file"`
					Level      string `yaml:"level"`
					Format     string `yaml:"format"`
					MaxSizeMB  int    `yaml:"max_size_mb"`
					MaxBackups int    `yaml:"max_backups"`
					MaxAgeDays int    `yaml:"max_age_days"`
					Compress   bool   `yaml:"compress"`
					Stdout     bool   `yaml:"stdout"`
				}{
					File:       filepath.Join("test_logs", level+".log"),
					Level:      level,
					MaxSizeMB:  100,
					MaxBackups: 3,
					MaxAgeDays: 30,
					Compress:   false,
					Stdout:     false,
				},
			}

			// Clean up after test
			defer func() {
				os.RemoveAll("test_logs")
			}()

			// Initialize logger
			Init(cfg)

			// Verify logger is set
			if slog.Default() == nil {
				t.Errorf("Expected logger to be initialized for level %s", level)
			}

			// Test logging
			slog.Info("Test message")
		})
	}
}

func TestInit_LogDirError(t *testing.T) {
	cfg := &config.Config{
		Log: struct {
			File       string `yaml:"file"`
			Level      string `yaml:"level"`
			Format     string `yaml:"format"`
			MaxSizeMB  int    `yaml:"max_size_mb"`
			MaxBackups int    `yaml:"max_backups"`
			MaxAgeDays int    `yaml:"max_age_days"`
			Compress   bool   `yaml:"compress"`
			Stdout     bool   `yaml:"stdout"`
		}{
			File:   string([]byte{0}), // caminho inválido
			Level:  "info",
			Stdout: false,
		},
	}
	Init(cfg)
	// Espera-se que não dê panic, apenas um print de erro
}

func TestInit_LogDirErrorWithValidPath(t *testing.T) {
	// Teste com um caminho que vai falhar no Windows
	cfg := &config.Config{
		Log: struct {
			File       string `yaml:"file"`
			Level      string `yaml:"level"`
			Format     string `yaml:"format"`
			MaxSizeMB  int    `yaml:"max_size_mb"`
			MaxBackups int    `yaml:"max_backups"`
			MaxAgeDays int    `yaml:"max_age_days"`
			Compress   bool   `yaml:"compress"`
			Stdout     bool   `yaml:"stdout"`
		}{
			File:   "C:\\Windows\\System32\\test.log", // caminho que pode falhar
			Level:  "info",
			Stdout: false,
		},
	}
	Init(cfg)
	// Espera-se que não dê panic, apenas um print de erro
}

func TestMultiHandler_Enabled_AllDisabled(t *testing.T) {
	// Criar handlers que não estão habilitados para nenhum nível
	disabledHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})
	multi := &MultiHandler{
		handlers: []slog.Handler{disabledHandler},
	}

	// Testar com um nível que não está habilitado (Info < Error)
	if multi.Enabled(nil, slog.LevelInfo) {
		t.Error("Expected handler to be disabled for Info level when all handlers require Error level")
	}
}

func TestMultiHandler_Enabled_AllFakeDisabled(t *testing.T) {
	// Testar com handlers fake que sempre retornam false
	multi := &MultiHandler{
		handlers: []slog.Handler{&disabledHandler{}, &disabledHandler{}},
	}

	// Testar com qualquer nível - deve retornar false
	if multi.Enabled(nil, slog.LevelInfo) {
		t.Error("Expected handler to be disabled when all fake handlers return false")
	}
	if multi.Enabled(nil, slog.LevelError) {
		t.Error("Expected handler to be disabled when all fake handlers return false")
	}
}

func TestMultiHandler_HandleError(t *testing.T) {
	multi := &MultiHandler{
		handlers: []slog.Handler{&errorHandler{}, &errorHandler{}},
	}
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "teste", 0)
	err := multi.Handle(context.Background(), record)
	if err == nil {
		t.Error("Esperava erro do handler fake")
	}
}

func TestMultiHandler_WithAttrsAndGroupType(t *testing.T) {
	fileHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	multi := &MultiHandler{handlers: []slog.Handler{fileHandler}}
	h1 := multi.WithAttrs([]slog.Attr{slog.String("k", "v")})
	h2 := multi.WithGroup("grupo")
	if _, ok := h1.(*MultiHandler); !ok {
		t.Error("WithAttrs deve retornar MultiHandler")
	}
	if _, ok := h2.(*MultiHandler); !ok {
		t.Error("WithGroup deve retornar MultiHandler")
	}
}

func TestInit_OnlyStdout(t *testing.T) {
	cfg := &config.Config{
		Log: struct {
			File       string `yaml:"file"`
			Level      string `yaml:"level"`
			Format     string `yaml:"format"`
			MaxSizeMB  int    `yaml:"max_size_mb"`
			MaxBackups int    `yaml:"max_backups"`
			MaxAgeDays int    `yaml:"max_age_days"`
			Compress   bool   `yaml:"compress"`
			Stdout     bool   `yaml:"stdout"`
		}{
			File:   "logs/test.log",
			Level:  "info",
			Stdout: true,
		},
	}
	Init(cfg)
	slog.Info("Só stdout")
}

func TestInit_LogDirErrorWithInvalidPath(t *testing.T) {
	// Teste com um caminho que realmente vai falhar
	cfg := &config.Config{
		Log: struct {
			File       string `yaml:"file"`
			Level      string `yaml:"level"`
			Format     string `yaml:"format"`
			MaxSizeMB  int    `yaml:"max_size_mb"`
			MaxBackups int    `yaml:"max_backups"`
			MaxAgeDays int    `yaml:"max_age_days"`
			Compress   bool   `yaml:"compress"`
			Stdout     bool   `yaml:"stdout"`
		}{
			File:   "\\invalid\\path\\with\\backslashes\\test.log", // caminho inválido
			Level:  "info",
			Stdout: false,
		},
	}
	Init(cfg)
	// Espera-se que não dê panic, apenas um print de erro
}
