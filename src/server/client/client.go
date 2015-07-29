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
* Client definition for generic handling of connections.
 */
package client

import (
	"errors"
	"fmt"
	"io"
	"net"
	"server/encryption"
	"server/util"
	"strings"
)

type Client interface {
	// Returns the IP address of the underlying connection.
	IPAddr() string

	// Spins off a generator to handle incoming packets from the
	// underlying connection. Errors will cause the generator to
	// exit and the error responsible will be written to the channel
	// before the connection is closed. Successful reads are indicated
	// by writing nil on the channel, at which point the underlying
	// buffer accessible via Data() will contain the result of the read.
	// This function will not manage the client connection; i.e., the
	// caller needs to open and close the connection as necessary
	// (namely on EOF, which is written to the channel as io.EOF).
	Process() error

	// Encrypts a block of data in-place with the server key so that
	// it can be sent to the client.
	Encrypt(data []byte, size uint32)

	// Decrypts a block of data from the client in-place in order for
	// it to be processed by the server.
	Decrypt(data []byte, size uint32)

	// Returns a slice of the underlying array containing the packet data.
	// This is not copied in order to try and keep the memory usage low.
	Data() []byte

	// Send the block pointed to by data to the client. Since the
	// entire slice is written, sending only part of the buffer can
	// be accomplished via slicing by the caller.
	Send(data []byte) error

	// Close the underlying socket and stop processing packets.
	Close()
}

// Interface for passing around the individual server types.
type ClientWrapper interface {
	Client() Client
}

// Client struct intended to be included as part of the client definitions
// in each of the servers. This struct wraps the connection handling logic
// used by the generator below to handle receiving packets.
type PSOClient struct {
	conn   *net.TCPConn
	ipAddr string
	port   string

	// Exported so that callers can change this if needed.
	HdrSize    int
	recvSize   int
	packetSize uint16
	buffer     []byte

	clientCrypt *encryption.PSOCrypt
	serverCrypt *encryption.PSOCrypt
}

func NewPSOClient(conn *net.TCPConn, hdrSize int) *PSOClient {
	addr := strings.Split(conn.RemoteAddr().String(), ":")
	c := &PSOClient{
		conn:        conn,
		ipAddr:      addr[0],
		port:        addr[1],
		HdrSize:     hdrSize,
		clientCrypt: encryption.NewCrypt(),
		serverCrypt: encryption.NewCrypt(),
		buffer:      make([]byte, 512),
	}

	// Hacky, but we know that BB packets must be of length 8.
	if hdrSize >= 8 {
		c.clientCrypt.CreateBBKeys()
		c.serverCrypt.CreateBBKeys()
	} else {
		c.clientCrypt.CreateKeys()
		c.serverCrypt.CreateKeys()
	}
	return c
}

func (c *PSOClient) IPAddr() string {
	return c.ipAddr
}

func (c *PSOClient) Encrypt(data []byte, size uint32) {
	c.serverCrypt.Encrypt(data, size)
}

func (c *PSOClient) Decrypt(data []byte, size uint32) {
	c.clientCrypt.Decrypt(data, size)
}

func (c *PSOClient) ClientVector() []uint8 {
	return c.clientCrypt.Vector
}

func (c *PSOClient) ServerVector() []uint8 {
	return c.serverCrypt.Vector
}

func (c *PSOClient) Process() error {
	// Extra bytes left in the buffer will just be ignored.
	c.recvSize = 0
	c.packetSize = 0
	hdr16 := uint16(c.HdrSize)

	// Wait for the packet header.
	for c.recvSize < c.HdrSize {
		bytes, err := c.conn.Read(c.buffer[c.recvSize:c.HdrSize])
		if bytes == 0 || err == io.EOF {
			// The client disconnected, we're done.
			return err
		} else if err != nil {
			fmt.Println("Sockt error")
			// Socket error, nothing we can do now
			return errors.New("Socket Error (" + c.ipAddr + ") " + err.Error())
		}
		c.recvSize += bytes

		if c.recvSize >= c.HdrSize {
			// We have our header; decrypt it.
			c.Decrypt(c.buffer[:c.HdrSize], uint32(c.HdrSize))
			c.packetSize, err = util.GetPacketSize(c.buffer[:2])
			if err != nil {
				// Something is seriously wrong if this causes an error. Bail.
				panic(err.Error())
			}
			// PSO likes to occasionally send us packets that are longer
			// than their declared size. Adjust the expected length just
			// in case in order to avoid leaving stray bytes in the buffer.
			for c.packetSize%hdr16 != 0 {
				c.packetSize++
			}
		}
	}
	pktSize := int(c.packetSize)

	// Grow the client's receive buffer if they send us a packet bigger
	// than its current capacity.
	if pktSize > cap(c.buffer) {
		newSize := pktSize + len(c.buffer)
		newBuf := make([]byte, newSize)
		copy(newBuf, c.buffer)
		c.buffer = newBuf
	}

	// Read in the rest of the packet.
	for c.recvSize < pktSize {
		remaining := pktSize - c.recvSize
		bytes, err := c.conn.Read(c.buffer[c.recvSize : c.recvSize+remaining])
		if err != nil {
			return errors.New("Socket Error (" + c.ipAddr + ") " + err.Error())
		}
		c.recvSize += bytes
	}

	// We have the whole thing; decrypt the rest of it.
	if c.packetSize > hdr16 {
		c.Decrypt(c.buffer[hdr16:c.packetSize], uint32(c.packetSize-hdr16))
	}
	return nil
}

func (c *PSOClient) Data() []byte {
	return c.buffer
}

func (c *PSOClient) Send(data []byte) error {
	_, err := c.conn.Write(data)
	return err
}

func (c *PSOClient) Close() {
	c.conn.Close()
}
