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

import (
	"container/list"
	"net"
	"sync"
)

type Client interface {
	Connection() *net.TCPConn
	IPAddr() string
}

// Synchronized list for maintaining a list of connected clients.
type ConnectionList struct {
	clientList *list.List
	mutex      sync.RWMutex
}

func NewClientList() *ConnectionList {
	newList := new(ConnectionList)
	newList.clientList = list.New()
	return newList
}

func (cl *ConnectionList) AddClient(c Client) {
	cl.mutex.Lock()
	cl.clientList.PushBack(c)
	cl.mutex.Unlock()
}

func (cl *ConnectionList) RemoveClient(c Client) {
	cl.mutex.Lock()
	for client := cl.clientList.Front(); client != nil; client = client.Next() {
		// TODO: Do we care if the client isn't present?
		if client.Value == c {
			cl.clientList.Remove(client)
			break
		}
	}
	cl.mutex.Unlock()
}

func (cl *ConnectionList) Count() int {
	return cl.clientList.Len()
}

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
