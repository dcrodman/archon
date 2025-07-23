package core

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger returns a logger intended to be used for general application logs.
func NewLogger(cfg *Config) (*zap.SugaredLogger, error) {
	logLvl, err := zapcore.ParseLevel(cfg.Logging.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("parsing log level: %w", err)
	}

	logConfig := zap.NewDevelopmentConfig()
	logConfig.Level = zap.NewAtomicLevelAt(logLvl)
	if cfg.Logging.LogFilePath != "" {
		logConfig.OutputPaths = []string{cfg.Logging.LogFilePath}
	}
	logConfig.DisableCaller = !cfg.Logging.IncludeCaller

	logConfig.EncoderConfig = zap.NewDevelopmentEncoderConfig()
	logConfig.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
	logConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	logger, err := logConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("building logger: %w", err)
	}

	return logger.Sugar(), nil
}
