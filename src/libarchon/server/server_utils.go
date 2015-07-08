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
	"fmt"
	"io"
	"libarchon/encryption"
	"libarchon/util"
	"net"
	"sync"
)

// Redeclaring this since it's not going to change and it saves
// us from having to pass it around as an arugment to Generate.
const BBHeaderSize = 8

// Client interface to make it possible to share common client-related
// functionality between servers without exposing the server-specific config.
type PSOClient interface {
	Client() *Client
	IPAddr() string
}

// Client struct intended to be included as part of the client definitions
// in each of the servers. This struct wraps the connection handling logic
// used by the generator below to handle receiving packets.
type Client struct {
	Conn   *net.TCPConn
	IpAddr string
	Port   string

	RecvData   []byte
	RecvSize   int
	PacketSize uint16

	ClientCrypt *encryption.PSOCrypt
	ServerCrypt *encryption.PSOCrypt
}

// Synchronized list for maintaining a list of connected clients.
type ConnectionList struct {
	clientList *list.List
	size       int
	mutex      sync.RWMutex
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

// Returns true if the list has a PSOClient matching the IP address of c.
// Note that this comparison is by IP address, not element value.
func (cl *ConnectionList) HasClient(c PSOClient) bool {
	found := false
	clAddr := c.IPAddr()
	cl.mutex.RLock()
	for client := cl.clientList.Front(); client != nil; client = client.Next() {
		if client.Value.(PSOClient).IPAddr() == clAddr {
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

// Spins off a generator to handle incoming packets from pc. Connection
// errors will cause the generator to exit and the error responsible will
// be written to the channel. Successful reads are indicated by writing
// nil on the channel, at which point the client struct pointed to by
// pc.Client() will contain the result of the read. This function will not
// manage the client connection; i.e., the caller needs to open and close
// the connection as necessary (namely EOF, which is written to the channel).
func Generate(pc PSOClient, hdrSize int) <-chan error {
	out := make(chan error)
	go func() {
		c := pc.Client()
		hdr16 := uint16(hdrSize)
		for {
			// Wait for the packet header.
			for c.RecvSize < hdrSize {
				bytes, err := c.Conn.Read(c.RecvData[c.RecvSize:hdrSize])
				if bytes == 0 || err == io.EOF {
					// The client disconnected, we're done.
					out <- err
					goto bail
				} else if err != nil {
					fmt.Println("Sockt error")
					// Socket error, nothing we can do now
					out <- errors.New("Socket Error (" + c.IpAddr + ") " + err.Error())
					goto bail
				}
				c.RecvSize += bytes

				if c.RecvSize >= hdrSize {
					// We have our header; decrypt it.
					c.ClientCrypt.Decrypt(c.RecvData[:hdrSize], uint32(hdrSize))
					c.PacketSize, err = util.GetPacketSize(c.RecvData[:2])
					if err != nil {
						// Something is seriously wrong if this causes an error. Bail.
						panic(err.Error())
					}
					// PSO likes to occasionally send us packets that are longer
					// than their declared size. Adjust the expected length just
					// in case in order to avoid leaving stray bytes in the buffer.
					for c.PacketSize%hdr16 != 0 {
						c.PacketSize++
					}
				}
			}
			pktSize := int(c.PacketSize)
			// Grow the client's receive buffer if they send us a packet bigger
			// than its current capacity.
			if pktSize > cap(c.RecvData) {
				newSize := pktSize + len(c.RecvData)
				newBuf := make([]byte, newSize)
				copy(newBuf, c.RecvData)
				c.RecvData = newBuf
			}

			// Read in the rest of the packet.
			for c.RecvSize < pktSize {
				remaining := pktSize - c.RecvSize
				bytes, err := c.Conn.Read(c.RecvData[c.RecvSize : c.RecvSize+remaining])
				if err != nil {
					out <- errors.New("Socket Error (" + c.IpAddr + ") " + err.Error())
					goto bail
				}
				c.RecvSize += bytes
			}

			// We have the whole thing; decrypt the rest of it.
			if c.PacketSize > hdr16 {
				c.ClientCrypt.Decrypt(
					c.RecvData[hdr16:c.PacketSize],
					uint32(c.PacketSize-hdr16))
			}
			// Write out our nil value on the channel to tell the client that
			// a packet was received.
			out <- nil

			// Extra bytes left in the buffer will just be ignored.
			c.RecvSize = 0
			c.PacketSize = 0
		}
	bail:
		// Cheating with a goto to save us having to break twice.
		close(out)
	}()
	return out
}
