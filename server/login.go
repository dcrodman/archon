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
package server

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/dcrodman/archon/server/util"
	"hash/crc32"
	"io"
	"net"
	"strconv"
)

const ClientVersionString = "TethVer12510"

var (
	// Cached and parsed representation of the character port.
	charRedirectPort uint16

	// Connected ships. Each Ship's id corresponds to its position
	// in the array - 1.
	shipList []Ship = make([]Ship, 1)

	// Cached parameter data to avoid computing it every time.
	paramHeaderData []byte
	paramChunkData  map[int][]byte
)

// Struct for representing available ships in the ship selection menu.
type ShipEntry struct {
	Unknown  uint16
	Id       uint32
	Padding  uint16
	Shipname [23]byte
}

// Handle account verification tasks.
func VerifyAccount(client *Client) (*LoginPkt, error) {
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
		client.SendSecurity(BBLoginErrorPassword, 0, 0)
		return nil, errors.New("Account does not exist for username: " + username)
	// Database error?
	case err != nil:
		client.SendClientMessage("Encountered an unexpected error while accessing the " +
			"database.\n\nPlease contact your server administrator.")
		log.Error(err.Error())
		return nil, err
	// Is the account banned?
	case isBanned:
		client.SendSecurity(BBLoginErrorBanned, 0, 0)
		return nil, errors.New("Account banned: " + username)
	// Has the account been activated?
	case !isActive:
		client.SendClientMessage("Encountered an unexpected error while accessing the " +
			"database.\n\nPlease contact your server administrator.")
		return nil, errors.New("Account must be activated for username: " + username)
	}
	// Copy over the config, which should indicate how far they are in the login flow.
	util.StructFromBytes(loginPkt.Security[:], &client.config)

	// TODO: Hardware ban check.
	return &loginPkt, nil
}

// Handle the initial login sent to the Login port.
func handleLogin(client *Client) error {
	loginPkt, err := VerifyAccount(client)
	if err != nil {
		return err
	}
	// The first time we receive this packet the client will have included the
	// version string in the security data; check it.
	if ClientVersionString != string(util.StripPadding(loginPkt.Security[:])) {
		client.SendSecurity(BBLoginErrorPatch, 0, 0)
		return errors.New("Incorrect version string")
	}
	// Newserv sets this field when the client first connects. I think this is
	// used to indicate that the client has made it through the LOGIN server,
	// but for now we'll just set it and leave it alone.
	client.config.Magic = 0x48615467

	client.SendSecurity(BBLoginErrorNone, client.guildcard, client.teamId)
	client.SendRedirect(charRedirectPort, config.HostnameBytes())
	return nil
}

// Handle initial login sent to the character port.
func handleCharLogin(client *Client) error {
	_, err := VerifyAccount(client)
	if err != nil {
		return err
	}
	client.SendSecurity(BBLoginErrorNone, client.guildcard, client.teamId)
	if client.config.CharSelected == 1 {
		client.SendTimestamp()
		client.SendShipList(shipList)
		client.SendScrollMessage()
	}
	return nil
}

// Handle the options request - load key config and other option data from the
// datebase or provide defaults for new accounts.
func handleKeyConfig(client *Client) error {
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
	client.SendOptions(optionData)
	return nil
}

// Handle the character select/preview request. Will either return information
// about a character given a particular slot in via 0xE5 response or ack the
// selection with an 0xE4 (also used for an empty slot).
func handleCharacterSelect(client *Client) error {
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
		client.SendCharacterAck(pkt.Slot, 2)
		return nil
	} else if err != nil {
		log.Error(err.Error())
		return err
	}

	if pkt.Selecting == 0x01 {
		// They've selected a character from the menu.
		client.config.CharSelected = 1
		client.config.SlotNum = uint8(pkt.Slot)
		client.SendSecurity(BBLoginErrorNone, client.guildcard, client.teamId)
		client.SendCharacterAck(pkt.Slot, 1)
	} else {
		// They have a character in that slot; send the character preview.
		copy(prev.GuildcardStr[:], gc[:])
		copy(prev.Name[:], name[:])
		client.SendCharacterPreview(prev)
	}
	return nil
}

// Load the player's saved guildcards, build the chunk data, and
// send the chunk header.
func handleGuildcardDataStart(client *Client) error {
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

	client.SendGuildcardHeader(checksum, client.gcDataSize)
	return nil
}

// Send another chunk of the client's guildcard data.
func handleGuildcardChunk(client *Client) {
	var chunkReq GuildcardChunkReqPacket
	util.StructFromBytes(client.Data(), &chunkReq)
	if chunkReq.Continue != 0x01 {
		// Cancelled sending guildcard chunks.
		return
	}
	client.SendGuildcardChunk(chunkReq.ChunkRequested)
}

// Create or update a character in a slot.
func handleCharacterUpdate(client *Client) error {
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
	client.SendCharacterAck(charPkt.Slot, 0)
	return nil
}

// Player selected one of the items on the ship select screen.
func handleMenuSelect(client *Client) {
	var pkt ShipMenuSelectionPacket
	util.StructFromBytes(client.Data(), &pkt)
	s := &shipList[pkt.Item-1]
	client.SendRedirect(s.port, s.ipAddr)
}

// Create and initialize a new struct to hold client information.
func NewLoginClient(conn *net.TCPConn) (*Client, error) {
	var err error
	lc := NewClient(conn, BBHeaderSize)
	if lc.SendWelcome() != 0 {
		err = errors.New("Error sending welcome packet to: " + lc.IPAddr())
		lc = nil
	}
	return lc, err
}

// Process packets sent to the LOGIN port by sending them off to another handler or by
// taking some brief action.
func LoginHandler(lc *Client) {
	for {
		err := lc.Process()
		if err == io.EOF {
			break
		} else if err != nil {
			// Error communicating with the client.
			log.Warn(err.Error())
			break
		}

		var pktHeader BBPktHeader
		util.StructFromBytes(lc.Data()[:BBHeaderSize], &pktHeader)

		if config.DebugMode {
			fmt.Printf("LOGIN: Got %v bytes from client:\n", pktHeader.Size)
			util.PrintPayload(lc.Data(), int(pktHeader.Size))
			fmt.Println()
		}

		switch pktHeader.Type {
		case LoginType:
			err = handleLogin(lc)
		case DisconnectType:
			// Just wait until we recv 0 from the client to d/c.
			break
		default:
			log.Info("Received unknown packet %x from %s", pktHeader.Type, lc.IPAddr())
		}
		if err != nil {
			log.Warn("Error in client communication: " + err.Error())
			return
		}
	}
}

// Process packets sent to the CHARACTER port by sending them off to another
// handler or by taking some brief action.
func CharacterHandler(lc *Client) {
	for {
		err := lc.Process()
		if err == io.EOF {
			break
		} else if err != nil {
			// Error communicating with the client.
			log.Warn(err.Error())
			break
		}

		var pktHeader BBPktHeader
		util.StructFromBytes(lc.Data(), &pktHeader)

		if config.DebugMode {
			fmt.Printf("CHAR: Got %v bytes from client:\n", pktHeader.Size)
			util.PrintPayload(lc.Data(), int(pktHeader.Size))
			fmt.Println()
		}

		switch pktHeader.Type {
		case LoginType:
			err = handleCharLogin(lc)
		case DisconnectType:
			// Just wait until we recv 0 from the client to d/c.
			break
		case LoginOptionsRequestType:
			err = handleKeyConfig(lc)
		case LoginCharPreviewReqType:
			err = handleCharacterSelect(lc)
		case LoginChecksumType:
			// Everybody else seems to ignore this, so...
			lc.SendChecksumAck(1)
		case LoginGuildcardReqType:
			err = handleGuildcardDataStart(lc)
		case LoginGuildcardChunkReqType:
			handleGuildcardChunk(lc)
		case LoginParameterHeaderReqType:
			lc.SendParameterHeader(uint32(len(paramFiles)), paramHeaderData)
		case LoginParameterChunkReqType:
			var pkt BBPktHeader
			util.StructFromBytes(lc.Data(), &pkt)
			lc.SendParameterChunk(paramChunkData[int(pkt.Flags)], pkt.Flags)
		case LoginSetFlagType:
			var pkt SetFlagPacket
			util.StructFromBytes(lc.Data(), &pkt)
			lc.flag = pkt.Flag
		case LoginCharPreviewType:
			err = handleCharacterUpdate(lc)
		case LoginMenuSelectType:
			handleMenuSelect(lc)
		default:
			log.Info("Received unknown packet %x from %s", pktHeader.Type, lc.IPAddr())
		}

		if err != nil {
			log.Warn("Error in client communication: " + err.Error())
			return
		}
	}
}

func InitLogin() {
	loadParameterFiles()
	loadBaseStats()

	charPort, _ := strconv.ParseUint(config.CharacterPort, 10, 16)
	charRedirectPort = uint16(charPort)

	// Create our ship entry for the built-in ship server. Any other connected
	// ships will be added to this list by the shipgate, if it's enabled.
	s := &shipList[0]
	s.id = 1
	s.ipAddr = config.HostnameBytes()
	port, _ := strconv.ParseUint(config.ShipPort, 10, 16)
	s.port = uint16(port)
	copy(s.name[:], util.ConvertToUtf16(config.ShipName))
}
