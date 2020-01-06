package logger

import (
	"sync"

	"go.uber.org/zap"
)

var instance *zap.SugaredLogger
var once sync.Once

// Instance returns a sugarred logger instance, initializing it if not already done
func Instance() *zap.SugaredLogger {
	once.Do(func() {
		logger, _ := zap.NewProduction()
		instance = logger.Sugar()
	})
	return instance
}
