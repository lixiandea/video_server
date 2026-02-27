package logging

import (
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.Logger

// InitLogger initializes the structured logger
func InitLogger(level string, outputPath string) error {
	// Define log level
	logLevel := zapcore.InfoLevel
	switch level {
	case "debug":
		logLevel = zapcore.DebugLevel
	case "warn":
		logLevel = zapcore.WarnLevel
	case "error":
		logLevel = zapcore.ErrorLevel
	}

	// Configure encoder
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Set output path
	outputPaths := []string{outputPath}
	if outputPath == "stdout" {
		outputPaths = []string{"stdout"}
	}

	// Create logger config
	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(logLevel),
		Development:      false,
		Encoding:         "json",
		EncoderConfig:    encoderConfig,
		OutputPaths:      outputPaths,
		ErrorOutputPaths: []string{"stderr"},
	}

	var err error
	logger, err = config.Build()
	if err != nil {
		return err
	}

	// Add caller skip to properly show the calling function
	logger = logger.WithOptions(zap.AddCallerSkip(1))
	
	return nil
}

// GetLogger returns the global logger instance
func GetLogger() *zap.Logger {
	if logger == nil {
		// Initialize with default settings if not initialized
		InitLogger("info", "stdout")
	}
	return logger
}

// Sync flushes any buffered log entries
func Sync() error {
	if logger != nil {
		return logger.Sync()
	}
	return nil
}

// WithRequestID 添加请求 ID 到日志
func WithRequestID(requestID string) zap.Field {
	return zap.String("request_id", requestID)
}

// Helper functions for common log operations
func Info(msg string, fields ...zap.Field) {
	GetLogger().Info(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	GetLogger().Error(msg, fields...)
}

func Debug(msg string, fields ...zap.Field) {
	GetLogger().Debug(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	GetLogger().Warn(msg, fields...)
}

// WithFields adds context fields to the logger
func WithFields(fields ...zap.Field) *zap.Logger {
	return GetLogger().With(fields...)
}

// Request logging helpers
func LogRequest(method, path string, statusCode int, duration time.Duration, userID interface{}) {
	fields := []zap.Field{
		zap.String("method", method),
		zap.String("path", path),
		zap.Int("status_code", statusCode),
		zap.Duration("duration", duration),
		zap.Time("timestamp", time.Now()),
	}
	
	if userID != nil {
		fields = append(fields, zap.Any("user_id", userID))
	}
	
	if statusCode >= 500 {
		Error("HTTP request failed", fields...)
	} else if statusCode >= 400 {
		Warn("HTTP request warning", fields...)
	} else {
		Info("HTTP request processed", fields...)
	}
}

// Database operation logging
func LogDBOperation(operation string, duration time.Duration, err error) {
	fields := []zap.Field{
		zap.String("operation", operation),
		zap.Duration("duration", duration),
		zap.Time("timestamp", time.Now()),
	}
	
	if err != nil {
		fields = append(fields, zap.Error(err))
		Error("Database operation failed", fields...)
	} else {
		Info("Database operation completed", fields...)
	}
}

// Business operation logging
func LogBusinessOperation(operation string, userID uint, details map[string]interface{}) {
	fields := []zap.Field{
		zap.String("operation", operation),
		zap.Uint("user_id", userID),
		zap.Time("timestamp", time.Now()),
	}
	
	for key, value := range details {
		fields = append(fields, zap.Any(key, value))
	}
	
	Info("Business operation completed", fields...)
}