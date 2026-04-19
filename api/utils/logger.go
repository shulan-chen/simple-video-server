package utils

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

// InitLogging 初始化日志
// 日志等级可通过环境变量 LOG_LEVEL 设置：debug, info, warn, error
// 默认为 debug
func InitLogging() {
	var err error

	// 从环境变量读取日志等级
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "debug" // 默认等级
	}

	// 解析日志等级
	var level zapcore.Level
	switch logLevel {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn", "warning":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	// 创建配置
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(level)

	// 构建 Logger
	Logger, err = config.Build()
	if err != nil {
		panic(err)
	}
}
