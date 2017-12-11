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
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"net"
	"os"
	"syscall"
	"time"

	"github.com/dcrodman/archon/prs"
	"github.com/dcrodman/archon/util"
)

const (
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

// Per-character stats as stored in config files.
type CharacterStats struct {
	ATP uint16
	MST uint16
	EVP uint16
	HP  uint16
	DFP uint16
	ATA uint16
	LCK uint16
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
	playerOptions, err := database.FindPlayerOptions(client.guildcard)
	if playerOptions == nil {
		// We don't have any saved key config - give them the defaults.
		playerOptions = &PlayerOptions{
			Guildcard: client.guildcard,
			KeyConfig: make([]byte, 420),
		}
		copy(playerOptions.KeyConfig, baseKeyConfig[:])
		database.UpdatePlayerOptions(playerOptions)
	} else if err != nil {
		log.Error(err.Error())
		return err
	}
	return server.sendOptions(client, playerOptions.KeyConfig)
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

	character, err := database.FindCharacter(client.guildcard, pkt.Slot)
	if character == nil {
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
	}
	// They have a character in that slot; send the character preview.
	return server.sendCharacterPreview(client, character)
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
func (server *CharacterServer) sendCharacterPreview(client *Client, character *Character) error {
	charPreview := &CharacterPreview{
		Experience:     character.Experience,
		Level:          character.Level,
		NameColor:      character.NameColor,
		Model:          character.Model,
		NameColorChksm: character.NameColorChecksum,
		SectionID:      character.SectionID,
		Class:          character.Class,
		V2Flags:        character.V2Flags,
		Version:        character.Version,
		V1Flags:        character.V1Flags,
		Costume:        character.Costume,
		Skin:           character.Skin,
		Face:           character.Face,
		Head:           character.Head,
		Hair:           character.Hair,
		HairRed:        character.HairRed,
		HairGreen:      character.HairGreen,
		HairBlue:       character.HairBlue,
		PropX:          character.ProportionX,
		PropY:          character.ProportionY,
		Playtime:       character.Playtime,
	}
	copy(charPreview.GuildcardStr[:], character.GuildcardStr[:])
	copy(charPreview.Name[:], character.Name[:])

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
	guildcards, err := database.FindGuildcardData(client.guildcard)
	if err != nil {
		return err
	}

	gcData := new(GuildcardData)
	// Maximum of 140 entries can be sent.
	for i, entry := range guildcards {
		// TODO: This may not actually work yet, but I haven't gotten to
		// figuring out how the other servers use it.
		pktEntry := gcData.Entries[i]
		pktEntry.Guildcard = uint32(entry.Guildcard)
		copy(pktEntry.Name[:], entry.Name)
		copy(pktEntry.TeamName[:], entry.TeamName)
		copy(pktEntry.Description[:], entry.Description)
		pktEntry.Language = entry.Language
		pktEntry.SectionID = entry.SectionID
		pktEntry.CharClass = entry.Class
		copy(pktEntry.Comment[:], entry.Comment)
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

// Player has modified a character via the dressing room or selected the recreate option.
// Recreate or update a character in a slot depending on which it was.
func (server *CharacterServer) HandleCharacterUpdate(client *Client) error {
	var charPkt CharPreviewPacket
	charPkt.Character = new(CharacterPreview)
	util.StructFromBytes(client.Data(), &charPkt)

	if client.flag == 0x02 {
		if err := server.updateCharacter(client.guildcard, &charPkt); err != nil {
			log.Error(err.Error())
			return err
		}
	} else {
		// Recreating; delete the existing character and start from scratch.
		if err := database.DeleteCharacter(client.guildcard, charPkt.Slot); err != nil {
			log.Error(err.Error())
			return err
		}

		p := charPkt.Character
		// Grab our base stats for this character class.
		stats := server.BaseStats[p.Class]

		character := &Character{
			Experience:        0,
			Level:             0,
			GuildcardStr:      p.GuildcardStr[:],
			NameColor:         p.NameColor,
			Model:             p.Model,
			NameColorChecksum: p.NameColorChksm,
			SectionID:         p.SectionID,
			Class:             p.Class,
			V2Flags:           p.V2Flags,
			Version:           p.Version,
			V1Flags:           p.V1Flags,
			Costume:           p.Costume,
			Skin:              p.Skin,
			Face:              p.Face,
			Head:              p.Head,
			Hair:              p.Hair,
			HairRed:           p.HairRed,
			HairGreen:         p.HairGreen,
			HairBlue:          p.HairBlue,
			ProportionX:       p.PropX,
			ProportionY:       p.PropY,
			Name:              p.Name[:],
			ATP:               stats.ATP,
			MST:               stats.MST,
			EVP:               stats.EVP,
			HP:                stats.HP,
			DFP:               stats.DFP,
			ATA:               stats.ATA,
			LCK:               stats.LCK,
			Meseta:            300,
		}
		/* TODO: Add the rest of these.
		--unsigned char keyConfig[232]; // 0x3E8 - 0x4CF;
		--techniques blob,
		--options blob,
		*/

		err := database.CreateCharacter(client.guildcard, charPkt.Slot, character)
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

func (server *CharacterServer) updateCharacter(guildcard uint32, pkt *CharPreviewPacket) error {
	// Player is using the dressing room; update the character.
	character, err := database.FindCharacter(guildcard, pkt.Slot)
	if character == nil {
		err = fmt.Errorf("Character does not exist in slot %d for guildcard %d",
			pkt.Slot, guildcard)
	} else if err == nil {
		p := pkt.Character
		character.NameColor = p.NameColor
		character.Model = p.Model
		character.NameColorChecksum = p.NameColorChksm
		character.SectionID = p.SectionID
		character.Class = p.Class
		character.Costume = p.Costume
		character.Skin = p.Skin
		character.Head = p.Head
		character.HairRed = p.HairRed
		character.HairGreen = p.HairGreen
		character.HairBlue = p.HairBlue
		character.ProportionX = p.PropX
		character.ProportionY = p.PropY
		copy(character.Name, p.Name[:])

		err = database.UpdateCharacter(guildcard, pkt.Slot, character)
	}
	return err
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
