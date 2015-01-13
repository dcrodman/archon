/*
* Archon Login Server
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
* CHARACTER server logic.
 */
package server

import (
	"fmt"
	"libarchon/util"
	"os"
	"sync"
)

// Main worker thread for the CHARACTER portion of the server.
func StartCharacter(wg *sync.WaitGroup) {
	loginConfig := GetConfig()
	socket, err := util.OpenSocket(loginConfig.Hostname, loginConfig.CharacterPort)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	fmt.Printf("Waiting for CHARACTER connections on %s:%s...\n\n",
		loginConfig.Hostname, loginConfig.CharacterPort)

	for {
		connection, err := socket.AcceptTCP()
		if err != nil {
			fmt.Println("Error accepting connection: " + err.Error())
			continue
		}
		fmt.Printf("Accepted CHARACTER connection from %s\n", connection.RemoteAddr())
	}
	wg.Done()
}
