package logger

import (
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger *zap.Logger
	level  zap.AtomicLevel
)

func init() {
	var err error

	level = zap.NewAtomicLevel()
	config := zap.NewProductionConfig()

	config.Level = level
	config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.RFC3339)
	config.EncoderConfig.StacktraceKey = ""

	logger, err = config.Build(zap.WithCaller(false))
	if err != nil {
		panic(err)
	}

	logger.Level()
}

func GetLogger() *zap.Logger {
	return logger
}

func Info(message string, fields ...zap.Field) {
	logger.Info(message, fields...)
}

func Debug(message string, fields ...zap.Field) {
	logger.Debug(message, fields...)
}

func Error(message string, fields ...zap.Field) {
	logger.Error(message, fields...)
}

func Fatal(message string, fields ...zap.Field) {
	logger.Fatal(message, fields...)
}

func SetLevel(l zapcore.Level) {
	level.SetLevel(l)
}

func LevelFlag() *zapcore.Level {
	return zap.LevelFlag("log-level", zapcore.InfoLevel, "Set log level")
}
