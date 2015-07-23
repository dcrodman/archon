/*
* Archon Server Library
* Copyright (C) 2014 Andrew Rodman
*
* This program is free software: you can redistribute it and/or modify
* it under the terms of the GNU General Public License as published by
* the Free Software Foundation, either version 3 of the License, or
* (at your option) any later version.
*
* This program is distributed in the hope that it will be useful,
* but WITHOUT ANY WARRANTY; without even the implied warranty of
* MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
* GNU General Public License for more details.
*
* You should have received a copy of the GNU General Public License
* along with this program.  If not, see <http://www.gnu.org/licenses/>.
* ---------------------------------------------------------------------
*
* Lightweight wrapper class around Go's threadsafe logging library
* in order to allow a little more flexibility with Archon's logging.
 */

package logger

import (
	"io"
	"log"
	"os"
)

// Constants for the configurable log level that control the amount
// of information written to the server logs. The higher the number,
// the greater the verbosity.
type LogPriority byte

const (
	High   LogPriority = 1
	Medium             = 2
	Low                = 3
)

type ServerLogger struct {
	logger *log.Logger
	// Minimum priority for messages. Logs with a priority below
	// this level will not be written by the logger.
	minLevel LogPriority
}

// Creates a new writer with all output written to the file located
// at filename with any logs with a priority lower than level silently
// ignored. Passing "" for filename will cause logs to be written to stdout.
func New(filename string, level LogPriority) (*ServerLogger, error) {
	var w io.Writer
	var err error
	if filename != "" {
		w, err = os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
	} else {
		w = os.Stdout
	}

	l := log.New(w, "", log.Ldate|log.Ltime)
	return &ServerLogger{logger: l, minLevel: level}, nil
}

// Lower priority server information.
func (l *ServerLogger) Info(format string, v ...interface{}) {
	if l.minLevel >= Low {
		l.logger.Printf("[INFO] "+format, v...)
	}
}

// Warnings from internal operations.
func (l *ServerLogger) Warn(format string, v ...interface{}) {
	if l.minLevel >= Medium {
		l.logger.Printf("[WARN] "+format, v...)
	}
}

// Errors require admin attention.
func (l *ServerLogger) Error(format string, v ...interface{}) {
	if l.minLevel >= High {
		l.logger.Printf("[ERROR] "+format, v...)
	}
}

// Important messages ignore the user's logging preferences
// as they're essential to the operations of the server.
func (l *ServerLogger) Important(format string, v ...interface{}) {
	l.logger.Printf(format, v...)
}

// Print a message and exit; the server can't continue to function.
func (l *ServerLogger) Fatal(format string, v ...interface{}) {
	l.logger.Fatalf(format, v)
}
