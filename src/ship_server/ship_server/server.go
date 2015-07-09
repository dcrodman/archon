/*
* Archon Ship Server
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
package ship_server

import (
	"errors"
	"fmt"
	"net"
)

var Shipgate struct {
	ip   *net.TCPAddr
	conn *net.TCPConn
}

func (s *Shipgate) Reconnect() {

}

func connectShipgate() error {
	cfg := GetConfig()
	shipgate.conn, err = net.DialTCP("tcp", cfg.ShipgateHost+":"+cfg.ShipgatePort, nil)
	if err != nil {
		return errors.New("Error connecting to shipgate: " + err.Error())
	}
}

func StartServer() {
	fmt.Println("Initializing Archon Ship server...")
	config := GetConfig()

	// Initialize our config singleton from one of two expected file locations.
	fmt.Printf("Loading config file %v...", ShipConfigFile)
	err := config.InitFromFile(ShipConfigFile)
	if err != nil {
		os.Chdir(ServerConfigDir)
		fmt.Printf("Failed.\nLoading config from %v...", ServerConfigDir+"/"+ShipConfigFile)
		err = config.InitFromFile(ShipConfigFile)
		if err != nil {
			fmt.Println("Failed.\nPlease check that one of these files exists and restart the server.")
			fmt.Printf("%s\n", err.Error())
			os.Exit(-1)
		}
	}
	fmt.Printf("Done.\n\n--Configuration Parameters--\n%v\n\n", config.String())
}
