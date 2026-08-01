package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

type compositeLogWriter struct {
	consoleWriter io.Writer
	fileWriter    io.Writer
	kafkaWriter   *kafka.Writer
}

func (w *compositeLogWriter) Write(p []byte) (n int, err error) {
	// Write to console
	if w.consoleWriter != nil {
		_, _ = w.consoleWriter.Write(p)
	}

	// Write to file
	if w.fileWriter != nil {
		_, _ = w.fileWriter.Write(p)
	}

	// Write to Kafka asynchronously
	if w.kafkaWriter != nil {
		msg := kafka.Message{
			Value: p,
		}
		_ = w.kafkaWriter.WriteMessages(context.Background(), msg)
	}

	return len(p), nil
}

type dailyFileWriter struct {
	mu          sync.Mutex
	logDir      string
	serviceName string
	currentDate string
	file        *os.File
}

func (w *dailyFileWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now().Format("2006-01-02")
	if w.file == nil || now != w.currentDate {
		if w.file != nil {
			_ = w.file.Close()
		}
		w.currentDate = now
		logFilePath := filepath.Join(w.logDir, fmt.Sprintf("%s-%s.log", w.serviceName, w.currentDate))
		
		// Ensure logDir exists
		_ = os.MkdirAll(w.logDir, 0755)

		f, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open daily log file %s: %v\n", logFilePath, err)
			return 0, err
		}
		w.file = f
	}

	return w.file.Write(p)
}

func getLogLevel(serviceName string) slog.Level {
	svcUpper := strings.ToUpper(serviceName)
	candidates := []string{
		svcUpper + "_LOG_LEVEL",
		"LOG_LEVEL_" + svcUpper,
		svcUpper + "_ERROR_LEVEL",
		"ERROR_LEVEL_" + svcUpper,
		"LOG_LEVEL",
		"ERROR_LEVEL",
	}

	var levelStr string
	for _, env := range candidates {
		if val := os.Getenv(env); val != "" {
			levelStr = val
			break
		}
	}

	if levelStr == "" {
		return slog.LevelInfo // Default level
	}

	switch strings.ToLower(strings.TrimSpace(levelStr)) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func isFileLogEnabled(serviceName string) bool {
	svcUpper := strings.ToUpper(serviceName)
	candidates := []string{
		svcUpper + "_FILE_LOG_ENABLED",
		"FILE_LOG_ENABLED_" + svcUpper,
		"FILE_LOG_ENABLED",
		"LOG_FILE_ENABLED",
	}

	for _, env := range candidates {
		if val := os.Getenv(env); val != "" {
			return strings.ToLower(strings.TrimSpace(val)) == "true"
		}
	}
	return false // Default is false (disabled)
}

// InitLogger initializes standard slog logger.
func InitLogger(serviceName string, brokers []string) *slog.Logger {
	var consoleWriter io.Writer = os.Stdout
	var fileWriter io.Writer
	var kafkaWriter *kafka.Writer

	// Setup file logger if enabled
	if isFileLogEnabled(serviceName) {
		logDir := "/logs"
		if envLogDir := os.Getenv("LOG_DIR"); envLogDir != "" {
			logDir = envLogDir
		}
		fileWriter = &dailyFileWriter{
			logDir:      logDir,
			serviceName: serviceName,
		}
	}

	// Setup Kafka writer
	if len(brokers) > 0 && brokers[0] != "" {
		kafkaWriter = &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        "platform-logs",
			Async:        true,
			RequiredAcks: kafka.RequireNone, // Fire-and-forget for log performance
		}
	}

	writer := &compositeLogWriter{
		consoleWriter: consoleWriter,
		fileWriter:    fileWriter,
		kafkaWriter:   kafkaWriter,
	}

	level := getLogLevel(serviceName)
	logger := slog.New(slog.NewJSONHandler(writer, &slog.HandlerOptions{
		Level: level,
	})).With(slog.String("service", serviceName))

	slog.SetDefault(logger)
	return logger
}
