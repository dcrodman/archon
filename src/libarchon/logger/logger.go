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
* Logging utility.
 */

package logger

import (
	"fmt"
	"io"
	"time"
)

// Constants for the configurable log level that control the amount of information
// written to the server logs. The higher the number, the lower the priority.
type LogPriority byte

const (
	CriticalPriority LogPriority = 1
	HighPriority                 = 2
	MediumPriority               = 3
	LowPriority                  = 4
)

type Logger struct {
	out io.Writer
	// Minimum priority for messages. Logs with a priority below
	// this level will not be written by the logger.
	minLevel LogPriority
}

func New(writer io.Writer, level LogPriority) *Logger {
	return &Logger{out: writer, minLevel: level}
}

func (logger *Logger) Info(msg string, priority LogPriority) {
	if logger.minLevel < priority {
		return
	}
	logger.logMsg(fmt.Sprintf("[INFO] %s\n", msg))
}

func (logger *Logger) Warn(msg string, priority LogPriority) {
	if logger.minLevel < priority {
		return
	}
	logger.logMsg(fmt.Sprintf("[WARNING] %s\n", msg))
}

func (logger *Logger) Error(msg string, priority LogPriority) {
	if logger.minLevel < priority {
		return
	}
	logger.logMsg(fmt.Sprintf("[ERROR] %s\n", msg))
}

// DB errors are considered critical, but may not be worth stopping the server.
// Log both to stdout and the log file for max visibility.
func (logger *Logger) DBError(msg string) {
	errMsg := fmt.Sprintf("[SQL ERROR] %s", msg)
	fmt.Println(errMsg)
	logger.Error(msg, CriticalPriority)
}

// Logs a message to either the user's configured logfile or to standard out. Only messages
// equal to or greater than the user's specified priority will be written.
func (logger *Logger) logMsg(msg string) {
	timestamp := time.Now().Format("06-01-02 15:04:05")
	_, err := logger.out.Write([]byte(fmt.Sprintf("%s %s", timestamp, msg)))
	if err != nil {
		fmt.Printf("WARNING: Error encountered writing to log: %s\n", err.Error())
	}
}
