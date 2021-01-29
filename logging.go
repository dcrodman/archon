package archon

import (
	"fmt"
	"io"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Log is the global, threadsafe logger that can be used by any server instance.
var Log *logrus.Logger

// InitLogger configures the global logger and should be called on startup.
func InitLogger() {
	var w io.Writer
	var err error

	logFile := viper.GetString("log_file_path")

	if logFile == "" {
		w = os.Stdout
	} else {
		w, err = os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			fmt.Println("ERROR: Failed to open Log file " + logFile)
			os.Exit(1)
		}
	}

	logLvl, err := logrus.ParseLevel(viper.GetString("log_level"))
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
