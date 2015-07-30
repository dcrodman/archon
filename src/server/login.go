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

// Login and Character server logic.
package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/crc32"
	"net/http"
	"server/util"
	"strconv"
)

const ClientVersionString = "TethVer12510"

var (
	// Cached and parsed representation of the character port.
	charRedirectPort uint16

	defaultShip ShipEntry
	shipList    []ShipEntry
	// shipListMutex sync.RWMutex

	// Cached parameter data to avoid computing it every time.
	paramHeaderData []byte
	paramChunkData  map[int][]byte
)

// Possible character classes as defined by the game.
type CharClass uint8

const (
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

// Struct for caching the parameter chunk data and header so
// that the param files aren't re-read every time.
type parameterEntry struct {
	Size     uint32
	Checksum uint32
	Offset   uint32
	Filename [0x40]uint8
}

// Per-player friend guildcard entries.
type GuildcardEntry struct {
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

// Per-player guildcard data chunk.
type GuildcardData struct {
	Unknown  [0x114]uint8
	Blocked  [0x1DE8]uint8 //This should be a struct once implemented
	Unknown2 [0x78]uint8
	Entries  [104]GuildcardEntry
	Unknown3 [0x1BC]uint8
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
	SectionId      byte
	Class          byte
	V2flags        byte
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

// Per-character stats.
type CharacterStats struct {
	ATP uint16
	MST uint16
	EVP uint16
	HP  uint16
	DFP uint16
	ATA uint16
	LCK uint16
}

// Struct for holding client-specific data.
type LoginClient struct {
	c         *PSOClient
	guildcard uint32
	teamId    uint32
	isGm      bool

	gcData     []byte
	gcDataSize uint16
	config     ClientConfig
	flag       uint32
}

func (lc *LoginClient) IPAddr() string { return lc.c.IPAddr() }
func (lc *LoginClient) Client() Client { return lc.c }
func (lc *LoginClient) Data() []byte   { return lc.c.Data() }

// Struct for representing available ships in the ship selection menu.
type ShipEntry struct {
	Unknown  uint16 // Always 0x12
	Id       uint32
	Padding  uint16
	Shipname [23]byte
}

// Handle the initial login sent to the Login port.
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

	SendSecurity(client, BBLoginErrorNone, client.guildcard, client.teamId)
	SendRedirect(client, charRedirectPort, config.HostnameBytes())
	return nil
}

// Handle initial login sent to the character port.
func handleCharLogin(client *LoginClient) error {
	_, err := verifyAccount(client)
	if err != nil {
		return err
	}
	SendSecurity(client, BBLoginErrorNone, client.guildcard, client.teamId)
	if client.config.CharSelected == 1 {
		SendTimestamp(client)
		SendShipList(client, shipList)
		SendScrollMessage(client)
	}
	return nil
}

// Handle account verification tasks common to both the login and character servers.
func verifyAccount(client *LoginClient) (*LoginPkt, error) {
	var loginPkt LoginPkt
	util.StructFromBytes(client.Data(), &loginPkt)

	// Passwords are stored as sha256 hashes, so hash what the client sent us for the query.
	hasher := sha256.New()
	hasher.Write(util.StripPadding(loginPkt.Password[:]))
	pktUername := string(util.StripPadding(loginPkt.Username[:]))
	pktPassword := hex.EncodeToString(hasher.Sum(nil)[:])

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
		return nil, errors.New("Account does not exist for username: " + username)
	// Database error?
	case err != nil:
		SendClientMessage(client, "Encountered an unexpected error while accessing the "+
			"database.\n\nPlease contact your server administrator.")
		log.Error(err.Error())
		return nil, err
	// Is the account banned?
	case isBanned:
		SendSecurity(client, BBLoginErrorBanned, 0, 0)
		return nil, errors.New("Account banned: " + username)
	// Has the account been activated?
	case !isActive:
		SendClientMessage(client, "Encountered an unexpected error while accessing the "+
			"database.\n\nPlease contact your server administrator.")
		return nil, errors.New("Account must be activated for username: " + username)
	}
	// Copy over the config, which should indicate how far they are in the login flow.
	util.StructFromBytes(loginPkt.Security[:], &client.config)

	// TODO: Hardware ban check.
	return &loginPkt, nil
}

// Handle the options request - load key config and other option data from the
// datebase or provide defaults for new accounts.
func handleKeyConfig(client *LoginClient) error {
	optionData := make([]byte, 420)
	archondb := config.DB()

	row := archondb.QueryRow(
		"SELECT key_config from player_options where guildcard = ?", client.guildcard)
	err := row.Scan(&optionData)
	if err == sql.ErrNoRows {
		// We don't have any saved key config - give them the defaults.
		copy(optionData[:420], baseKeyConfig[:])
		_, err = archondb.Exec("INSERT INTO player_options (guildcard, key_config) "+
			" VALUES (?, ?)", client.guildcard, optionData[:420])
	}
	if err != nil {
		log.Error(err.Error())
		return err
	}
	SendOptions(client, optionData)
	return nil
}

// Handle the character select/preview request. Will either return information
// about a character given a particular slot in via 0xE5 response or ack the
// selection with an 0xE4 (also used for an empty slot).
func handleCharacterSelect(client *LoginClient) error {
	var pkt CharSelectionPacket
	util.StructFromBytes(client.Data(), &pkt)
	prev := new(CharacterPreview)

	// Character preview request.
	archondb := config.DB()
	var gc, name []uint8
	row := archondb.QueryRow("SELECT experience, level, guildcard_str, "+
		" name_color, name_color_chksm, model, section_id, char_class, "+
		"v2_flags, version, v1_flags, costume, skin, face, head, hair, "+
		"hair_red, hair_green, hair_blue, proportion_x, proportion_y, "+
		"name, playtime FROM characters WHERE guildcard = ? AND slot_num = ?",
		client.guildcard, pkt.Slot)
	err := row.Scan(&prev.Experience, &prev.Level, &gc,
		&prev.NameColor, &prev.NameColorChksm, &prev.Model, &prev.SectionId,
		&prev.Class, &prev.V2flags, &prev.Version, &prev.V1Flags, &prev.Costume,
		&prev.Skin, &prev.Face, &prev.Head, &prev.Hair, &prev.HairRed,
		&prev.HairGreen, &prev.HairBlue, &prev.PropX, &prev.PropY,
		&name, &prev.Playtime)

	if err == sql.ErrNoRows {
		// We don't have a character for this slot.
		SendCharacterAck(client, pkt.Slot, 2)
		return nil
	} else if err != nil {
		log.Error(err.Error())
		return err
	}

	if pkt.Selecting == 0x01 {
		// They've selected a character from the menu.
		client.config.CharSelected = 1
		client.config.SlotNum = uint8(pkt.Slot)
		SendSecurity(client, BBLoginErrorNone, client.guildcard, client.teamId)
		SendCharacterAck(client, pkt.Slot, 1)
	} else {
		// They have a character in that slot; send the character preview.
		copy(prev.GuildcardStr[:], gc[:])
		copy(prev.Name[:], name[:])
		SendCharacterPreview(client, prev)
	}
	return nil
}

// Load the player's saved guildcards, build the chunk data, and
// send the chunk header.
func handleGuildcardDataStart(client *LoginClient) error {
	archondb := config.DB()
	rows, err := archondb.Query(
		"SELECT friend_gc, name, team_name, description, language, "+
			"section_id, char_class, comment FROM guildcard_entries "+
			"WHERE guildcard = ?", client.guildcard)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	defer rows.Close()
	gcData := new(GuildcardData)

	// Maximum of 140 entries can be sent.
	for i := 0; rows.Next() && i < 140; i++ {
		// TODO: This may not actually work yet, but I haven't gotten to
		// figuring out how this is used yet.
		var name, teamName, desc, comment []uint8
		entry := &gcData.Entries[i]
		err = rows.Scan(&entry.Guildcard, &name, &teamName, &desc,
			&entry.Language, &entry.SectionID, &entry.CharClass, &comment)
		if err != nil {
			log.Error(err.Error())
			return err
		}
	}
	var size int
	client.gcData, size = util.BytesFromStruct(gcData)
	checksum := crc32.ChecksumIEEE(client.gcData)
	client.gcDataSize = uint16(size)

	SendGuildcardHeader(client, checksum, client.gcDataSize)
	return nil
}

// Send another chunk of the client's guildcard data.
func handleGuildcardChunk(client *LoginClient) {
	var chunkReq GuildcardChunkReqPacket
	util.StructFromBytes(client.Data(), &chunkReq)
	if chunkReq.Continue != 0x01 {
		// Cancelled sending guildcard chunks.
		return
	}
	SendGuildcardChunk(client, chunkReq.ChunkRequested)
}

// Create or update a character in a slot.
func handleCharacterUpdate(client *LoginClient) error {
	var charPkt CharPreviewPacket
	charPkt.Character = new(CharacterPreview)
	util.StructFromBytes(client.Data(), &charPkt)
	prev := charPkt.Character

	archonDB := config.DB()
	if client.flag == 0x02 {
		// Player is using the dressing room; update the character. Messy
		// query, but unavoidable if we don't want to be stuck with blobs.
		_, err := archonDB.Exec("UPDATE characters SET name_color=?, model=?, "+
			"name_color_chksm=?, section_id=?, char_class=?, costume=?, skin=?, "+
			"head=?, hair_red=?, hair_green=?, hair_blue,=? proportion_x=?, "+
			"proportion_y=?, name=? WHERE guildcard = ? AND slot_num = ?",
			prev.NameColor, prev.Model, prev.NameColorChksm, prev.SectionId,
			prev.Class, prev.Costume, prev.Skin, prev.Head, prev.HairRed,
			prev.HairGreen, prev.HairBlue, prev.Name[:], prev.PropX, prev.PropY,
			client.guildcard, charPkt.Slot)
		if err != nil {
			log.Error(err.Error())
			return err
		}
	} else {
		// Delete a character if it already exists.
		_, err := archonDB.Exec("DELETE FROM characters WHERE "+
			"guildcard = ? AND slot_num = ?", client.guildcard, charPkt.Slot)
		if err != nil {
			log.Error(err.Error())
			return err
		}
		// Grab our base stats for this character class.
		stats := baseStats[prev.Class]

		// TODO: Set up the default inventory and techniques.
		meseta := 300

		/* TODO: Add the rest of these.
		--unsigned char keyConfig[232]; // 0x3E8 - 0x4CF;
		--techniques blob,
		--options blob,
		*/

		// Create the new character.
		_, err = archonDB.Exec("INSERT INTO characters VALUES (?, ?, 0, 1, ?, "+
			"?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?, "+
			"?, ?, ?, ?, ?, ?, 0, 0)", client.guildcard, charPkt.Slot,
			prev.GuildcardStr[:], prev.NameColor, prev.Model, prev.NameColorChksm,
			prev.SectionId, prev.Class, prev.V2flags, prev.Version, prev.V1Flags,
			prev.Costume, prev.Skin, prev.Face, prev.Head, prev.Hair, prev.HairRed,
			prev.HairGreen, prev.HairBlue, prev.PropX, prev.PropY, prev.Name[:],
			stats.ATP, stats.MST, stats.EVP, stats.HP, stats.DFP, stats.ATA,
			stats.LCK, meseta)
		if err != nil {
			log.Error(err.Error())
			return err
		}
	}

	// Send the security packet with the updated state and slot number so that
	// we know a character has been selected.
	client.config.CharSelected = 1
	client.config.SlotNum = uint8(charPkt.Slot)
	SendSecurity(client, BBLoginErrorNone, client.guildcard, client.teamId)

	SendCharacterAck(client, charPkt.Slot, 0)
	return nil
}

// Player selected one of the items on the ship select screen.
func handleMenuSelect(client *LoginClient) {

}

// Create and initialize a new struct to hold client information.
func newLoginClient(c *PSOClient) (*LoginClient, error) {
	var err error
	loginClient := &LoginClient{c: c}
	if SendLoginWelcome(loginClient) != 0 {
		err = errors.New("Error sending welcome packet to: " + loginClient.IPAddr())
		loginClient = nil
	}
	return loginClient, err
}

// Return a JSON string to the client with the name, hostname, port,
// and player count.
func handleShipCountRequest(w http.ResponseWriter, req *http.Request) {

}

// Process packets sent to the LOGIN port by sending them off to another handler or by
// taking some brief action.
func processLoginPacket(client *LoginClient) error {
	// var pktHeader BBPktHeader
	// c := Client()
	// util.StructFromBytes(c.Data()[:BBHeaderSize], &pktHeader)

	// if config.DebugMode {
	// 	fmt.Printf("Got %v bytes from client:\n", pktHeader.Size)
	// 	util.PrintPayload(client.Data(), int(pktHeader.Size))
	// 	fmt.Println()
	// }

	// var err error
	// switch pktHeader.Type {
	// case LoginType:
	// 	err = handleLogin(client)
	// case DisconnectType:
	// 	// Just wait until we recv 0 from the client to d/c.
	// 	break
	// default:
	// 	log.Info("Received unknown packet %x from %s", pktHeader.Type, c.IPAddr())
	// }
	// return err
	return nil
}

// Process packets sent to the CHARACTER port by sending them off to another
// handler or by taking some brief action.
func processCharacterPacket(client *LoginClient) error {
	var pktHeader BBPktHeader
	util.StructFromBytes(client.Data(), &pktHeader)

	if config.DebugMode {
		fmt.Printf("Got %v bytes from client:\n", pktHeader.Size)
		util.PrintPayload(client.Data(), int(pktHeader.Size))
		fmt.Println()
	}

	var err error
	switch pktHeader.Type {
	case LoginType:
		err = handleCharLogin(client)
	case DisconnectType:
		// Just wait until we recv 0 from the client to d/c.
		break
	case LoginOptionsRequestType:
		err = handleKeyConfig(client)
	case LoginCharPreviewReqType:
		err = handleCharacterSelect(client)
	case LoginChecksumType:
		// Everybody else seems to ignore this, so...
		SendChecksumAck(client, 1)
	case LoginGuildcardReqType:
		err = handleGuildcardDataStart(client)
	case LoginGuildcardChunkReqType:
		handleGuildcardChunk(client)
	case LoginParameterHeaderReqType:
		SendParameterHeader(client, uint32(len(paramFiles)), paramHeaderData)
	case LoginParameterChunkReqType:
		var pkt BBPktHeader
		util.StructFromBytes(client.Data(), &pkt)
		SendParameterChunk(client, paramChunkData[int(pkt.Flags)], pkt.Flags)
	case LoginSetFlagType:
		var pkt SetFlagPacket
		util.StructFromBytes(client.Data(), &pkt)
		client.flag = pkt.Flag
	case LoginCharPreviewType:
		err = handleCharacterUpdate(client)
	case LoginMenuSelectType:
		handleMenuSelect(client)
	default:
		log.Info("Received unknown packet %x from %s", pktHeader.Type, client.IPAddr())
	}
	return err
}

func InitLogin() {
	loadParameterFiles()
	loadBaseStats()

	// Create our "No Ships" item to indicate the absence of any ship servers.
	defaultShip.Unknown = 0x12
	defaultShip.Id = 1
	copy(defaultShip.Shipname[:], util.ConvertToUtf16("No Ships"))
	shipList = append(shipList, defaultShip)

	// Open up our web port for retrieving player counts. If we're in debug mode, add a path
	// for dumping pprof output containing the stack traces of all running goroutines.
	// TODO: Move this
	http.HandleFunc("/list", handleShipCountRequest)
	go http.ListenAndServe(":"+config.WebPort, nil)

	charPort, _ := strconv.ParseUint(config.CharacterPort, 10, 16)
	charRedirectPort = uint16(charPort)
}
