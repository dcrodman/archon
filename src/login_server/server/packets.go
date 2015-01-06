// Packet constants and handlers. All handlers return 0 on success, negative int on
// db error, and a positive int for any other errors.
package server

import (
	"libtethealla/util"
)

// Packet headers.

const WELCOME_TYPE = 0x03

const WELCOME_SIZE = 0xC8

// Other constants.

const bbCopyright = "Phantasy Star Online Blue Burst Game Server. Copyright 1999-2004 SONICTEAM."

var copyrightBytes []byte = make([]byte, 96)

// Struct types.

type BBPktHeader struct {
	Size    uint16
	Type    uint16
	Padding uint32
}

type WelcomePkt struct {
	Header       BBPktHeader
	Copyright    []byte  // 96b
	ServerVector []uint8 // 48b
	ClientVector []uint8 // 48b
}

func SendPacket(client *Client, pkt []byte, length int) int {
	// Write will return the number of bytes sent, but at this point I'm assuming that the
	// method will handle sending all of bytes to the client (as opposed to C's send) so I'm
	// going to ignore it unless it becomes a problem.
	_, err := client.conn.Write(pkt[:length])
	if err != nil {
		// TODO: Log error.
		return -1
	}
	return 0
}

func SendWelcome(client *Client) int {
	pkt := new(WelcomePkt)
	pkt.Header.Size = WELCOME_SIZE
	pkt.Header.Type = WELCOME_TYPE
	pkt.Copyright = copyrightBytes
	pkt.ClientVector = client.clientCrypt.Vector
	pkt.ServerVector = client.serverCrypt.Vector

	data := util.StructToBytes(pkt)
	return SendPacket(client, data, WELCOME_SIZE)
}

func init() {
	copy(copyrightBytes, bbCopyright)
}
