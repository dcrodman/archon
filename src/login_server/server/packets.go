// Packet constants and handlers. All handlers return 0 on success, negative int on
// db error, and a positive int for any other errors.
package server

import (
	"fmt"
	"libarchon/util"
)

// Packet headers.
const BBHeaderSize = 0x08
const WelcomeType = 0x03
const WelcomeSize = 0xC8

// Other constants.
const bbCopyright = "Phantasy Star Online Blue Burst Game Server. Copyright 1999-2004 SONICTEAM."

var copyrightBytes []byte = make([]byte, 96)

// Packet structures.
type BBPktHeader struct {
	Size    uint16
	Type    uint16
	Padding uint32
}

type WelcomePkt struct {
	Header       BBPktHeader
	Copyright    [96]byte
	ServerVector [48]uint8
	ClientVector [48]uint8
}

// Send the packet serialized (or otherwise contained) in pkt to a client.
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

// Send the welcome packet to a client with the copyright message and encryption vectors.
func SendWelcome(client *Client) int {
	pkt := new(WelcomePkt)
	pkt.Header.Size = WelcomeSize
	pkt.Header.Type = WelcomeType
	copy(pkt.Copyright[:], copyrightBytes)
	copy(pkt.ClientVector[:], client.clientCrypt.Vector)
	copy(pkt.ServerVector[:], client.serverCrypt.Vector)

	data := util.BytesFromStruct(pkt)
	fmt.Println("Sending Welcome Packet")
	util.PrintPayload(data, WelcomeSize)
	return SendPacket(client, data, WelcomeSize)
}

func init() {
	copy(copyrightBytes, bbCopyright)
}
