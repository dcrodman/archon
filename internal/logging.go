package internal

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.SugaredLogger

// InitLogger initializes the global logger, which is used (or derived) by all sub-servers.
func InitLogger() error {
	logLvl, err := zapcore.ParseLevel(Config.Logging.LogLevel)
	if err != nil {
		return fmt.Errorf("parsing log level: %w", err)
	}

	logConfig := zap.NewDevelopmentConfig()
	logConfig.Level = zap.NewAtomicLevelAt(logLvl)
	if Config.Logging.LogFilePath != "" {
		logConfig.OutputPaths = []string{Config.Logging.LogFilePath}
	}
	logConfig.DisableCaller = !Config.Logging.IncludeCaller

	logConfig.EncoderConfig = zap.NewDevelopmentEncoderConfig()
	logConfig.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
	logConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	logger, err := logConfig.Build()
	if err != nil {
		return fmt.Errorf("building logger: %w", err)
	}

	Logger = logger.Sugar()
	return nil
}
