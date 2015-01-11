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
 */
package util

import "net"

// Create a TCP socket that is listening and ready to Accept().
func OpenSocket(host, port string) (*net.TCPListener, error) {
	hostAddress, err := net.ResolveTCPAddr("tcp", host+":"+port)
	if err != nil {
		return nil, &ServerError{Message: "Error creating socket: " + err.Error()}
	}
	socket, err := net.ListenTCP("tcp", hostAddress)
	if err != nil {
		return nil, &ServerError{Message: "Error Listening on Socket: " + err.Error()}
	}
	return socket, nil
}
