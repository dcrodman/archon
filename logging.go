package archon

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"os"
)

var Log *logrus.Logger

func init() {
	var w io.Writer
	var err error

	if Config.Logfile != "" {
		w, err = os.OpenFile(Config.Logfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			fmt.Println("ERROR: Failed to open Log file " + Config.Logfile)
			os.Exit(1)
		}
	} else {
		w = os.Stdout
	}

	logLvl, err := logrus.ParseLevel(Config.LogLevel)
	if err != nil {
		fmt.Println("ERROR: Failed to parse Log level: " + err.Error())
		os.Exit(1)
	}

	Log = &logrus.Logger{
		Out: w,
		Formatter: &logrus.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
			FullTimestamp:   true,
			DisableSorting:  true,
		},
		Hooks: make(logrus.LevelHooks),
		Level: logLvl,
	}
}
