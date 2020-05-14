package launcher

import (
	"container/list"
	"github.com/dcrodman/archon/internal/server"
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

func (cl *clientList) add(c *server.Client) {
	cl.Lock()
	cl.clients.PushBack(c)
	cl.Unlock()
}

func (cl *clientList) remove(c *server.Client) {
	clAddr := c.IPAddr()
	cl.Lock()

	for clientElem := cl.clients.Front(); clientElem != nil; clientElem = clientElem.Next() {
		client := clientElem.Value.(*server.Client)

		if client.IPAddr() == clAddr {
			cl.clients.Remove(clientElem)
			break
		}
	}

	cl.Unlock()
}

// Note: this comparison is by IP address, not element value.
func (cl *clientList) has(c *server.Client) bool {
	clAddr := c.IPAddr()

	cl.RLock()
	defer cl.RUnlock()

	for clientElem := cl.clients.Front(); clientElem != nil; clientElem = clientElem.Next() {
		if c.IPAddr() == clAddr {
			return true
		}
	}
	return false
}

func (cl *clientList) len() int {
	cl.RLock()
	defer cl.RUnlock()
	return cl.clients.Len()
}
