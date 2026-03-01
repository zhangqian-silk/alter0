package observability

import "alter0/app/pkg/logger"

func Init(logDir string) error {
	return logger.Init(logDir)
}

func Info(format string, args ...interface{}) {
	logger.Info(format, args...)
}

func Error(format string, args ...interface{}) {
	logger.Error(format, args...)
}
