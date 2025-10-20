package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	// Logger is the global logger instance
	Logger zerolog.Logger
)

// Init initializes the global logger
func Init(level string, enableConsole bool) {
	// Set log level
	var logLevel zerolog.Level
	switch level {
	case "debug":
		logLevel = zerolog.DebugLevel
	case "info":
		logLevel = zerolog.InfoLevel
	case "warn":
		logLevel = zerolog.WarnLevel
	case "error":
		logLevel = zerolog.ErrorLevel
	default:
		logLevel = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(logLevel)

	// Configure output
	var output io.Writer = os.Stderr

	if enableConsole {
		// Pretty console output for development
		output = zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.RFC3339,
		}
	}

	// Create logger
	Logger = zerolog.New(output).
		With().
		Timestamp().
		Caller().
		Logger()

	// Set as global logger
	log.Logger = Logger
}

// Debug logs a debug message
func Debug(msg string) {
	Logger.Debug().Msg(msg)
}

// Debugf logs a formatted debug message
func Debugf(format string, args ...interface{}) {
	Logger.Debug().Msgf(format, args...)
}

// Info logs an info message
func Info(msg string) {
	Logger.Info().Msg(msg)
}

// Infof logs a formatted info message
func Infof(format string, args ...interface{}) {
	Logger.Info().Msgf(format, args...)
}

// Warn logs a warning message
func Warn(msg string) {
	Logger.Warn().Msg(msg)
}

// Warnf logs a formatted warning message
func Warnf(format string, args ...interface{}) {
	Logger.Warn().Msgf(format, args...)
}

// Error logs an error message
func Error(msg string) {
	Logger.Error().Msg(msg)
}

// Errorf logs a formatted error message
func Errorf(format string, args ...interface{}) {
	Logger.Error().Msgf(format, args...)
}

// Fatal logs a fatal message and exits
func Fatal(msg string) {
	Logger.Fatal().Msg(msg)
}

// Fatalf logs a formatted fatal message and exits
func Fatalf(format string, args ...interface{}) {
	Logger.Fatal().Msgf(format, args...)
}

// WithField adds a field to the logger
func WithField(key string, value interface{}) *zerolog.Event {
	return Logger.Info().Interface(key, value)
}

// WithError adds an error to the logger
func WithError(err error) *zerolog.Event {
	return Logger.Error().Err(err)
}
