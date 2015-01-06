// Packet constants and handlers. All handlers return 0 on success, negative int on
// db error, and a positive int for any other errors.
package server

import (
	//"bytes"
	"fmt"
	"libtethealla/util"
)

// Packet headers.

const WELCOME_TYPE = 0x03
const WELCOME_SIZE = 0x93

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

// Build/Send functions.

func SendPacket(client *Client, pkt []byte) int {
	fmt.Println("Would Send:")
	util.PrintPayload(pkt)
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
	return SendPacket(client, data)
}

func init() {
	copy(copyrightBytes, bbCopyright)
}
