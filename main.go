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
* ---------------------------------------------------------------------
 */
package main

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/dcrodman/archon/util"
	_ "github.com/go-sql-driver/mysql"
	"io"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"runtime/pprof"
	"strconv"
	"sync"
)

const (
	ServerConfigDir  = "/usr/local/share/archon"
	ServerConfigFile = "server_config.json"
	CertificateFile  = "certificate.pem"
	KeyFile          = "key.pem"
)

var (
	log *logrus.Logger
)

// Server defines the methods implemented by all sub-servers that can be
// registered and started when the server is brought up.
type Server interface {
	// Uniquely identifying string, mostly used for logging.
	Name() string
	// Port on which the server should listen for connections.
	Port() string
	// Perform any pre-startup initialization.
	Init()
	// Client factory responsible for performing whatever initialization is
	// needed for Client objects to represent new connections.
	NewClient(conn *net.TCPConn) (*Client, error)
	// Process the packet in the client's buffer. The dispatcher will
	// read the latest packet from the client before calling.
	Handle(c *Client) error
}

type Dispatcher struct {
	host    string
	servers []Server
	conns   *ConnList
	log     *logrus.Logger
}

// Registers a server instance to be brought up once the dispatcher is run.
func (d *Dispatcher) register(s Server) {
	d.servers = append(d.servers, s)
}

// Iterate over our registered servers, creating a goroutine for each
// one to listen on its registered port.
func (d *Dispatcher) start(wg *sync.WaitGroup) {
	for _, s := range d.servers {
		s.Init()
		// Open our server socket. All sockets must be open for the server
		// to launch correctly, so errors are terminal.
		hostAddr, err := net.ResolveTCPAddr("tcp", config.Hostname+":"+s.Port())
		if err != nil {
			fmt.Println("Error creating socket: " + err.Error())
			os.Exit(1)
		}
		socket, err := net.ListenTCP("tcp", hostAddr)
		if err != nil {
			fmt.Println("Error listening on socket: " + err.Error())
			os.Exit(1)
		}

		go func(serv Server) {
			wg.Add(1)
			// Poll until we can accept more clients.
			for d.conns.Count() < config.MaxConnections {
				conn, err := socket.AcceptTCP()
				if err != nil {
					d.log.Warnf("Failed to accept connection: %v", err.Error())
					continue
				}
				c, err := serv.NewClient(conn)
				if err != nil {
					d.log.Warn(err.Error())
				} else {
					d.log.Infof("Accepted %s connection from %s", serv.Name(), c.IPAddr())
					d.dispatch(c, serv)
				}
			}
			wg.Done()
		}(s)
	}
	// Pass through again to prevent the output from changing due to race cond.
	for _, s := range d.servers {
		fmt.Printf("Waiting for %s connections on %v:%v\n", s.Name(), d.host, s.Port())
	}
	d.log.Infof("Dispatcher: Server Initialized")
}

// Spawn a dedicated Goroutine for Client and handle communications
// until the connection is closed.
func (d *Dispatcher) dispatch(c *Client, s Server) {
	go func() {
		// Defer so that we catch any panics, d/c the client, and
		// remove them from the list regardless of the connection state.
		defer func() {
			if err := recover(); err != nil {
				d.log.Errorf("Error in client communication: %s: %s\n%s\n",
					c.IPAddr(), err, debug.Stack())
			}
			c.Close()
			d.conns.Remove(c)
			d.log.Infof("Disconnected %s client %s", s.Name(), c.IPAddr())
		}()
		d.conns.Add(c)

		// Connection loop; process packets until the connection is closed.
		var pktHeader PCHeader
		for {
			err := c.Process()
			if err == io.EOF {
				break
			} else if err != nil {
				// Error communicating with the client.
				d.log.Warn(err.Error())
				break
			}

			// PC and BB header packets have the same structure for the first four
			// bytes, so for basic inspection it's safe to treat them the same way.
			util.StructFromBytes(c.Data()[:PCHeaderSize], &pktHeader)
			if config.DebugMode {
				fmt.Printf("%s: Got %v bytes from client:\n\n", s.Name(), pktHeader.Size)
				util.PrintPayload(c.Data(), int(pktHeader.Size))
				fmt.Println()
			}

			if err = s.Handle(c); err != nil {
				d.log.Warn("Error in client communication: " + err.Error())
				return
			}
		}
	}()
}

func initLogger(filename string) {
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

func main() {
	fmt.Println("Archon PSO Server, Copyright (C) 2014 Andrew Rodman\n" +
		"=====================================================\n" +
		"This program is free software: you can redistribute it and/or\n" +
		"modify it under the terms of the GNU General Public License as\n" +
		"published by the Free Software Foundation, either version 3 of\n" +
		"the License, or (at your option) any later version.\n" +
		"This program is distributed WITHOUT ANY WARRANTY; See LICENSE for details.\n")

	// Initialize our config singleton from one of two expected file locations.
	fmt.Printf("Loading config file %v...", ServerConfigFile)
	err := config.InitFromFile(ServerConfigFile)
	if err != nil {
		os.Chdir(ServerConfigDir)
		fmt.Printf("Failed.\nLoading config from %v...", ServerConfigDir+"/"+ServerConfigFile)
		err = config.InitFromFile(ServerConfigFile)
		if err != nil {
			fmt.Println("Failed.\nPlease check that one of these files exists and restart the server.")
			fmt.Printf("Error: %s\n", err)
			os.Exit(1)
		}
	}
	fmt.Printf("Done.\n\n--Configuration Parameters--\n%v\n\n", config.String())

	// Initialize the database.
	fmt.Printf("Connecting to MySQL database %s:%s...", config.DBHost, config.DBPort)
	err = config.InitDb()
	if err != nil {
		fmt.Println("Failed.\nPlease make sure the database connection parameters are correct.")
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	fmt.Println("Done.\n")
	defer config.CloseDB()

	// If we're in debug mode, spawn off an HTTP server that, when hit, dumps
	// pprof output containing the stack traces of all running goroutines.
	if config.DebugMode {
		http.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
			pprof.Lookup("goroutine").WriteTo(resp, 1)
		})
		go http.ListenAndServe(config.WebPort, nil)
	}

	initLogger(config.Logfile)

	// Register all of the server handlers and their corresponding ports.
	dispatcher := Dispatcher{
		host:    config.Hostname,
		servers: make([]Server, 0),
		conns:   NewClientList(),
		log:     log,
	}

	dispatcher.register(new(PatchServer))
	dispatcher.register(new(DataServer))
	dispatcher.register(new(LoginServer))
	dispatcher.register(new(CharacterServer))
	dispatcher.register(new(ShipServer))

	// The available block ports will depend on how the server is configured,
	// so once we've read the config then add the server entries on the fly.
	shipPort, _ := strconv.ParseInt(config.ShipPort, 10, 16)
	for i := 1; i <= config.NumBlocks; i++ {
		dispatcher.register(BlockServer{
			name: fmt.Sprintf("BLOCK%d", i),
			port: strconv.FormatInt(shipPort+int64(i), 10),
		})
	}

	// Start up all of our servers and block until they exit.
	var wg sync.WaitGroup
	dispatcher.start(&wg)
	wg.Wait()
}
