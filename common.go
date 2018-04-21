package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/dcrodman/archon/util"
)

// CharClass is an enumeration of the possible character classes.
type CharClass uint8

const (
	// Possible character classes as defined by the game.
	Humar     CharClass = 0x00
	Hunewearl           = 0x01
	Hucast              = 0x02
	Ramar               = 0x03
	Racast              = 0x04
	Racaseal            = 0x05
	Fomarl              = 0x06
	Fonewm              = 0x07
	Fonewearl           = 0x08
	Hucaseal            = 0x09
	Fomar               = 0x0A
	Ramarl              = 0x0B
)

// Per-player guildcard data chunk.
type GuildcardData struct {
	Unknown  [0x114]uint8
	Blocked  [0x1DE8]uint8 //This should be a struct once implemented
	Unknown2 [0x78]uint8
	Entries  [104]GuildcardDataEntry
	Unknown3 [0x1BC]uint8
}

// Per-player friend guildcard entries.
type GuildcardDataEntry struct {
	Guildcard   uint32
	Name        [24]uint16
	TeamName    [16]uint16
	Description [88]uint16
	Reserved    uint8
	Language    uint8
	SectionID   uint8
	CharClass   uint8
	padding     uint32
	Comment     [88]uint16
}

// Struct used by Character Info packet.
type CharacterPreview struct {
	Experience     uint32
	Level          uint32
	GuildcardStr   [16]byte
	Unknown        [2]uint32
	NameColor      uint32
	Model          byte
	Padding        [15]byte
	NameColorChksm uint32
	SectionID      byte
	Class          byte
	V2Flags        byte
	Version        byte
	V1Flags        uint32
	Costume        uint16
	Skin           uint16
	Face           uint16
	Head           uint16
	Hair           uint16
	HairRed        uint16
	HairGreen      uint16
	HairBlue       uint16
	PropX          float32
	PropY          float32
	Name           [24]uint8
	Playtime       uint32
}

// Copyright message expected by the client when connecting.
var LoginCopyright = []byte("Phantasy Star Online Blue Burst Game Server. Copyright 1999-2004 SONICTEAM.")

// VerifyAccount performs all account verification tasks.
func VerifyAccount(client *Client) (*LoginPkt, error) {
	var loginPkt LoginPkt
	util.StructFromBytes(client.Data(), &loginPkt)

	pktUsername := string(util.StripPadding(loginPkt.Username[:]))
	pktPassword := hashPassword(loginPkt.Password[:])
	account, err := database.FindAccount(pktUsername)

	switch {
	case err != nil:
		SendClientMessage(client, "Encountered an unexpected error while accessing the "+
			"database.\n\nPlease contact your server administrator.")
		log.Error(err.Error())
	case account == nil:
	case account.Password != pktPassword:
		// The same error is returned for invalid passwords as attempts to log in
		// with a nonexistent username as some measure of account security.
		SendSecurity(client, BBLoginErrorPassword, 0, 0)
		return nil, errors.New("Account does not exist for username: " + pktUsername)
	case !account.Active:
		SendClientMessage(client, "Encountered an unexpected error while accessing the "+
			"database.\n\nPlease contact your server administrator.")
		return nil, errors.New("Account must be activated for username: " + pktUsername)
	case account.Banned:
		SendSecurity(client, BBLoginErrorBanned, 0, 0)
		return nil, errors.New("Account banned: " + pktUsername)
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
	DebugLog("Sending Client Message Packet")
	return EncryptAndSend(client, pkt)
}

// SendWelcome transmits the welcome packet to a client with the copyright message and encryption vectors.
func SendWelcome(client *Client) error {
	pkt := new(WelcomePkt)
	pkt.Header.Type = LoginWelcomeType
	pkt.Header.Size = 0xC8
	copy(pkt.Copyright[:], LoginCopyright)
	copy(pkt.ClientVector[:], client.ClientVector())
	copy(pkt.ServerVector[:], client.ServerVector())

	DebugLog("Sending Welcome Packet")
	data, size := util.BytesFromStruct(pkt)
	return client.SendRaw(data, size)
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
	DebugLog("Sending Security Packet")
	return EncryptAndSend(client, pkt)
}

// SendRedirect sends the client the address of the next server to which they should connect.
func SendRedirect(client *Client, ipAddr []byte, port uint16) error {
	pkt := new(RedirectPacket)
	pkt.Header.Type = RedirectType
	pkt.Port = port
	copy(pkt.IPAddr[:], ipAddr)

	DebugLog("Sending Redirect Packet")
	return EncryptAndSend(client, pkt)
}

// EncryptAndSend will encode the packet and let Client encrypt and transmit it.
func EncryptAndSend(client *Client, pkt interface{}) error {
	data, size := util.BytesFromStruct(pkt)
	return client.SendEncrypted(data, size)
}
