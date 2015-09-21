/*
* Archon PSO Server
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
* The BLOCK and SHIP server logic.
 */
package main

import (
	"errors"
	"fmt"
	"github.com/dcrodman/archon/util"
	"io"
	"net"
	"strconv"
)

const BlockName = "Block"

var (
	// Precomputed block packet.
	blockPkt *BlockListPacket
)

type Block struct {
	Unknown   uint16
	BlockId   uint32
	Padding   uint16
	BlockName [36]byte
}

func handleShipLogin(sc *Client) error {
	if _, err := VerifyAccount(sc); err != nil {
		return err
	}
	sc.SendSecurity(BBLoginErrorNone, sc.guildcard, sc.teamId)
	return nil
}

func handleBlockSelection(sc *Client) error {
	var pkt MenuSelectionPacket
	util.StructFromBytes(sc.Data(), &pkt)

	// Grab the chosen block and redirect them to the selected block server.
	port, _ := strconv.ParseInt(config.ShipPort, 10, 16)
	selectedBlock := pkt.ItemId
	if int(selectedBlock) > config.NumBlocks {
		return errors.New(fmt.Sprintf("Block selection %v out of range %v", selectedBlock, config.NumBlocks))
	}
	fmt.Printf("Redirecting to port: %v\n", uint16(port)+uint16(selectedBlock))
	sc.SendRedirect(uint16(uint32(port)+selectedBlock), config.HostnameBytes())
	return nil
}

func NewShipClient(conn *net.TCPConn) (*Client, error) {
	sc := NewClient(conn, BBHeaderSize)
	err := error(nil)
	if sc.SendWelcome() != 0 {
		err = errors.New("Error sending welcome packet to: " + sc.IPAddr())
		sc = nil
	}
	return sc, err
}

func ShipHandler(sc *Client) {
	var pktHeader BBHeader
	for {
		err := sc.Process()
		if err == io.EOF {
			break
		} else if err != nil {
			// Error communicating with the client.
			log.Warn(err.Error())
			break
		}

		util.StructFromBytes(sc.Data()[:BBHeaderSize], &pktHeader)
		if config.DebugMode {
			fmt.Printf("SHIP: Got %v bytes from client:\n", pktHeader.Size)
			util.PrintPayload(sc.Data(), int(pktHeader.Size))
			fmt.Println()
		}

		switch pktHeader.Type {
		case LoginType:
			err = handleShipLogin(sc)
			sc.SendBlockList(blockPkt)
		case MenuSelectType:
			err = handleBlockSelection(sc)
		}

		if err != nil {
			log.Warn("Error in client communication: " + err.Error())
			return
		}
	}
}

func BlockHandler(sc *Client) {
	var pktHeader BBHeader
	for {
		err := sc.Process()
		if err == io.EOF {
			break
		} else if err != nil {
			// Error communicating with the client.
			log.Warn(err.Error())
			break
		}

		util.StructFromBytes(sc.Data()[:BBHeaderSize], &pktHeader)
		if config.DebugMode {
			fmt.Printf("BLOCK: Got %v bytes from client:\n", pktHeader.Size)
			util.PrintPayload(sc.Data(), int(pktHeader.Size))
			fmt.Println()
		}

		switch pktHeader.Type {
		case LoginType:
			err := handleShipLogin(sc)
			// TODO: Send lobby data
		}

		if err != nil {
			log.Warn("Error in client communication: " + err.Error())
			return
		}
	}
}

func InitShip() {
	// Precompute the block list packet since it's not going to change.
	numBlocks := config.NumBlocks
	ship := shipList[0]

	blockPkt = &BlockListPacket{
		Header:  BBHeader{Type: BlockListType, Flags: uint32(numBlocks)},
		Unknown: 0x08,
		Blocks:  make([]Block, numBlocks),
	}
	shipName := fmt.Sprintf("%d:%s", ship.id, ship.name)
	copy(blockPkt.ShipName[:], util.ConvertToUtf16(shipName))

	for i := 0; i < numBlocks; i++ {
		b := &blockPkt.Blocks[i]
		b.Unknown = 0x12
		// TODO: Teth sets this to (0xEFFFFFFF - block num)?
		b.BlockId = uint32(i + 1)
		blockName := fmt.Sprintf("BLOCK %02d", i+1)
		copy(b.BlockName[:], util.ConvertToUtf16(blockName))
	}
}
