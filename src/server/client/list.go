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
package client

import (
	"container/list"
	"sync"
)

// Synchronized list for maintaining a list of connected clients.
type ConnList struct {
	clientList *list.List
	size       int
	mutex      sync.RWMutex
}

// Factory method for creating lists of connected clients.
func NewList() *ConnList {
	return &ConnList{clientList: list.New()}
}

// Appends a client to the end of the connection list.
func (cl *ConnList) Add(c ClientWrapper) {
	cl.mutex.Lock()
	cl.clientList.PushBack(c)
	cl.size++
	cl.mutex.Unlock()
}

// Returns true if the list has a Client matching the IP address of c.
// Note that this comparison is by IP address, not element value.
func (cl *ConnList) Has(c ClientWrapper) bool {
	found := false
	clAddr := c.Client().IPAddr()
	cl.mutex.RLock()
	for client := cl.clientList.Front(); client != nil; client = client.Next() {
		if client.Value.(ClientWrapper).Client().IPAddr() == clAddr {
			found = true
			break
		}
	}
	cl.mutex.RUnlock()
	return found
}

func (cl *ConnList) Remove(c ClientWrapper) {
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

func (cl *ConnList) Count() int {
	cl.mutex.RLock()
	length := cl.size
	cl.mutex.RUnlock()
	return length
}
