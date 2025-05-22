package contextlog

import (
	"context"
	"fmt"
	"log/slog"

	"gemini-grc/contextutil"
)

// SlogEventWithContext adds context information as structured fields to the log event.
func SlogEventWithContext(ctx context.Context, logger *slog.Logger) *slog.Logger {
	// Start with the provided logger
	if logger == nil {
		// If logger isn't initialized, use the default logger
		return slog.Default()
	}

	// Get context values - will be added directly to log records
	host := contextutil.GetHostFromContext(ctx)
	requestID := contextutil.GetRequestIDFromContext(ctx)
	component := contextutil.GetComponentFromContext(ctx)
	workerID := contextutil.GetWorkerIDFromContext(ctx)
	url := contextutil.GetURLFromContext(ctx)

	// Add all context fields to the logger
	if host != "" {
		logger = logger.With("host", host)
	}

	if requestID != "" {
		logger = logger.With("request_id", requestID)
	}

	if workerID >= 0 {
		logger = logger.With("worker_id", workerID)
	}

	if component != "" {
		logger = logger.With("component", component)
	}

	if url != "" {
		logger = logger.With("url", url)
	}

	return logger
}

// LogDebugWithContext logs a debug message with context information.
func LogDebugWithContext(ctx context.Context, logger *slog.Logger, format string, args ...interface{}) {
	if logger == nil {
		return
	}

	// Create logger with context fields
	contextLogger := SlogEventWithContext(ctx, logger)

	// Format the message
	message := fmt.Sprintf(format, args...)

	// Log with context data in the record attributes
	contextLogger.Debug(message)
}

// LogInfoWithContext logs an info message with context information.
func LogInfoWithContext(ctx context.Context, logger *slog.Logger, format string, args ...interface{}) {
	if logger == nil {
		return
	}

	// Create logger with context fields
	contextLogger := SlogEventWithContext(ctx, logger)

	// Format the message
	message := fmt.Sprintf(format, args...)

	// Log with context data in the record attributes
	contextLogger.Info(message)
}

// LogWarnWithContext logs a warning message with context information.
func LogWarnWithContext(ctx context.Context, logger *slog.Logger, format string, args ...interface{}) {
	if logger == nil {
		return
	}

	// Create logger with context fields
	contextLogger := SlogEventWithContext(ctx, logger)

	// Format the message
	message := fmt.Sprintf(format, args...)

	// Log with context data in the record attributes
	contextLogger.Warn(message)
}

// LogErrorWithContext logs an error message with context information
func LogErrorWithContext(ctx context.Context, logger *slog.Logger, format string, args ...interface{}) {
	if logger == nil {
		return
	}

	// Create logger with context fields
	contextLogger := SlogEventWithContext(ctx, logger)

	// Format the message
	msg := fmt.Sprintf(format, args...)

	// Log with context data in the record attributes
	contextLogger.Error(msg, slog.String("error", msg))
}
