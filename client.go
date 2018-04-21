/*
 * Client definition for generic handling of connections.
 */
package main

import (
	"errors"
	"fmt"
	crypto "github.com/dcrodman/archon/encryption"
	"github.com/dcrodman/archon/util"
	"io"
	"net"
	"strings"
)

// Client struct intended to be included as part of the client definitions
// in each of the servers. This struct wraps the connection handling logic
// used by Process() below to handle receiving packets.
type Client struct {
	conn   *net.TCPConn
	ipAddr string
	port   string

	hdrSize    uint16
	recvSize   int
	packetSize uint16
	buffer     []byte

	clientCrypt *crypto.PSOCrypt
	serverCrypt *crypto.PSOCrypt

	guildcard uint32
	teamId    uint32
	isGm      bool

	// Patch server; list of files that need update.
	updateList []*PatchEntry

	gcData     []byte
	gcDataSize uint16
	config     ClientConfig
	flag       uint32
}

func NewClient(conn *net.TCPConn, hdrSize uint16, cCrypt, sCrypt *crypto.PSOCrypt) *Client {
	addr := strings.Split(conn.RemoteAddr().String(), ":")
	c := &Client{
		conn:        conn,
		ipAddr:      addr[0],
		port:        addr[1],
		hdrSize:     hdrSize,
		clientCrypt: cCrypt,
		serverCrypt: sCrypt,
		buffer:      make([]byte, 512),
	}
	return c
}

func (c *Client) IPAddr() string {
	return c.ipAddr
}

func (c *Client) ClientVector() []uint8 {
	return c.clientCrypt.Vector
}

func (c *Client) ServerVector() []uint8 {
	return c.serverCrypt.Vector
}

// Data returns the current contents of the buffer read from the client.
func (c *Client) Data() []byte {
	return c.buffer
}

func (c *Client) SendEncrypted(data []byte, length int) error {
	bytes, blen := fixLength(data, uint16(length), c.hdrSize)
	if config.DebugMode {
		util.PrintPayload(bytes, int(blen))
		fmt.Println()
	}

	c.Encrypt(bytes, uint32(blen))
	return c.SendRaw(bytes, int(blen))
}

// fixLength pads the length of a packet to a multiple of 8 and set the first two bytes of the header.
func fixLength(data []byte, length uint16, hdrSize uint16) ([]byte, uint16) {
	for length%hdrSize != 0 {
		length++
		data = append(data, 0)
	}
	data[0] = byte(length & 0xFF)
	data[1] = byte((length & 0xFF00) >> 8)
	return data, length
}

// SendRow writes all data contained in the slice to the client as-is.
// Note: Packets sent to BB Clients must have a length divisible by 8.
func (c *Client) SendRaw(data []byte, length int) error {
	bytesSent := 0
	for bytesSent < length {
		b, err := c.conn.Write(data[:length])
		if err != nil {
			return fmt.Errorf("Error sending to client %v: %s", c.IPAddr(), err.Error())
		}
		bytesSent += b
	}
	return nil
}

// Encrypt a block of data of the given size in-place using the server's cipher
// in order to prep it for sending to the client.
func (c *Client) Encrypt(data []byte, size uint32) {
	c.serverCrypt.Encrypt(data, size)
}

// Decrypt a block of data of the given size in-place using the client's cipher
// in order to prep it for reading.
func (c *Client) Decrypt(data []byte, size uint32) {
	c.clientCrypt.Decrypt(data, size)
}

// Process blocks until we read the next packet from the client. Once we get the full
// packet, this method decrypts and stores it in the client's buffer variable.
//
// Warning: Calling this method without first grabbing the data from the buffer will
// cause you to lose the packet since it overwrites the contents of buffer.
func (c *Client) Process() error {
	// Extra bytes left in the buffer will just be ignored.
	c.recvSize = 0
	c.packetSize = 0
	hdrint := int(c.hdrSize)

	// Wait for the packet header.
	for c.recvSize < hdrint {
		bytes, err := c.conn.Read(c.buffer[c.recvSize:c.hdrSize])
		if bytes == 0 || err == io.EOF {
			// The client disconnected, we're done.
			return err
		} else if err != nil {
			fmt.Println("Sockt error")
			// Socket error, nothing we can do now.
			return errors.New("Socket Error (" + c.ipAddr + ") " + err.Error())
		}
		c.recvSize += bytes

		if c.recvSize >= hdrint {
			// We have our header; decrypt it.
			c.Decrypt(c.buffer[:c.hdrSize], uint32(c.hdrSize))
			c.packetSize, err = util.GetPacketSize(c.buffer[:2])
			if err != nil {
				// Something is seriously wrong if this causes an error. Bail.
				panic(err.Error())
			}
			// PSO likes to occasionally send us packets that are longer than their declared
			// size, but are always a multiple of the length of the packet header. Adjust the
			// expected length just in case in order to avoid leaving stray bytes in the buffer.
			c.packetSize += c.packetSize % c.hdrSize
		}
	}
	pktSize := int(c.packetSize)

	// Grow the client's receive buffer if they send us a packet bigger than its current capacity.
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
	if c.packetSize > c.hdrSize {
		c.Decrypt(c.buffer[c.hdrSize:c.packetSize], uint32(c.packetSize-c.hdrSize))
	}
	return nil
}

func (c *Client) Close() {
	c.conn.Close()
}
