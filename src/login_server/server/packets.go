package server

/*
 * Packet constants and handlers. All handlers return 0 on success, negative int on db error, and a
 * positive int for any other errors.
 */

import (
	"bytes"
	"fmt"
	//"unsafe"
)

/* Packet headers */

const WELCOME_TYPE = 0x93

/* Other constants */

const bbCopyright = "Phantasy Star Online Blue Burst Game Server. Copyright 1999-2004 SONICTEAM."

var copyrightBytes []byte = make([]byte, 96)

type BBPktHeader struct {
	pktSize uint16
	pktType uint16
	padding uint32
}

type WelcomePkt struct {
	header       BBPktHeader
	copyright    []byte  // 96b
	serverVector []uint8 // 48b
	clientVector []uint8 // 48b
}

func SendPacket(client *Client, pkt []byte) int {
	fmt.Println("Packet: " + string(pkt))
	return 0
}

func SendWelcome(client *Client) int {
	pkt := new(WelcomePkt)
	pkt.header.pktType = WELCOME_TYPE
	pkt.copyright = copyrightBytes
	copy(pkt.clientVector, client.clientCrypt.Vector)
	copy(pkt.serverVector, client.serverCrypt.Vector)

	//SendPacket(client, )
	//pkt.header.pktSize = unsafe.Sizeof(pkt)
	return 0
}

func init() {
	tmp := new(bytes.Buffer)
	tmp.WriteString(bbCopyright)
	copy(copyrightBytes, bbCopyright)
}
