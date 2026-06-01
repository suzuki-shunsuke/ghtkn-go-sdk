package log_test

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/log"
)

func TestNewLogger(t *testing.T) {
	logger := log.NewLogger()
	if logger == nil {
		t.Fatal("NewLogger() returned nil")
	}

	if logger.Expire == nil {
		t.Error("Expire function is nil")
	}
	if logger.FailedToOpenBrowser == nil {
		t.Error("FailedToOpenBrowser function is nil")
	}
	if logger.AccessTokenIsNotFoundInBackend == nil {
		t.Error("AccessTokenIsNotFoundInBackend function is nil")
	}
}

func TestLogger_Expire(t *testing.T) {
	var buf bytes.Buffer
	slogger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	logger := log.NewLogger()
	expirationDate := time.Date(2024, time.December, 25, 12, 0, 0, 0, time.UTC)

	logger.Expire(slogger, expirationDate)

	output := buf.String()
	if !strings.Contains(output, "access token expires") {
		t.Errorf("Expected log to contain 'access token expires', got: %s", output)
	}
	if !strings.Contains(output, "2024-12-25T12:00:00Z") {
		t.Errorf("Expected log to contain formatted expiration date, got: %s", output)
	}
}

func TestLogger_FailedToOpenBrowser(t *testing.T) {
	var buf bytes.Buffer
	slogger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	logger := log.NewLogger()
	testErr := errors.New("browser not found")

	logger.FailedToOpenBrowser(slogger, testErr)

	output := buf.String()
	if !strings.Contains(output, "failed to open the browser") {
		t.Errorf("Expected log to contain 'failed to open the browser', got: %s", output)
	}
	if !strings.Contains(output, "browser not found") {
		t.Errorf("Expected log to contain error message, got: %s", output)
	}
}

func TestLogger_AccessTokenIsNotFoundInBackend(t *testing.T) {
	var buf bytes.Buffer
	slogger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	logger := log.NewLogger()

	logger.AccessTokenIsNotFoundInBackend(slogger)

	output := buf.String()
	if !strings.Contains(output, "access token is not found in backend") {
		t.Errorf("Expected log to contain 'access token is not found in backend', got: %s", output)
	}
}
