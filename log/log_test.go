package log_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/platforma-dev/platforma/log"
)

func TestNew_TextLogger(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := log.New(&buf, "text", slog.LevelInfo, nil)

	if logger == nil {
		t.Fatal("expected non-nil logger")
	}

	logger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("expected output to contain 'test message', got: %s", output)
	}
}

func TestNew_JSONLogger(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := log.New(&buf, "json", slog.LevelInfo, nil)

	if logger == nil {
		t.Fatal("expected non-nil logger")
	}

	logger.Info("json test")

	output := buf.String()
	if !strings.Contains(output, `"msg":"json test"`) {
		t.Errorf("expected JSON output, got: %s", output)
	}
}

func TestNew_InvalidTypeDefaultsToText(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := log.New(&buf, "invalid", slog.LevelInfo, nil)

	if logger == nil {
		t.Fatal("expected non-nil logger")
	}

	logger.Info("test")

	output := buf.String()
	// Text format should not have JSON structure
	if strings.Contains(output, `{"msg"`) {
		t.Error("expected text format, got JSON")
	}
}

func TestNew_LogLevel(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		level    slog.Level
		logDebug bool
		logInfo  bool
		logWarn  bool
		logError bool
	}{
		{
			name:     "Debug level logs all",
			level:    slog.LevelDebug,
			logDebug: true,
			logInfo:  true,
			logWarn:  true,
			logError: true,
		},
		{
			name:     "Info level skips debug",
			level:    slog.LevelInfo,
			logDebug: false,
			logInfo:  true,
			logWarn:  true,
			logError: true,
		},
		{
			name:     "Warn level skips debug and info",
			level:    slog.LevelWarn,
			logDebug: false,
			logInfo:  false,
			logWarn:  true,
			logError: true,
		},
		{
			name:     "Error level only logs errors",
			level:    slog.LevelError,
			logDebug: false,
			logInfo:  false,
			logWarn:  false,
			logError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			logger := log.New(&buf, "text", tc.level, nil)

			logger.Debug("debug message")
			logger.Info("info message")
			logger.Warn("warn message")
			logger.Error("error message")

			output := buf.String()

			if tc.logDebug != strings.Contains(output, "debug message") {
				t.Errorf("debug logging mismatch: expected %v, got %v", tc.logDebug, strings.Contains(output, "debug message"))
			}
			if tc.logInfo != strings.Contains(output, "info message") {
				t.Errorf("info logging mismatch: expected %v, got %v", tc.logInfo, strings.Contains(output, "info message"))
			}
			if tc.logWarn != strings.Contains(output, "warn message") {
				t.Errorf("warn logging mismatch: expected %v, got %v", tc.logWarn, strings.Contains(output, "warn message"))
			}
			if tc.logError != strings.Contains(output, "error message") {
				t.Errorf("error logging mismatch: expected %v, got %v", tc.logError, strings.Contains(output, "error message"))
			}
		})
	}
}

func TestContextKeys(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		contextKey  interface{}
		contextVal  string
		expectedKey string
	}{
		{"DomainName", log.DomainNameKey, "user-domain", "domainName"},
		{"TraceID", log.TraceIDKey, "trace-123", "traceId"},
		{"ServiceName", log.ServiceNameKey, "web-service", "serviceName"},
		{"StartupTask", log.StartupTaskKey, "init-db", "startupTask"},
		{"UserID", log.UserIDKey, "user-456", "userId"},
		{"WorkerID", log.WorkerIDKey, "worker-789", "workerId"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			logger := log.New(&buf, "text", slog.LevelInfo, nil)

			ctx := context.WithValue(context.Background(), tc.contextKey, tc.contextVal)

			logger.InfoContext(ctx, "test message")

			output := buf.String()
			if !strings.Contains(output, tc.expectedKey+"="+tc.contextVal) {
				t.Errorf("expected output to contain %s=%s, got: %s", tc.expectedKey, tc.contextVal, output)
			}
		})
	}
}

func TestContextHandler_CustomKeys(t *testing.T) {
	t.Parallel()

	customKey := "customKey"
	customContextKey := struct{ name string }{name: "custom"}

	var buf bytes.Buffer
	logger := log.New(&buf, "text", slog.LevelInfo, map[string]any{
		customKey: customContextKey,
	})

	ctx := context.WithValue(context.Background(), customContextKey, "custom-value")

	logger.InfoContext(ctx, "test message")

	output := buf.String()
	if !strings.Contains(output, "customKey=custom-value") {
		t.Errorf("expected custom key in output, got: %s", output)
	}
}

func TestSetDefault(t *testing.T) {
	t.Parallel()

	// Note: SetDefault modifies global state, so we need to be careful
	// We'll test by creating a custom logger and verifying it's used
	var buf bytes.Buffer
	customLogger := log.New(&buf, "text", slog.LevelInfo, nil)

	originalLogger := log.Logger
	defer func() { log.SetDefault(originalLogger) }()

	log.SetDefault(customLogger)

	if log.Logger != customLogger {
		t.Error("expected Logger to be set to custom logger")
	}
}

func TestPackageLevelFunctions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		logFunc  func(string, ...any)
		expected string
	}{
		{"Debug", log.Debug, "level=DEBUG"},
		{"Info", log.Info, "level=INFO"},
		{"Warn", log.Warn, "level=WARN"},
		{"Error", log.Error, "level=ERROR"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Can't fully test without capturing stdout or modifying global logger
			// This just verifies the functions don't panic
			tc.logFunc("test message")
		})
	}
}

func TestPackageLevelContextFunctions(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	testCases := []struct {
		name    string
		logFunc func(context.Context, string, ...any)
	}{
		{"DebugContext", log.DebugContext},
		{"InfoContext", log.InfoContext},
		{"WarnContext", log.WarnContext},
		{"ErrorContext", log.ErrorContext},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Verify functions don't panic
			tc.logFunc(ctx, "test message")
		})
	}
}

func TestContextHandler_LogWithAttributes(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := log.New(&buf, "text", slog.LevelInfo, nil)

	logger.Info("test message", "key1", "value1", "key2", 42)

	output := buf.String()
	if !strings.Contains(output, "key1=value1") {
		t.Errorf("expected key1=value1 in output, got: %s", output)
	}
	if !strings.Contains(output, "key2=42") {
		t.Errorf("expected key2=42 in output, got: %s", output)
	}
}

func TestContextHandler_MultipleContextValues(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := log.New(&buf, "text", slog.LevelInfo, nil)

	ctx := context.Background()
	ctx = context.WithValue(ctx, log.TraceIDKey, "trace-123")
	ctx = context.WithValue(ctx, log.ServiceNameKey, "api-service")
	ctx = context.WithValue(ctx, log.UserIDKey, "user-456")

	logger.InfoContext(ctx, "multi-context test")

	output := buf.String()
	if !strings.Contains(output, "traceId=trace-123") {
		t.Error("expected traceId in output")
	}
	if !strings.Contains(output, "serviceName=api-service") {
		t.Error("expected serviceName in output")
	}
	if !strings.Contains(output, "userId=user-456") {
		t.Error("expected userId in output")
	}
}

func TestContextHandler_NonStringContextValue(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := log.New(&buf, "text", slog.LevelInfo, nil)

	// Context value is not a string - should be ignored
	ctx := context.WithValue(context.Background(), log.TraceIDKey, 12345)

	logger.InfoContext(ctx, "test message")

	output := buf.String()
	// Non-string value should be ignored
	if strings.Contains(output, "traceId=12345") {
		t.Error("expected non-string context value to be ignored")
	}
}

func TestJSONLogger_Format(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := log.New(&buf, "json", slog.LevelInfo, nil)

	ctx := context.WithValue(context.Background(), log.TraceIDKey, "trace-json")

	logger.InfoContext(ctx, "json format test", "customKey", "customValue")

	output := buf.String()

	// Verify JSON structure
	if !strings.Contains(output, `"msg":"json format test"`) {
		t.Error("expected msg field in JSON")
	}
	if !strings.Contains(output, `"traceId":"trace-json"`) {
		t.Error("expected traceId field in JSON")
	}
	if !strings.Contains(output, `"customKey":"customValue"`) {
		t.Error("expected customKey field in JSON")
	}
}

func TestLogger_ConcurrentWrites(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := log.New(&buf, "text", slog.LevelInfo, nil)

	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func(id int) {
			logger.Info("concurrent message", "id", id)
			done <- true
		}(i)
	}

	for i := 0; i < 100; i++ {
		<-done
	}

	output := buf.String()
	if !strings.Contains(output, "concurrent message") {
		t.Error("expected concurrent messages in output")
	}
}

func TestContextHandler_EmptyContext(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := log.New(&buf, "text", slog.LevelInfo, nil)

	// Use empty context - should not cause errors
	logger.InfoContext(context.Background(), "empty context test")

	output := buf.String()
	if !strings.Contains(output, "empty context test") {
		t.Errorf("expected message in output, got: %s", output)
	}
}

func TestContextHandler_NilCustomKeys(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := log.New(&buf, "text", slog.LevelInfo, nil)

	logger.Info("test with nil custom keys")

	output := buf.String()
	if !strings.Contains(output, "test with nil custom keys") {
		t.Error("expected message in output")
	}
}

func TestNew_WithCustomKeysAndContext(t *testing.T) {
	t.Parallel()

	customKey := "requestId"
	customContextKey := struct{ name string }{name: "request"}

	var buf bytes.Buffer
	logger := log.New(&buf, "text", slog.LevelInfo, map[string]any{
		customKey: customContextKey,
	})

	ctx := context.Background()
	ctx = context.WithValue(ctx, customContextKey, "req-999")
	ctx = context.WithValue(ctx, log.TraceIDKey, "trace-999")

	logger.InfoContext(ctx, "custom and default keys")

	output := buf.String()

	if !strings.Contains(output, "requestId=req-999") {
		t.Error("expected custom key in output")
	}
	if !strings.Contains(output, "traceId=trace-999") {
		t.Error("expected default key in output")
	}
}

func TestLogger_DifferentLevelsInJSON(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := log.New(&buf, "json", slog.LevelDebug, nil)

	logger.Debug("debug msg")
	logger.Info("info msg")
	logger.Warn("warn msg")
	logger.Error("error msg")

	output := buf.String()

	if !strings.Contains(output, `"level":"DEBUG"`) {
		t.Error("expected DEBUG level in JSON")
	}
	if !strings.Contains(output, `"level":"INFO"`) {
		t.Error("expected INFO level in JSON")
	}
	if !strings.Contains(output, `"level":"WARN"`) {
		t.Error("expected WARN level in JSON")
	}
	if !strings.Contains(output, `"level":"ERROR"`) {
		t.Error("expected ERROR level in JSON")
	}
}

func TestContextKey_TypeSafety(t *testing.T) {
	t.Parallel()

	// Verify context keys exist and can be used
	// We can't verify the exact type since it's unexported, but we can verify they work
	ctx := context.Background()
	ctx = context.WithValue(ctx, log.DomainNameKey, "test")
	ctx = context.WithValue(ctx, log.TraceIDKey, "test")
	ctx = context.WithValue(ctx, log.ServiceNameKey, "test")
	ctx = context.WithValue(ctx, log.StartupTaskKey, "test")
	ctx = context.WithValue(ctx, log.UserIDKey, "test")
	ctx = context.WithValue(ctx, log.WorkerIDKey, "test")

	// If we got here without panic, the keys work
	if ctx == nil {
		t.Error("context should not be nil")
	}
}

func TestLogger_WithComplexAttributes(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := log.New(&buf, "json", slog.LevelInfo, nil)

	logger.Info("complex attributes",
		"string", "value",
		"int", 42,
		"bool", true,
		"float", 3.14,
	)

	output := buf.String()

	if !strings.Contains(output, `"string":"value"`) {
		t.Error("expected string attribute")
	}
	if !strings.Contains(output, `"int":42`) {
		t.Error("expected int attribute")
	}
	if !strings.Contains(output, `"bool":true`) {
		t.Error("expected bool attribute")
	}
	if !strings.Contains(output, `"float":3.14`) {
		t.Error("expected float attribute")
	}
}

func TestContextHandler_PreservesAllContextKeys(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := log.New(&buf, "text", slog.LevelInfo, nil)

	ctx := context.Background()
	ctx = context.WithValue(ctx, log.DomainNameKey, "domain1")
	ctx = context.WithValue(ctx, log.TraceIDKey, "trace1")
	ctx = context.WithValue(ctx, log.ServiceNameKey, "service1")
	ctx = context.WithValue(ctx, log.StartupTaskKey, "task1")
	ctx = context.WithValue(ctx, log.UserIDKey, "user1")
	ctx = context.WithValue(ctx, log.WorkerIDKey, "worker1")

	logger.InfoContext(ctx, "all keys test")

	output := buf.String()

	expectedKeys := []string{
		"domainName=domain1",
		"traceId=trace1",
		"serviceName=service1",
		"startupTask=task1",
		"userId=user1",
		"workerId=worker1",
	}

	for _, expected := range expectedKeys {
		if !strings.Contains(output, expected) {
			t.Errorf("expected %q in output, got: %s", expected, output)
		}
	}
}