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
* Debugging utilities for server admins.
 */
package server

import (
	"container/list"
	"errors"
	"net"
	"net/http"
	"runtime/pprof"
	"sync"
)

// Client interface to make it possible to share common client-related
// functionality between servers without exposing the server-specific config.
type PSOClient interface {
	Connection() *net.TCPConn
}

// Synchronized list for maintaining a list of connected clients.
type ConnectionList struct {
	clientList *list.List
	size       int
	mutex      sync.RWMutex
}

// Creates a simple Http server on host, listening for requests to the url
// at path. Responses are dumps from pprof containing the stack traces of
// all running goroutines.
func CreateStackTraceServer(host, path string) {
	http.HandleFunc(path, func(resp http.ResponseWriter, req *http.Request) {
		pprof.Lookup("goroutine").WriteTo(resp, 1)
	})
	http.ListenAndServe(host, nil)
}

// Opens a TCP socket on host:port and returns either an error or
// a listener socket ready to Accept().
func OpenSocket(host, port string) (*net.TCPListener, error) {
	hostAddress, err := net.ResolveTCPAddr("tcp", host+":"+port)
	if err != nil {
		return nil, errors.New("Error creating socket: " + err.Error())
	}
	socket, err := net.ListenTCP("tcp", hostAddress)
	if err != nil {
		return nil, errors.New("Error Listening on Socket: " + err.Error())
	}
	return socket, nil
}

// Factory method for creating new ConnectionLists.
func NewClientList() *ConnectionList {
	newList := new(ConnectionList)
	newList.clientList = list.New()
	return newList
}

func (cl *ConnectionList) AddClient(c PSOClient) {
	cl.mutex.Lock()
	cl.clientList.PushBack(c)
	cl.size++
	cl.mutex.Unlock()
}

func (cl *ConnectionList) HasClient(c PSOClient) bool {
	found := false
	cl.mutex.RLock()
	for client := cl.clientList.Front(); client != nil; client = client.Next() {
		if client.Value == c {
			found = true
			break
		}
	}
	cl.mutex.RUnlock()
	return found
}

func (cl *ConnectionList) RemoveClient(c PSOClient) {
	cl.mutex.Lock()
	for client := cl.clientList.Front(); client != nil; client = client.Next() {
		if client.Value == c {
			cl.clientList.Remove(client)
			cl.size--
			break
		}
	}
	cl.mutex.Unlock()
}

func (cl *ConnectionList) Count() int {
	cl.mutex.RLock()
	length := cl.size
	cl.mutex.RUnlock()
	return length
}
