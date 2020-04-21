package server

import (
	"container/list"
	"github.com/spf13/viper"
	"sync"
)

// Archon uses a shared list of clients across all servers in order to prevent
// clients from connecting to any part of the server and causing problems.
var globalClientList = &clientList{
	clients: list.New(),
	RWMutex: sync.RWMutex{},
}

func isServerFull() bool {
	return globalClientList.len() >= viper.GetInt("max_connections")
}

// A concurrency-safe wrapper around container/list for maintaining a collection of connected clients.
type clientList struct {
	clients *list.List
	sync.RWMutex
}

func (c *clientList) add(cl *ConnectionState) {
	c.Lock()
	c.clients.PushBack(cl)
	c.Unlock()
}

func (c *clientList) remove(cl *ConnectionState) {
	clAddr := cl.IPAddr()
	c.Lock()
	for clientElem := c.clients.Front(); clientElem != nil; clientElem = clientElem.Next() {
		client := clientElem.Value.(*ConnectionState)
		if client.IPAddr() == clAddr {
			c.clients.Remove(clientElem)
			break
		}
	}
	c.Unlock()
}

// Note: this comparison is by IP address, not element value.
func (c *clientList) has(cl *ConnectionState) bool {
	clAddr := cl.IPAddr()
	c.RLock()
	defer c.RUnlock()
	for clientElem := c.clients.Front(); clientElem != nil; clientElem = clientElem.Next() {
		if cl.IPAddr() == clAddr {
			return true
		}
	}
	return false
}

func (c *clientList) len() int {
	c.RLock()
	defer c.RUnlock()
	return c.clients.Len()
}
