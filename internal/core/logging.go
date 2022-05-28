package core

import (
	"fmt"
	"io"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func NewLogger(cfg *Config) (*logrus.Logger, error) {
	var w io.Writer
	var err error

	if cfg.LogFilePath == "" {
		w = os.Stdout
	} else {
		w, err = os.OpenFile(cfg.LogFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("error opening log file: %w", err)
		}
	}

	logLvl, err := logrus.ParseLevel(viper.GetString("log_level"))
	if err != nil {
		fmt.Println("error parsing Log level: " + err.Error())
	}

	logger := &logrus.Logger{
		Out: w,
		Formatter: &logrus.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
			FullTimestamp:   true,
			DisableSorting:  true,
		},
		Hooks: make(logrus.LevelHooks),
		Level: logLvl,
	}

	return logger, nil
}
