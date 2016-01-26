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
	"github.com/dcrodman/archon/prs"
	"github.com/dcrodman/archon/util"
	"hash/crc32"
	"io/ioutil"
	"net"
	"os"
	"strconv"
)

const (
	// Client version string we're expecting during auth.
	ClientVersionString = "TethVer12510"
	// Maximum size of a block of parameter or guildcard data.
	MaxChunkSize = 0x6800
)

var (
	// Cached and parsed representation of the character port.
	charRedirectPort uint16

	// Connected ships. Each Ship's id corresponds to its position
	// in the array - 1.
	shipList []Ship = make([]Ship, 1)

	// Cached parameter data to avoid computing it every time.
	paramHeaderData []byte
	paramChunkData  map[int][]byte

	// Parameter files we're expecting. I still don't really know what they're
	// for yet, so emulating what I've seen others do.
	paramFiles = []string{
		"ItemMagEdit.prs",
		"ItemPMT.prs",
		"BattleParamEntry.dat",
		"BattleParamEntry_on.dat",
		"BattleParamEntry_lab.dat",
		"BattleParamEntry_lab_on.dat",
		"BattleParamEntry_ep4.dat",
		"BattleParamEntry_ep4_on.dat",
		"PlyLevelTbl.prs",
	}

	// Starting stats for any new character. The CharClass constants can be used
	// to index into this array to obtain the base stats for each class.
	BaseStats [12]CharacterStats

	// Id sent in the menu selection packet to tell the client
	// that the selection was made on the ship menu.
	ShipSelectionMenuId uint16 = 0x13
)

// Entry in the available ships lis on the ship selection menu.
type ShipMenuEntry struct {
	MenuId   uint16
	ShipId   uint32
	Padding  uint16
	Shipname [23]byte
}

// Struct for caching the parameter chunk data and header so
// that the param files aren't re-read every time.
type parameterEntry struct {
	Size     uint32
	Checksum uint32
	Offset   uint32
	Filename [0x40]uint8
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
		return nil, errors.New("Account does not exist for username: " + pktUername)
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

	// TODO: Account, hardware, and IP ban checks.
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
	if _, err := VerifyAccount(client); err != nil {
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
		stats := BaseStats[prev.Class]

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
func handleShipSelection(client *Client) error {
	var pkt MenuSelectionPacket
	util.StructFromBytes(client.Data(), &pkt)
	selectedShip := pkt.ItemId - 1
	if selectedShip < 0 || selectedShip >= uint32(len(shipList)) {
		return errors.New("Invalid ship selection: " + string(selectedShip))
	}
	s := &shipList[selectedShip]
	client.SendRedirect(s.port, s.ipAddr)
	return nil
}

// Create and initialize a new Login client so long as we're able
// to send the welcome packet to begin encryption.
func NewLoginClient(conn *net.TCPConn) (*Client, error) {
	var err error
	lc := NewClient(conn, BBHeaderSize)
	if lc.SendWelcome() != 0 {
		err = errors.New("Error sending welcome packet to: " + lc.IPAddr())
		lc = nil
	}
	return lc, err
}

// Login sub-server definition.
type LoginServer struct{}

func (server LoginServer) Name() string { return "LOGIN" }

func (server LoginServer) Port() string { return config.LoginPort }

// Load the PSOBB parameter files, build the parameter header,
// and init/cache the param file chunks for the EB packets.
func (server LoginServer) loadParameterFiles() {
	offset := 0
	var tmpChunkData []byte

	paramDir := config.ParametersDir
	fmt.Printf("Loading parameters from %s...\n", paramDir)
	for _, paramFile := range paramFiles {
		data, err := ioutil.ReadFile(paramDir + "/" + paramFile)
		if err != nil {
			fmt.Println("Error reading parameter file: " + err.Error())
			os.Exit(1)
		}
		fileSize := len(data)

		entry := new(parameterEntry)
		entry.Size = uint32(fileSize)
		entry.Checksum = crc32.ChecksumIEEE(data)
		entry.Offset = uint32(offset)
		copy(entry.Filename[:], []uint8(paramFile))

		offset += fileSize

		// We don't care what the actual entries are for the packet, so just append
		// the bytes to save us having to do the conversion every time.
		bytes, _ := util.BytesFromStruct(entry)
		paramHeaderData = append(paramHeaderData, bytes...)

		tmpChunkData = append(tmpChunkData, data...)
		fmt.Printf("%s (%v bytes, checksum: %v\n", paramFile, fileSize, entry.Checksum)
	}

	// Offset should at this point be the total size of the files
	// to send - break it all up into indexable chunks.
	paramChunkData = make(map[int][]byte)
	chunks := offset / MaxChunkSize
	for i := 0; i < chunks; i++ {
		dataOff := i * MaxChunkSize
		paramChunkData[i] = tmpChunkData[dataOff : dataOff+MaxChunkSize]
		offset -= MaxChunkSize
	}
	// Add any remaining data
	if offset > 0 {
		paramChunkData[chunks] = tmpChunkData[chunks*MaxChunkSize:]
	}
}

func (server *LoginServer) Init() {
	server.loadParameterFiles()

	// Load the base stats for creating new characters. Newserv, Sylverant, and Tethealla
	// all seem to rely on this file, so we'll do the same.
	statsFile, _ := os.Open("parameters/PlyLevelTbl.prs")
	compressed, err := ioutil.ReadAll(statsFile)
	if err != nil {
		fmt.Println("Error reading stats file: " + err.Error())
		os.Exit(1)
	}
	decompressedSize := prs.DecompressSize(compressed)
	decompressed := make([]byte, decompressedSize)
	prs.Decompress(compressed, decompressed)

	for i := 0; i < 12; i++ {
		util.StructFromBytes(decompressed[i*14:], &BaseStats[i])
	}

	charPort, _ := strconv.ParseUint(config.CharacterPort, 10, 16)
	charRedirectPort = uint16(charPort)
	fmt.Println()
}

func (server LoginServer) NewClient(conn *net.TCPConn) (*Client, error) {
	return NewLoginClient(conn)
}

func (server LoginServer) Handle(c *Client) error {
	var err error = nil
	var hdr BBHeader
	util.StructFromBytes(c.Data()[:BBHeaderSize], &hdr)

	switch hdr.Type {
	case LoginType:
		err = handleLogin(c)
	case DisconnectType:
		// Just wait until we recv 0 from the client to d/c.
		break
	default:
		log.Infof("Received unknown packet %x from %s", hdr.Type, c.IPAddr())
	}
	return err
}

// Character sub-server definition.
type CharacterServer struct{}

func (server CharacterServer) Name() string { return "CHARACTER" }

func (server CharacterServer) Port() string { return config.CharacterPort }

func (server *CharacterServer) Init() {}

func (server CharacterServer) NewClient(conn *net.TCPConn) (*Client, error) {
	return NewLoginClient(conn)
}

func (server CharacterServer) Handle(c *Client) error {
	var err error = nil
	var hdr BBHeader
	util.StructFromBytes(c.Data()[:BBHeaderSize], &hdr)

	switch hdr.Type {
	case LoginType:
		err = handleCharLogin(c)
	case DisconnectType:
		// Just wait until we recv 0 from the client to d/c.
		break
	case LoginOptionsRequestType:
		err = handleKeyConfig(c)
	case LoginCharPreviewReqType:
		err = handleCharacterSelect(c)
	case LoginChecksumType:
		// Everybody else seems to ignore this, so...
		c.SendChecksumAck(1)
	case LoginGuildcardReqType:
		err = handleGuildcardDataStart(c)
	case LoginGuildcardChunkReqType:
		handleGuildcardChunk(c)
	case LoginParameterHeaderReqType:
		c.SendParameterHeader(uint32(len(paramFiles)), paramHeaderData)
	case LoginParameterChunkReqType:
		var pkt BBHeader
		util.StructFromBytes(c.Data(), &pkt)
		c.SendParameterChunk(paramChunkData[int(pkt.Flags)], pkt.Flags)
	case LoginSetFlagType:
		var pkt SetFlagPacket
		util.StructFromBytes(c.Data(), &pkt)
		c.flag = pkt.Flag
	case LoginCharPreviewType:
		err = handleCharacterUpdate(c)
	case LoginMenuSelectType:
		err = handleShipSelection(c)
	default:
		log.Infof("Received unknown packet %x from %s", hdr.Type, c.IPAddr())
	}
	return err
}
