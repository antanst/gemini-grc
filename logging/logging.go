// Package logging provides a simple, structured logging interface using slog.
// It offers colored output for better readability in terminal environments.
package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
)

// Global logger instance.
var slogLogger *slog.Logger

// Current log level - used to filter logs.
var currentLogLevel = slog.LevelInfo

// ANSI color codes for terminal output.
const (
	colorReset = "\033[0m"
	levelDebug = "\033[37m" // Gray
	levelInfo  = "\033[32m" // Green
	levelWarn  = "\033[33m" // Yellow
	levelError = "\033[31m" // Red
)

// Standard helper functions for logging
func LogDebug(format string, args ...interface{}) {
	if slogLogger != nil {
		slogLogger.Debug(fmt.Sprintf(format, args...))
	}
}

func LogInfo(format string, args ...interface{}) {
	if slogLogger != nil {
		slogLogger.Info(fmt.Sprintf(format, args...))
	}
}

func LogWarn(format string, args ...interface{}) {
	if slogLogger != nil {
		slogLogger.Warn(fmt.Sprintf(format, args...))
	}
}

func LogError(format string, args ...interface{}) {
	if slogLogger != nil {
		msg := fmt.Sprintf(format, args...)
		slogLogger.Error(msg, slog.String("error", msg))
	}
}

// InitSlogger initializes the slog logger with custom handler.
func InitSlogger(level slog.Level) {
	// Set the global log level
	currentLogLevel = level

	// Create the handler with color support
	baseHandler := NewColorHandler(os.Stderr)

	// Create and set the new logger
	slogLogger = slog.New(baseHandler)

	// Set as default logger
	slog.SetDefault(slogLogger)

	// Print a startup message to verify logging is working
	slogLogger.Info("Slog initialized", "level", level.String())
}

// GetSlogger returns the current global slog logger instance.
// Can be used by other packages
func GetSlogger() *slog.Logger {
	if slogLogger == nil {
		return slog.Default()
	}
	return slogLogger
}

// ColorHandler formats logs with colors for better terminal readability
type ColorHandler struct {
	out   io.Writer
	mu    *sync.Mutex
	attrs []slog.Attr // Store attributes for this handler
}

// NewColorHandler creates a new handler that writes colored logs to the provided writer
func NewColorHandler(w io.Writer) *ColorHandler {
	if w == nil {
		w = os.Stderr
	}
	return &ColorHandler{
		out:   w,
		mu:    &sync.Mutex{},
		attrs: make([]slog.Attr, 0),
	}
}

// Enabled checks if the given log level is enabled
func (h *ColorHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= currentLogLevel
}

// Handle processes a log record, formatting it with colors
func (h *ColorHandler) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Format time
	timeStr := fmt.Sprintf("[%s]", r.Time.Format("2006-01-02 15:04:05"))

	// Format level
	var levelStr string
	switch r.Level {
	case slog.LevelDebug:
		levelStr = fmt.Sprintf("%sDEBUG%s", levelDebug, colorReset)
	case slog.LevelInfo:
		levelStr = fmt.Sprintf("%sINFO%s", levelInfo, colorReset)
	case slog.LevelWarn:
		levelStr = fmt.Sprintf("%sWARN%s", levelWarn, colorReset)
	case slog.LevelError:
		levelStr = fmt.Sprintf("%sERROR%s", levelError, colorReset)
	default:
		levelStr = r.Level.String()
	}

	// Build prefix
	prefix := fmt.Sprintf("%s %s ", timeStr, levelStr)

	// Format message - we'll collect any special fields separately
	attrMap := make(map[string]string)

	// First collect attributes from the handler itself
	for _, attr := range h.attrs {
		attrMap[attr.Key] = attr.Value.String()
	}

	// Then extract from record attributes, which might override handler attributes
	r.Attrs(func(a slog.Attr) bool {
		attrMap[a.Key] = a.Value.String()
		return true
	})

	// Format message with attributes on the same line
	msg := fmt.Sprintf("%s%s", prefix, r.Message)

	// Add attributes to the same line if present
	if len(attrMap) > 0 {
		// Add a space after the message
		msg += " "

		// Build attribute string
		attrs := make([]string, 0, len(attrMap))
		for k, v := range attrMap {
			attrs = append(attrs, fmt.Sprintf("%s=%s", k, v))
		}

		// Join all attributes with spaces
		msg += strings.Join(attrs, " ")
	}

	// Add newline at the end
	msg += "\n"

	// Write to output
	_, err := io.WriteString(h.out, msg)
	return err
}

// WithGroup returns a new Handler that inherits from this Handler
func (h *ColorHandler) WithGroup(name string) slog.Handler {
	return h // For simplicity, we don't support groups
}

// WithAttrs returns a new Handler whose attributes include both
// the receiver's attributes and the arguments
func (h *ColorHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// Create a new handler with the same output but additional attributes
	newHandler := &ColorHandler{
		out:   h.out,
		mu:    h.mu,
		attrs: append(append([]slog.Attr{}, h.attrs...), attrs...),
	}
	return newHandler
}
