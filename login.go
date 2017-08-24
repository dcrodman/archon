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
 */
package main

import (
	"errors"
	crypto "github.com/dcrodman/archon/encryption"
	"github.com/dcrodman/archon/util"
	"net"
	"strconv"
)

// Struct for caching the parameter chunk data and header so
// that the param files aren't re-read every time.
type parameterEntry struct {
	Size     uint32
	Checksum uint32
	Offset   uint32
	Filename [0x40]uint8
}

// Create and initialize a new Login client so long as we're able
// to send the welcome packet to begin encryption.
func NewLoginClient(conn *net.TCPConn) (*Client, error) {
	var err error
	cCrypt := crypto.NewBBCrypt()
	sCrypt := crypto.NewBBCrypt()
	lc := NewClient(conn, BBHeaderSize, cCrypt, sCrypt)
	if SendWelcome(lc) != nil {
		err = errors.New("Error sending welcome packet to: " + lc.IPAddr())
		lc = nil
	}
	return lc, err
}

// Login sub-server definition.
type LoginServer struct {
	// Cached and parsed representation of the character port.
	charRedirectPort uint16
}

func (server LoginServer) Name() string { return "LOGIN" }

func (server LoginServer) Port() string { return config.LoginPort }

func (server *LoginServer) Init() error {
	charPort, _ := strconv.ParseUint(config.CharacterPort, 10, 16)
	server.charRedirectPort = uint16(charPort)
	return nil
}

func (server *LoginServer) NewClient(conn *net.TCPConn) (*Client, error) {
	return NewLoginClient(conn)
}

func (server *LoginServer) Handle(c *Client) error {
	var hdr BBHeader
	util.StructFromBytes(c.Data()[:BBHeaderSize], &hdr)

	var err error
	switch hdr.Type {
	case LoginType:
		err = server.HandleLogin(c)
	case DisconnectType:
		// Just wait until we recv 0 from the client to d/c.
		break
	default:
		log.Infof("Received unknown packet %x from %s", hdr.Type, c.IPAddr())
	}
	return err
}

func (server *LoginServer) HandleLogin(client *Client) error {
	loginPkt, err := VerifyAccount(client)
	if err != nil {
		return err
	}

	// The first time we receive this packet the client will have included the
	// version string in the security data; check it.
	if ClientVersionString != string(util.StripPadding(loginPkt.Security[:])) {
		SendSecurity(client, BBLoginErrorPatch, 0, 0)
		return errors.New("Incorrect version string")
	}

	// Newserv sets this field when the client first connects. I think this is
	// used to indicate that the client has made it through the LOGIN server,
	// but for now we'll just set it and leave it alone.
	client.config.Magic = 0x48615467

	ipAddr := config.BroadcastIP()
	SendSecurity(client, BBLoginErrorNone, client.guildcard, client.teamId)
	return SendRedirect(client, ipAddr[:], server.charRedirectPort)
}
