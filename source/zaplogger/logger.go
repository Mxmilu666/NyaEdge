package zaplogger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var logger *zap.Logger

func Setup() (*zap.Logger, error) {
	fileEncoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	fileSync := zapcore.AddSync(&lumberjack.Logger{
		Filename:   "./log/" + "nyaedge-center.log",
		MaxSize:    10,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   true,
	})

	fileCore := zapcore.NewCore(fileEncoder, fileSync, zap.NewAtomicLevelAt(zapcore.DebugLevel))

	consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	consoleSync := zapcore.AddSync(zapcore.Lock(os.Stdout))

	consoleCore := zapcore.NewCore(consoleEncoder, consoleSync, zap.NewAtomicLevelAt(zapcore.DebugLevel))

	logger = zap.New(zapcore.NewTee(fileCore, consoleCore))

	defer logger.Sync()

	return logger, nil
}

func Getlogger() *zap.Logger {
	return logger
}
