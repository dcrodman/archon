/*
* Archon PSO Server
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
 */
package main

import (
	"container/list"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
)

const (
	// ServerConfigDir is the configuration directory that Archon will fall back to.
	ServerConfigDir = "/usr/local/etc/archon"
	// ServerConfigFile is the filename of the config file Archon expects.
	ServerConfigFile = "config.yaml"
)

// Global variables that should not be globals at some point.
var (
	log        *logrus.Logger
	configPath = flag.String("conf", "", "Full path to a custom config file location")
)

func main() {
	fmt.Println("Archon PSO Server, Copyright (C) 2014 Andrew Rodman\n" +
		"=====================================================\n" +
		"This program is free software: you can redistribute it and/or\n" +
		"modify it under the terms of the GNU General Public License as\n" +
		"published by the Free Software Foundation, either version 3 of\n" +
		"the License, or (at your option) any later version.\n" +
		"This program is distributed WITHOUT ANY WARRANTY; See LICENSE for details.\n")
	flag.Parse()

	// Initialize our config singleton from one of two expected file locations.
	var err error
	if *configPath == "" {
		fmt.Printf("Loading configuration file %v...", ServerConfigFile)
		if config.InitFromFile(ServerConfigFile) != nil {
			fmt.Printf("Failed.\nLoading configuration from %v...", ServerConfigDir+"/"+ServerConfigFile)
			os.Chdir(ServerConfigDir)
			err = config.InitFromFile(ServerConfigFile)
		}
	} else {
		fmt.Printf("Loading configuration file %v...", *configPath)
		err = config.InitFromFile(*configPath)
	}

	if err != nil {
		fmt.Println("Failed.\nPlease check that one of these files exists and restart the server.")
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("Done.\n\n--Configuration Parameters--\n%v\n\n", config.String())

	// Set up the database singleton with the params from the config file.
	fmt.Printf("Connecting to database %s:%s...", config.DBHost, config.DBPort)
	database, err = InitializeDatabase()
	if err != nil {
		fmt.Println("Failed: " + err.Error())
		os.Exit(1)
	}
	// TODO: This should probably be done in a signal handler or somewhere more guaranteed.
	defer database.Close()
	fmt.Print("Done.\n\n")

	StartDebugServer()
	initializeLogger(config.Logfile)

	c := controller{
		host:        config.Hostname,
		servers:     make([]Server, 0),
		connections: &clientList{clients: list.New()},
	}
	registerServers(&c)

	// Start up all of our servers and block until they exit.
	wg := c.start()
	if wg != nil {
		wg.Wait()
	}
}

// Set up the logger to write to the specified filename.
func initializeLogger(filename string) {
	var w io.Writer
	var err error
	if filename != "" {
		w, err = os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			fmt.Println("ERROR: Failed to open log file " + config.Logfile)
			os.Exit(1)
		}
	} else {
		w = os.Stdout
	}

	logLvl, err := logrus.ParseLevel(config.LogLevel)
	if err != nil {
		fmt.Println("ERROR: Failed to parse log level: " + err.Error())
		os.Exit(1)
	}
	log = &logrus.Logger{
		Out: w,
		Formatter: &logrus.TextFormatter{
			TimestampFormat: "2006-1-_2 15:04:05",
			FullTimestamp:   true,
			DisableSorting:  true,
		},
		Hooks: make(logrus.LevelHooks),
		Level: logLvl,
	}
}

// Register all of the server handlers and their corresponding ports.
func registerServers(controller *controller) {
	controller.registerServer(new(PatchServer))
	controller.registerServer(new(DataServer))
	controller.registerServer(new(LoginServer))
	controller.registerServer(new(CharacterServer))
	controller.registerServer(new(ShipgateServer))
	controller.registerServer(new(ShipServer))

	// The available block ports will depend on how the server is configured,
	// so once we've read the config then add the server entries on the fly.
	shipPort, _ := strconv.ParseInt(config.ShipPort, 10, 16)
	for i := 1; i <= config.NumBlocks; i++ {
		controller.registerServer(&BlockServer{
			name: fmt.Sprintf("BLOCK%d", i),
			port: strconv.FormatInt(shipPort+int64(i), 10),
		})
	}
}
