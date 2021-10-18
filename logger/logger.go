package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	bootstrapLogging()
}

// Logger is a logger abstraction meant to not be tied to an specific implementation
type Logger struct {
	log *zap.SugaredLogger
}

var log *Logger

// Debug logs a message in debug level
func (l *Logger) Debug(args ...interface{}) {
	l.log.Debug(args...)
}

// Debugf logs a formatted message in debug level
func (l *Logger) Debugf(template string, args ...interface{}) {
	l.log.Debugf(template, args...)
}

// Info logs a message in info level
func (l *Logger) Info(args ...interface{}) {
	l.log.Info(args...)
}

// Infof logs a formatted message in info level
func (l *Logger) Infof(template string, args ...interface{}) {
	l.log.Infof(template, args...)
}

// Warn logs a message in warn level
func (l *Logger) Warn(args ...interface{}) {
	l.log.Warn(args...)
}

// Warnf logs a formatted message in warn level
func (l *Logger) Warnf(template string, args ...interface{}) {
	l.log.Warnf(template, args...)
}

// Error logs a message in error level
func (l *Logger) Error(args ...interface{}) {
	l.log.Error(args...)
}

// Errorf logs a formatted message in error level
func (l *Logger) Errorf(template string, args ...interface{}) {
	l.log.Errorf(template, args...)
}

// bootstrapLogging bootstraps a basic logger
func bootstrapLogging() {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.CallerKey = ""
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	zapLog, err := config.Build()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(zapLog)
	log = &Logger{zap.S()}
}

// Log returns a global logger instance
func Log() *Logger {
	return log
}
