// internal/utils/logging.go
package utils

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	LogFileName = "app.log"
	LogFileMode = 0644
)

var Logger *zap.Logger

// Init configures zap to write to both console and a log file.
// This should be called once at application startup.
func Init() error {
	logFile, err := os.OpenFile(LogFileName, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, LogFileMode)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", LogFileName, err)
	}

	// Configure encoder
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	// Create cores for console and file
	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
	fileEncoder := zapcore.NewJSONEncoder(encoderConfig)

	// Determine log level from environment variable `LOG_LEVEL` (default: info)
	envLevel := os.Getenv("LOG_LEVEL")
	var level zapcore.Level
	if envLevel == "" {
		level = zapcore.InfoLevel
	} else {
		if err := level.UnmarshalText([]byte(envLevel)); err != nil {
			fmt.Printf("unknown LOG_LEVEL '%s', defaulting to 'info'\n", envLevel)
			level = zapcore.InfoLevel
		}
	}

	consoleCore := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level)
	fileCore := zapcore.NewCore(fileEncoder, zapcore.AddSync(logFile), level)

	// Combine cores
	core := zapcore.NewTee(consoleCore, fileCore)
	Logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	// Emit a startup message describing the chosen log level.
	Logger.Info("logging initialized", zap.String("log_level", level.String()))

	return nil
}

// Sync flushes any buffered log entries.
func Sync() error {
	if Logger != nil {
		return Logger.Sync()
	}
	return nil
}

// WithComponent returns a logger pre-bound with a `component` field so callers
// don't have to repeat the same field across messages in a component.
func WithComponent(component string) *zap.Logger {
	if Logger == nil {
		return nil
	}
	return Logger.With(zap.String("component", component))
}
