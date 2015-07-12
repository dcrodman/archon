/*
* Archon Login Server
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
*
* LOGIN server logic.
 */

package login_server

import (
	"errors"
	"fmt"
	"libarchon/logger"
	"libarchon/util"
)

const ClientVersionString = "TethVer12510"

func handleLogin(client *LoginClient) error {
	loginPkt, err := verifyAccount(client)
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

	config := GetConfig()
	SendSecurity(client, BBLoginErrorNone, client.guildcard, client.teamId)
	SendRedirect(client, uint16(config.RedirectPort()), config.HostnameBytes())
	return nil
}

// Process packets sent to the LOGIN port by sending them off to another handler or by
// taking some brief action.
func processLoginPacket(client *LoginClient) error {
	var pktHeader BBPktHeader
	c := client.Client()
	util.StructFromBytes(c.Data()[:BBHeaderSize], &pktHeader)

	if GetConfig().DebugMode {
		fmt.Printf("Got %v bytes from client:\n", pktHeader.Size)
		util.PrintPayload(client.Data(), int(pktHeader.Size))
		fmt.Println()
	}

	var err error = nil
	switch pktHeader.Type {
	case LoginType:
		err = handleLogin(client)
	case DisconnectType:
		// Just wait until we recv 0 from the client to d/c.
		break
	default:
		msg := fmt.Sprintf("Received unknown packet %x from %s", pktHeader.Type, c.IPAddr())
		log.Info(msg, logger.MediumPriority)
	}
	return err
}
