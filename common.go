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
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/dcrodman/archon/util"
)

// VerifyAccount performs all account verification tasks.
func VerifyAccount(client *Client) (*LoginPkt, error) {
	var loginPkt LoginPkt
	util.StructFromBytes(client.Data(), &loginPkt)

	pktUername := string(util.StripPadding(loginPkt.Username[:]))
	pktPassword := hashPassword(loginPkt.Password[:])

	var username, password string
	var isBanned, isActive bool
	row := config.DB().QueryRow("SELECT username, password, "+
		"guildcard, is_gm, is_banned, is_active, team_id from account_data "+
		"WHERE username = ? and password = ?", pktUername, pktPassword)
	err := row.Scan(&username, &password, &client.guildcard,
		&client.isGm, &isBanned, &isActive, &client.teamId)

	switch {
	// Check if we have a valid username/combination.
	case err == sql.ErrNoRows:
		// The same error is returned for invalid passwords as attempts to log in
		// with a nonexistent username as some measure of account security. Note
		// that if this is changed to query by username and add a password check,
		// the index on account_data will need to be modified.
		SendSecurity(client, BBLoginErrorPassword, 0, 0)
		return nil, errors.New("Account does not exist for username: " + pktUername)
	case err != nil:
		SendClientMessage(client, "Encountered an unexpected error while accessing the "+
			"database.\n\nPlease contact your server administrator.")
		log.Error(err.Error())
	case isBanned:
		SendSecurity(client, BBLoginErrorBanned, 0, 0)
		return nil, errors.New("Account banned: " + username)
	case !isActive:
		SendClientMessage(client, "Encountered an unexpected error while accessing the "+
			"database.\n\nPlease contact your server administrator.")
		return nil, errors.New("Account must be activated for username: " + username)
	}
	// Copy over the config, which should indicate how far they are in the login flow.
	util.StructFromBytes(loginPkt.Security[:], &client.config)

	// TODO: Account, hardware, and IP ban checks.
	return &loginPkt, nil
}

// Passwords are stored as sha256 hashes, so hash what the client sent us for the query.
func hashPassword(password []byte) string {
	hasher := sha256.New()
	hasher.Write(util.StripPadding(password))
	return hex.EncodeToString(hasher.Sum(nil)[:])
}

// SendClientMessage is used for error messages to the client, usually used before disconnecting.
func SendClientMessage(client *Client, message string) error {
	pkt := &LoginClientMessagePacket{
		Header: BBHeader{Type: LoginClientMessageType},
		// English? Tethealla sets this.
		Language: 0x00450009,
		Message:  util.ConvertToUtf16(message),
	}
	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Client Message Packet")
	}
	return client.SendEncrypted(data, size)
}

// SendWelcome transmits the welcome packet to a client with the copyright message and encryption vectors.
func SendWelcome(client *Client) error {
	pkt := new(WelcomePkt)
	pkt.Header.Type = LoginWelcomeType
	pkt.Header.Size = 0xC8
	copy(pkt.Copyright[:], LoginCopyright)
	copy(pkt.ClientVector[:], client.ClientVector())
	copy(pkt.ServerVector[:], client.ServerVector())

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Welcome Packet")
		util.PrintPayload(data, size)
		fmt.Println()
	}
	return client.SendEncrypted(data, size)
}

// SendSecurity transmits initialization packet with information about the user's
// authentication status. This is used by everything except the patch server.
func SendSecurity(client *Client, errorCode BBLoginError, guildcard uint32, teamId uint32) error {
	// Constants set according to how Newserv does it.
	pkt := &SecurityPacket{
		Header:       BBHeader{Type: LoginSecurityType},
		ErrorCode:    uint32(errorCode),
		PlayerTag:    0x00010000,
		Guildcard:    guildcard,
		TeamId:       teamId,
		Config:       &client.config,
		Capabilities: 0x00000102,
	}

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Security Packet")
	}
	return client.SendEncrypted(data, size)
}

// SendRedirect sends the client the address of the next server to which they should connect.
func SendRedirect(client *Client, ipAddr []byte, port uint16) error {
	pkt := new(RedirectPacket)
	pkt.Header.Type = RedirectType
	pkt.Port = port
	copy(pkt.IPAddr[:], ipAddr)

	data, size := util.BytesFromStruct(pkt)
	if config.DebugMode {
		fmt.Println("Sending Redirect Packet")
	}
	return client.SendEncrypted(data, size)
}
