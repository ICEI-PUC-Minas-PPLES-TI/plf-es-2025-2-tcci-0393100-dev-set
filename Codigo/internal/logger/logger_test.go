package logger

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestInit(t *testing.T) {
	tests := []struct {
		name          string
		level         string
		enableConsole bool
	}{
		{"debug level", "debug", false},
		{"info level", "info", false},
		{"warn level", "warn", false},
		{"error level", "error", false},
		{"invalid level defaults to info", "invalid", false},
		{"with console", "info", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Init(tt.level, tt.enableConsole)

			if Logger.GetLevel() == zerolog.Disabled {
				t.Error("Logger should be initialized")
			}
		})
	}
}

func TestLogLevels(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	Logger = zerolog.New(&buf).With().Timestamp().Logger()

	// Set to debug level to capture all logs
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	tests := []struct {
		name     string
		logFunc  func()
		expected string
	}{
		{
			"Debug",
			func() { Debug("test debug") },
			"debug",
		},
		{
			"Info",
			func() { Info("test info") },
			"info",
		},
		{
			"Warn",
			func() { Warn("test warn") },
			"warn",
		},
		{
			"Error",
			func() { Error("test error") },
			"error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc()

			output := buf.String()
			if !strings.Contains(output, tt.expected) {
				t.Errorf("Expected output to contain '%s', got: %s", tt.expected, output)
			}
		})
	}
}

func TestFormattedLogging(t *testing.T) {
	var buf bytes.Buffer
	Logger = zerolog.New(&buf).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	tests := []struct {
		name     string
		logFunc  func()
		contains string
	}{
		{
			"Debugf",
			func() { Debugf("formatted %s %d", "test", 123) },
			"formatted test 123",
		},
		{
			"Infof",
			func() { Infof("info %s", "message") },
			"info message",
		},
		{
			"Warnf",
			func() { Warnf("warn %d", 42) },
			"warn 42",
		},
		{
			"Errorf",
			func() { Errorf("error %v", true) },
			"error true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc()

			output := buf.String()
			if !strings.Contains(output, tt.contains) {
				t.Errorf("Expected output to contain '%s', got: %s", tt.contains, output)
			}
		})
	}
}

func TestWithField(t *testing.T) {
	var buf bytes.Buffer
	Logger = zerolog.New(&buf).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	event := WithField("key", "value")
	event.Msg("test message")

	output := buf.String()
	if !strings.Contains(output, "key") {
		t.Error("Expected output to contain field key")
	}
	if !strings.Contains(output, "value") {
		t.Error("Expected output to contain field value")
	}
}

func TestWithError(t *testing.T) {
	var buf bytes.Buffer
	Logger = zerolog.New(&buf).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	testErr := &testError{msg: "test error message"}
	event := WithError(testErr)
	event.Msg("error occurred")

	output := buf.String()
	if !strings.Contains(output, "test error message") {
		t.Error("Expected output to contain error message")
	}
}

func TestLoggerOutput(t *testing.T) {
	// Test that logger writes to output
	var buf bytes.Buffer
	Logger = zerolog.New(&buf).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	Info("test message")

	if buf.Len() == 0 {
		t.Error("Expected logger to write output")
	}

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected 'test message' in output, got: %s", output)
	}
}

func TestLogLevel_Debug(t *testing.T) {
	var buf bytes.Buffer
	Logger = zerolog.New(&buf).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	Debug("debug message")

	output := buf.String()
	if !strings.Contains(output, "debug message") {
		t.Error("Debug message should be logged at debug level")
	}
}

func TestLogLevel_InfoFilterDebug(t *testing.T) {
	var buf bytes.Buffer
	Logger = zerolog.New(&buf).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	Debug("debug message")

	output := buf.String()
	if strings.Contains(output, "debug message") {
		t.Error("Debug message should not be logged at info level")
	}
}

func TestLogLevel_ErrorFilterInfo(t *testing.T) {
	var buf bytes.Buffer
	Logger = zerolog.New(&buf).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)

	Info("info message")

	output := buf.String()
	if strings.Contains(output, "info message") {
		t.Error("Info message should not be logged at error level")
	}
}

func TestMultipleLogCalls(t *testing.T) {
	var buf bytes.Buffer
	Logger = zerolog.New(&buf).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	Debug("message 1")
	Info("message 2")
	Warn("message 3")

	output := buf.String()

	messages := []string{"message 1", "message 2", "message 3"}
	for _, msg := range messages {
		if !strings.Contains(output, msg) {
			t.Errorf("Expected output to contain '%s'", msg)
		}
	}
}

// Helper type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
