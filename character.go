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
	"database/sql"
	"errors"
	"fmt"
	"github.com/dcrodman/archon/prs"
	"github.com/dcrodman/archon/util"
	"hash/crc32"
	"io/ioutil"
	"net"
	"os"
	"syscall"
	"time"
)

const (
	// Client version string we're expecting during auth.
	ClientVersionString = "TethVer12510"
	// Maximum size of a block of parameter or guildcard data.
	MaxChunkSize = 0x6800
	// Expected format of the timestamp sent to the client.
	TimeFormat = "2006:01:02: 15:05:05"
	// Id sent in the menu selection packet to tell the client
	// that the selection was made on the ship menu.
	ShipSelectionMenuId uint16 = 0x13
)

var (
	// Connected ships. Each Ship's id corresponds to its position in the array - 1.
	shipList []Ship = make([]Ship, 1)

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
)

// Entry in the available ships lis on the ship selection menu.
type ShipMenuEntry struct {
	MenuId  uint16
	ShipId  uint32
	Padding uint16

	Shipname [23]byte
}

type CharacterServer struct {
	// Cached parameter data to avoid computing it every time.
	paramHeaderData []byte
	paramChunkData  map[int][]byte

	// Starting stats for any new character. The CharClass constants can be used
	// to index into this array to obtain the base stats for each class.
	BaseStats [12]CharacterStats
}

func (server CharacterServer) Name() string { return "CHARACTER" }

func (server CharacterServer) Port() string { return config.CharacterPort }

func (server *CharacterServer) Init() error {
	if err := server.loadParameterFiles(); err != nil {
		return err
	}

	// Load the base stats for creating new characters. Newserv, Sylverant, and Tethealla
	// all seem to rely on this file, so we'll do the same.
	paramDir := config.ParametersDir
	statsFile, _ := os.Open(paramDir + "/PlyLevelTbl.prs")
	compressed, err := ioutil.ReadAll(statsFile)
	if err != nil {
		return errors.New("Error reading stats file: " + err.Error())
	}

	decompressedSize := prs.DecompressSize(compressed)
	decompressed := make([]byte, decompressedSize)
	prs.Decompress(compressed, decompressed)

	for i := 0; i < 12; i++ {
		util.StructFromBytes(decompressed[i*14:], &server.BaseStats[i])
	}

	fmt.Println()
	return nil
}

// Load the PSOBB parameter files, build the parameter header,
// and init/cache the param file chunks for the EB packets.
func (server *CharacterServer) loadParameterFiles() error {
	offset := 0
	var tmpChunkData []byte

	paramDir := config.ParametersDir
	fmt.Printf("Loading parameters from %s...\n", paramDir)
	for _, paramFile := range paramFiles {
		data, err := ioutil.ReadFile(paramDir + "/" + paramFile)
		if err != nil {
			return errors.New("Error reading parameter file: " + err.Error())
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
		server.paramHeaderData = append(server.paramHeaderData, bytes...)

		tmpChunkData = append(tmpChunkData, data...)
		fmt.Printf("%s (%v bytes, checksum: %v)\n", paramFile, fileSize, entry.Checksum)
	}

	// Offset should at this point be the total size of the files
	// to send - break it all up into indexable chunks.
	server.paramChunkData = make(map[int][]byte)
	chunks := offset / MaxChunkSize
	for i := 0; i < chunks; i++ {
		dataOff := i * MaxChunkSize
		server.paramChunkData[i] = tmpChunkData[dataOff : dataOff+MaxChunkSize]
		offset -= MaxChunkSize
	}
	// Add any remaining data
	if offset > 0 {
		server.paramChunkData[chunks] = tmpChunkData[chunks*MaxChunkSize:]
	}
	return nil
}

func (server *CharacterServer) NewClient(conn *net.TCPConn) (*Client, error) {
	return NewLoginClient(conn)
}

func (server *CharacterServer) Handle(c *Client) error {
	var hdr BBHeader
	util.StructFromBytes(c.Data()[:BBHeaderSize], &hdr)

	var err error
	switch hdr.Type {
	case LoginType:
		err = server.HandleCharLogin(c)
	case LoginOptionsRequestType:
		err = server.HandleOptionsRequest(c)
	case LoginCharPreviewReqType:
		err = server.HandleCharacterSelect(c)
	case LoginChecksumType:
		// Everybody else seems to ignore this, so...
		err = server.sendChecksumAck(c)
	case LoginGuildcardReqType:
		err = server.HandleGuildcardDataStart(c)
	case LoginGuildcardChunkReqType:
		server.HandleGuildcardChunk(c)
	case LoginParameterHeaderReqType:
		err = server.sendParameterHeader(c, uint32(len(paramFiles)), server.paramHeaderData)
	case LoginParameterChunkReqType:
		var pkt BBHeader
		util.StructFromBytes(c.Data(), &pkt)
		err = server.sendParameterChunk(c, server.paramChunkData[int(pkt.Flags)], pkt.Flags)
	case LoginSetFlagType:
		var pkt SetFlagPacket
		util.StructFromBytes(c.Data(), &pkt)
		c.flag = pkt.Flag
	case LoginCharPreviewType:
		err = server.HandleCharacterUpdate(c)
	case MenuSelectType:
		err = server.HandleShipSelection(c)
	case DisconnectType:
		// Just wait until we recv 0 from the client to d/c.
		break
	default:
		log.Infof("Received unknown packet %x from %s", hdr.Type, c.IPAddr())
	}
	return err
}

func (server *CharacterServer) HandleCharLogin(client *Client) error {
	var err error
	if pkt, err := VerifyAccount(client); err == nil {
		err = server.sendSecurity(client, BBLoginErrorNone, client.guildcard, client.teamId)
		if err != nil {
			return err
		}
		// At this point, if we've chosen (or created) a character then the
		// client will send us the slot number and the corresponding phase.
		if pkt.SlotNum >= 0 && pkt.Phase == 4 {
			if err = server.sendTimestamp(client); err != nil {
				return err
			}
			if err = server.sendShipList(client, shipList); err != nil {
				return err
			}
			if err = server.sendScrollMessage(client); err != nil {
				return err
			}
		}
	}
	return err
}

// Send the security initialization packet with information about the user's
// authentication status.
func (server *CharacterServer) sendSecurity(client *Client, errorCode BBLoginError,
	guildcard uint32, teamId uint32) error {

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

// Send a timestamp packet in order to indicate the server's current time.
func (server *CharacterServer) sendTimestamp(client *Client) error {
	pkt := new(TimestampPacket)
	pkt.Header.Type = LoginTimestampType

	var tv syscall.Timeval
	syscall.Gettimeofday(&tv)
	t := time.Now().Format(TimeFormat)
	stamp := fmt.Sprintf("%s.%03d", t, uint64(tv.Usec/1000))
	copy(pkt.Timestamp[:], stamp)

	DebugLog("Sending Timestamp Packet")
	return EncryptAndSend(client, pkt)
}

// Send the menu items for the ship select screen.
func (server *CharacterServer) sendShipList(client *Client, ships []Ship) error {
	pkt := &ShipListPacket{
		Header:      BBHeader{Type: LoginShipListType, Flags: 0x01},
		Unknown:     0x02,
		Unknown2:    0xFFFFFFF4,
		Unknown3:    0x04,
		ShipEntries: make([]ShipMenuEntry, len(ships)),
	}
	copy(pkt.ServerName[:], "Archon")

	// TODO: Will eventually need a mutex for read.
	for i, ship := range ships {
		item := &pkt.ShipEntries[i]
		item.MenuId = ShipSelectionMenuId
		item.ShipId = ship.id
		copy(item.Shipname[:], util.ConvertToUtf16(string(ship.name[:])))
	}

	DebugLog("Sending Ship List Packet")
	return EncryptAndSend(client, pkt)
}

// Send whatever scrolling message was read out of the config file for the login screen.
func (server *CharacterServer) sendScrollMessage(client *Client) error {
	pkt := &ScrollMessagePacket{
		Header:  BBHeader{Type: LoginScrollMessageType},
		Message: config.ScrollMessageBytes(),
	}

	data, size := util.BytesFromStruct(pkt)
	// The end of the message appears to be garbled unless
	// there is a block of extra bytes on the end; add an extra
	// and let fixLength add the rest.
	data = append(data, 0x00)
	DebugLog("Sending Scroll Message Packet")
	return client.SendEncrypted(data, size+1)
}

// Load key config and other option data from the database or provide defaults for new accounts.
func (server *CharacterServer) HandleOptionsRequest(client *Client) error {
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
	return server.sendOptions(client, optionData)
}

// Send the client's configuration options. keyConfig should be 420 bytes long and either
// point to the default keys array or loaded from the database.
func (server *CharacterServer) sendOptions(client *Client, keyConfig []byte) error {
	if len(keyConfig) != 420 {
		panic("Received keyConfig of length " + string(len(keyConfig)) + "; should be 420")
	}
	pkt := new(OptionsPacket)
	pkt.Header.Type = LoginOptionsType

	pkt.PlayerKeyConfig.Guildcard = client.guildcard
	copy(pkt.PlayerKeyConfig.KeyConfig[:], keyConfig[:0x16C])
	copy(pkt.PlayerKeyConfig.JoystickConfig[:], keyConfig[0x16C:])

	// Sylverant sets these to enable all team rewards? Not sure what this means yet.
	pkt.PlayerKeyConfig.TeamRewards[0] = 0xFFFFFFFF
	pkt.PlayerKeyConfig.TeamRewards[1] = 0xFFFFFFFF

	DebugLog("Sending Key Config Packet")
	return EncryptAndSend(client, pkt)
}

// Handle the character select/preview request. Will either return information
// about a character given a particular slot in via 0xE5 response or ack the
// selection with an 0xE4 (also used for an empty slot).
func (server *CharacterServer) HandleCharacterSelect(client *Client) error {
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
		return server.sendCharacterAck(client, pkt.Slot, 2)
	} else if err != nil {
		log.Error(err.Error())
		return err
	}

	if pkt.Selecting == 0x01 {
		// They've selected a character from the menu.
		client.config.SlotNum = uint8(pkt.Slot)
		server.sendSecurity(client, BBLoginErrorNone, client.guildcard, client.teamId)
		return server.sendCharacterAck(client, pkt.Slot, 1)
	} else {
		// They have a character in that slot; send the character preview.
		copy(prev.GuildcardStr[:], gc[:])
		copy(prev.Name[:], name[:])
		return server.sendCharacterPreview(client, prev)
	}
}

// Send the character acknowledgement packet. 0 indicates a creation ack, 1 is
// ack'ing a selected character, and 2 indicates that a character doesn't exist
// in the slot requested via preview request.
func (server *CharacterServer) sendCharacterAck(client *Client, slotNum uint32, flag uint32) error {
	pkt := &CharAckPacket{
		Header: BBHeader{Type: LoginCharAckType},
		Slot:   slotNum,
		Flag:   flag,
	}
	DebugLog("Sending Character Ack Packet")
	return EncryptAndSend(client, pkt)
}

// Send the preview packet containing basic details about a character in the selected slot.
func (server *CharacterServer) sendCharacterPreview(client *Client, charPreview *CharacterPreview) error {
	pkt := &CharPreviewPacket{
		Header:    BBHeader{Type: LoginCharPreviewType},
		Slot:      0,
		Character: charPreview,
	}
	DebugLog("Sending Character Preview Packet")
	return EncryptAndSend(client, pkt)
}

// Acknowledge the checksum the client sent us. We don't actually do
// anything with it but the client won't proceed otherwise.
func (server *CharacterServer) sendChecksumAck(client *Client) error {
	pkt := new(ChecksumAckPacket)
	pkt.Header.Type = LoginChecksumAckType
	pkt.Ack = uint32(1)

	DebugLog("Sending Checksum Ack Packet")
	return EncryptAndSend(client, pkt)
}

// Load the player's saved guildcards, build the chunk data, and send the chunk header.
func (server *CharacterServer) HandleGuildcardDataStart(client *Client) error {
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
		// figuring out how the other servers use it.
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

	return server.sendGuildcardHeader(client, checksum, client.gcDataSize)
}

// Send the header containing metadata about the guildcard chunk.
func (server *CharacterServer) sendGuildcardHeader(client *Client, checksum uint32, dataLen uint16) error {
	pkt := &GuildcardHeaderPacket{
		Header:   BBHeader{Type: LoginGuildcardHeaderType},
		Unknown:  0x00000001,
		Length:   dataLen,
		Checksum: checksum,
	}
	DebugLog("Sending Guildcard Header Packet")
	return EncryptAndSend(client, pkt)
}

// Send another chunk of the client's guildcard data.
func (server *CharacterServer) HandleGuildcardChunk(client *Client) {
	var chunkReq GuildcardChunkReqPacket
	util.StructFromBytes(client.Data(), &chunkReq)
	if chunkReq.Continue == 0x01 {
		server.sendGuildcardChunk(client, chunkReq.ChunkRequested)
	}
	// Anything else is a request to cancel sending guildcard chunks.
}

// Send the specified chunk of guildcard data.
func (server *CharacterServer) sendGuildcardChunk(client *Client, chunkNum uint32) error {
	pkt := new(GuildcardChunkPacket)
	pkt.Header.Type = LoginGuildcardChunkType
	pkt.Chunk = chunkNum

	// The client will only accept 0x6800 bytes of a chunk per packet.
	offset := uint16(chunkNum) * MaxChunkSize
	remaining := client.gcDataSize - offset
	if remaining > MaxChunkSize {
		pkt.Data = client.gcData[offset : offset+MaxChunkSize]
	} else {
		pkt.Data = client.gcData[offset:]
	}

	DebugLog("Sending Guildcard Chunk Packet")
	return EncryptAndSend(client, pkt)
}

// Send the header for the parameter files we're about to start sending.
func (server *CharacterServer) sendParameterHeader(client *Client, numEntries uint32, entries []byte) error {
	pkt := &ParameterHeaderPacket{
		Header:  BBHeader{Type: LoginParameterHeaderType, Flags: numEntries},
		Entries: entries,
	}
	DebugLog("Sending Parameter Header Packet")
	return EncryptAndSend(client, pkt)
}

// Index into chunkData and send the specified chunk of parameter data.
func (server *CharacterServer) sendParameterChunk(client *Client, chunkData []byte, chunk uint32) error {
	pkt := &ParameterChunkPacket{
		Header: BBHeader{Type: LoginParameterChunkType},
		Chunk:  chunk,
		Data:   chunkData,
	}
	DebugLog("Sending Parameter Chunk Packet")
	return EncryptAndSend(client, pkt)
}

// Create or update a character in a slot.
func (server *CharacterServer) HandleCharacterUpdate(client *Client) error {
	var charPkt CharPreviewPacket
	charPkt.Character = new(CharacterPreview)
	util.StructFromBytes(client.Data(), &charPkt)
	p := charPkt.Character

	archonDB := config.DB()
	if client.flag == 0x02 {
		// Player is using the dressing room; update the character. Messy
		// query, but unavoidable if we don't want to be stuck with blobs.
		_, err := archonDB.Exec("UPDATE characters SET name_color=?, model=?, "+
			"name_color_chksm=?, section_id=?, char_class=?, costume=?, skin=?, "+
			"head=?, hair_red=?, hair_green=?, hair_blue,=? proportion_x=?, "+
			"proportion_y=?, name=? WHERE guildcard = ? AND slot_num = ?",
			p.NameColor, p.Model, p.NameColorChksm, p.SectionId,
			p.Class, p.Costume, p.Skin, p.Head, p.HairRed,
			p.HairGreen, p.HairBlue, p.Name[:], p.PropX, p.PropY,
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
		stats := server.BaseStats[p.Class]

		// TODO: Set up the default inventory and techniques.
		meseta := 300

		/* TODO: Add the rest of these.
		--unsigned char keyConfig[232]; // 0x3E8 - 0x4CF;
		--techniques blob,
		--options blob,
		*/

		// Create the new character.
		_, err = archonDB.Exec("INSERT INTO characters (guildcard, slot_num,"+
			"experience, level, guildcard_str, name_color, model, name_color_chksm,"+
			"section_id, char_class, v2_flags, version, v1_flags, costume,"+
			"skin, face, head, hair, hair_red, hair_green, hair_blue,"+
			"proportion_x, proportion_y, name, playtime, atp, mst, evp, "+
			"hp, dfp, ata, lck, meseta, bank_use, bank_meseta) "+
			"VALUES (?, ?, 0, 0, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, "+
			"?, ?, ?, ?, ?, ?, 0, ?, ?, ?, ?, ?, ?, ?, ?, 0, 0)",
			client.guildcard, charPkt.Slot, p.GuildcardStr[:], p.NameColor,
			p.Model, p.NameColorChksm, p.SectionId, p.Class, p.V2flags,
			p.Version, p.V1Flags, p.Costume, p.Skin, p.Face, p.Head,
			p.Hair, p.HairRed, p.HairGreen, p.HairBlue, p.PropX, p.PropY,
			p.Name[:], stats.ATP, stats.MST, stats.EVP, stats.HP, stats.DFP, stats.ATA,
			stats.LCK, meseta)
		if err != nil {
			log.Error(err.Error())
			return err
		}
	}

	// Send the security packet with the updated state and slot number so that
	// we know a character has been selected.
	client.config.SlotNum = uint8(charPkt.Slot)
	return server.sendCharacterAck(client, charPkt.Slot, 0)
}

// Player selected one of the items on the ship select screen.
func (server *CharacterServer) HandleShipSelection(client *Client) error {
	var pkt MenuSelectionPacket
	util.StructFromBytes(client.Data(), &pkt)
	selectedShip := pkt.ItemId - 1
	if selectedShip < 0 || selectedShip >= uint32(len(shipList)) {
		return errors.New("Invalid ship selection: " + string(selectedShip))
	}
	s := &shipList[selectedShip]
	return SendRedirect(client, s.ipAddr[:], s.port)
}
