package character

import (
	"context"
	"fmt"
	"hash/crc32"
	"net"
	"strconv"
	"sync"
	"syscall"
	"time"
	"unicode/utf16"

	"github.com/sirupsen/logrus"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/dcrodman/archon/internal/core"
	"github.com/dcrodman/archon/internal/core/bytes"
	"github.com/dcrodman/archon/internal/core/client"
	"github.com/dcrodman/archon/internal/core/proto"
	"github.com/dcrodman/archon/internal/packets"
	"github.com/dcrodman/archon/internal/shipgate"
)

const (
	// Maximum size of a block of parameter or guildcard data.
	maxDataChunkSize = 0x6800
	// Expected format of the timestamp sent to the client.
	timeFormat = "2006:01:02: 15:05:05"
	// Id sent in the menu selection packet to tell the client
	// that the selection was made on the ship menu.
	ShipSelectionMenuId uint16 = 0x12
)

var (
	// Copyright in the welcome packet. The client expects exactly this string and will
	// crash if it does not exactly match.
	loginCopyright = []byte("Phantasy Star Online Blue Burst Game Server. Copyright 1999-2004 SONICTEAM.")

	// Scrolling message that appears across the top of the ship selection screen.
	shipSelectionScrollMessage     []byte
	shipSelectionScrollMessageInit sync.Once
)

func clientFlagCacheKey(c *client.Client) string {
	return fmt.Sprintf("client-flags-%d", c.Account.Id)
}

// Server is the CHARACTER server implementation. Clients are sent to this server
//
//	after authenticating with LOGIN. Each client connects to the server in four
//
// different phases (each one is a new connection):
//  1. Data download (login options, guildcard, and character previews).
//  2. Character selection
//  3. (Optional) Character creation/modification (recreate and dressing room)
//  4. Confirmation and ship selection
//
// The ship list is obtained by communicating with the shipgate server since ships
// do not directly connect to this server.
type Server struct {
	Name   string
	Config *core.Config
	Logger *logrus.Logger

	kvCache        *Cache
	shipgateClient shipgate.Shipgate
}

func (s *Server) Identifier() string {
	return s.Name
}

func (s *Server) Init(ctx context.Context) error {
	s.kvCache = NewCache()
	s.shipgateClient = shipgate.NewRPCClient(s.Config)

	if err := initParameterData(s.Logger, s.Config.CharacterServer.ParametersDir); err != nil {
		return err
	}
	return nil
}

func (s *Server) SetUpClient(c *client.Client) {
	c.CryptoSession = client.NewBlueBurstCryptoSession()
	c.DebugTags["server_type"] = "character"
}

func (s *Server) Handshake(c *client.Client) error {
	pkt := &packets.Welcome{
		Header:       packets.BBHeader{Type: packets.LoginWelcomeType, Size: 0xC8},
		Copyright:    [96]byte{},
		ServerVector: [48]byte{},
		ClientVector: [48]byte{},
	}
	copy(pkt.Copyright[:], loginCopyright)
	copy(pkt.ServerVector[:], c.CryptoSession.ServerVector())
	copy(pkt.ClientVector[:], c.CryptoSession.ClientVector())

	return c.SendRaw(pkt)
}

func (s *Server) Handle(ctx context.Context, c *client.Client, data []byte) error {
	var packetHeader packets.BBHeader
	bytes.StructFromBytes(data[:packets.BBHeaderSize], &packetHeader)

	var err error
	switch packetHeader.Type {
	case packets.LoginType:
		var loginPkt packets.Login
		bytes.StructFromBytes(data, &loginPkt)
		err = s.handleLogin(ctx, c, &loginPkt)
	case packets.LoginOptionsRequestType:
		err = s.handleOptionsRequest(ctx, c)
	case packets.LoginCharSelectType:
		var pkt packets.CharacterSelection
		bytes.StructFromBytes(data, &pkt)
		err = s.handleCharacterSelect(ctx, c, &pkt)
	case packets.LoginChecksumType:
		// Everybody else seems to ignore this, so...
		err = s.sendChecksumAck(c)
	case packets.LoginGuildcardReqType:
		err = s.handleGuildcardDataStart(ctx, c)
	case packets.LoginGuildcardChunkReqType:
		var chunkReq packets.GuildcardChunkRequest
		bytes.StructFromBytes(data, &chunkReq)
		err = s.handleGuildcardChunk(c, &chunkReq)
	case packets.LoginParameterHeaderReqType:
		err = s.sendParameterHeader(c, uint32(len(paramFiles)), paramHeaderData)
	case packets.LoginParameterChunkReqType:
		var pkt packets.BBHeader
		bytes.StructFromBytes(data, &pkt)
		err = s.sendParameterChunk(c, paramChunkData[int(pkt.Flags)], pkt.Flags)
	case packets.LoginSetFlagType:
		var pkt packets.SetFlag
		bytes.StructFromBytes(data, &pkt)
		s.setClientFlag(c, &pkt)
	case packets.LoginCharPreviewType:
		var charPkt packets.CharacterSummary
		bytes.StructFromBytes(data, &charPkt)
		err = s.handleCharacterUpdate(ctx, c, &charPkt)
	case packets.MenuSelectType:
		var menuSelectionPkt packets.MenuSelection
		bytes.StructFromBytes(data, &menuSelectionPkt)
		err = s.handleShipSelection(ctx, c, &menuSelectionPkt)
	case packets.DisconnectType:
		// Just wait for the client to disconnect.
		break
	default:
		s.Logger.Infof("received unknown packet %x from %s", packetHeader.Type, c.IPAddr())
	}
	return err
}

func (s *Server) handleLogin(ctx context.Context, c *client.Client, loginPkt *packets.Login) error {
	username := string(bytes.StripPadding(loginPkt.Username[:]))
	password := string(bytes.StripPadding(loginPkt.Password[:]))

	account, err := s.shipgateClient.AuthenticateAccount(ctx, &shipgate.AuthenticateAccountRequest{
		Username: username,
		Password: password,
	})
	if err != nil {
		switch err {
		case shipgate.ErrInvalidCredentials:
			return s.sendSecurity(c, packets.BBLoginErrorPassword)
		case shipgate.ErrAccountBanned:
			return s.sendSecurity(c, packets.BBLoginErrorBanned)
		default:
			sendErr := s.sendMessage(c, cases.Title(language.English).String(err.Error()))
			if sendErr == nil {
				return sendErr
			}
			return err
		}
	}

	if err := s.sendSecurity(c, packets.BBLoginErrorNone); err != nil {
		return err
	}

	c.Account = account
	c.TeamID = uint32(account.TeamId)
	c.Guildcard = uint32(account.Guildcard)

	// At this point, the user has chosen (or created) a character and the
	// client needs the ship list.
	if loginPkt.Phase == packets.ShipSelection {
		if err = s.sendTimestamp(c); err != nil {
			return err
		}
		if err = s.sendShipList(ctx, c); err != nil {
			return err
		}
		if err = s.sendScrollMessage(c); err != nil {
			return err
		}
	}

	return nil
}

// send the security initialization packet with information about the user's
// authentication status.
func (s *Server) sendSecurity(c *client.Client, errorCode uint32) error {
	cfg := packets.ClientConfig{
		Magic:        c.Config.Magic,
		CharSelected: c.Config.CharSelected,
		SlotNum:      c.Config.SlotNum,
		Flags:        c.Config.Flags,
	}
	copy(cfg.Ports[:], c.Config.Ports[:])
	copy(cfg.Unused[:], c.Config.Unused[:])
	copy(cfg.Unused2[:], c.Config.Unused2[:])

	// Constants set according to how Newserv does it.
	return c.Send(&packets.Security{
		Header:       packets.BBHeader{Type: packets.LoginSecurityType},
		ErrorCode:    errorCode,
		PlayerTag:    0x00010000,
		Guildcard:    c.Guildcard,
		TeamID:       c.TeamID,
		Config:       cfg,
		Capabilities: 0x00000102,
	})
}

// Sends a message to the client. In this case whatever message is sent
// here will be displayed in a dialog box after the patch screen.
func (s *Server) sendMessage(c *client.Client, message string) error {
	return c.Send(&packets.LoginClientMessage{
		Header:   packets.BBHeader{Type: packets.LoginClientMessageType},
		Language: 0x00450009,
		Message:  bytes.ConvertToUtf16(message),
	})
}

// Send a timestamp packet in order to indicate the server's current time.
func (s *Server) sendTimestamp(c *client.Client) error {
	pkt := &packets.Timestamp{
		Header:    packets.BBHeader{Type: packets.LoginTimestampType},
		Timestamp: [28]byte{},
	}

	var tv syscall.Timeval
	_ = syscall.Gettimeofday(&tv)
	stamp := fmt.Sprintf("%s.%03d", time.Now().Format(timeFormat), uint64(tv.Usec/1000))
	copy(pkt.Timestamp[:], stamp)

	return c.Send(pkt)
}

// Send the menu items for the ship select screen.
func (s *Server) sendShipList(ctx context.Context, c *client.Client) error {
	shipList, err := s.shipgateClient.GetActiveShips(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("error retrieving ship list: %w", err)
	}

	pkt := &packets.ShipList{
		Header: packets.BBHeader{
			Type:  packets.LoginShipListType,
			Flags: uint32(len(shipList.Ships)),
		},
		Unknown:  0x20,
		Unknown2: 0xFFFFFFF4,
		Unknown3: 0x04,
	}
	copy(pkt.ServerName[:], bytes.ConvertToUtf16("Archon"))

	if len(shipList.Ships) == 0 {
		pkt.ShipEntries = append(pkt.ShipEntries, packets.ShipListEntry{
			MenuID: 0xFF,
			ShipID: 0xFF,
		})
		// pkt.Header.Flags = 1
		copy(pkt.ShipEntries[0].ShipName[:], ("No Ships!")[:])
	} else {

		for i, ship := range shipList.Ships {
			entry := packets.ShipListEntry{
				MenuID: uint16(i + 1),
				ShipID: uint32(ship.Id),
			}
			copy(entry.ShipName[:], bytes.ConvertToUtf16(ship.Name[:]))
			pkt.ShipEntries = append(pkt.ShipEntries, entry)
		}
	}

	return c.Send(pkt)
}

// send whatever scrolling message was read out of the config file for the login screen.
func (s *Server) sendScrollMessage(c *client.Client) error {
	// Returns the scroll message displayed along the top of the ship selection screen,
	// lazily computing it from the config file and storing it in a package var.
	shipSelectionScrollMessageInit.Do(func() {
		shipSelectionScrollMessage = bytes.ConvertToUtf16(
			s.Config.CharacterServer.ScrollMessage,
		)
		// The end of the message appears to be garbled unless there is an extra byte...?
		shipSelectionScrollMessage = append(shipSelectionScrollMessage, 0x00)
	})

	return c.Send(&packets.ScrollMessagePacket{
		Header:  packets.BBHeader{Type: packets.LoginScrollMessageType},
		Message: shipSelectionScrollMessage,
	})
}

// LoadConfig key config and other option data from the database or provide defaults for new accounts.
func (s *Server) handleOptionsRequest(ctx context.Context, c *client.Client) error {
	var (
		err           error
		resp          *shipgate.GetPlayerOptionsResponse
		playerOptions *proto.PlayerOptions
	)
	if resp, err = s.shipgateClient.GetPlayerOptions(ctx, &shipgate.GetPlayerOptionsRequest{
		AccountId: c.Account.Id,
	}); err != nil {
		return fmt.Errorf("error handling options request: %w", err)
	}

	if resp.Exists {
		playerOptions = resp.PlayerOptions
	}
	if playerOptions == nil {
		// We don't have any saved key config - give them the defaults.
		playerOptions = &proto.PlayerOptions{
			KeyConfig: make([]byte, 420),
		}
		copy(playerOptions.KeyConfig, BaseKeyConfig[:])

		if _, err = s.shipgateClient.UpsertPlayerOptions(ctx, &shipgate.UpsertPlayerOptionsRequest{
			AccountId: c.Account.Id,

			PlayerOptions: playerOptions,
		}); err != nil {
			return fmt.Errorf("error creating player options: %w", err)
		}
	}

	return s.sendOptions(c, playerOptions.KeyConfig)
}

// send the client's configuration options. keyConfig should be 420 bytes long and either
// point to the default keys array or loaded from the database.
func (s *Server) sendOptions(c *client.Client, keyConfig []byte) error {
	if len(keyConfig) != 420 {
		return fmt.Errorf("received keyConfig of length %d; should be 420", len(keyConfig))
	}

	pkt := &packets.Options{
		Header: packets.BBHeader{Type: packets.LoginOptionsType},
	}
	pkt.PlayerKeyConfig.Guildcard = c.Guildcard
	copy(pkt.PlayerKeyConfig.KeyConfig[:], keyConfig[:0x16C])
	copy(pkt.PlayerKeyConfig.JoystickConfig[:], keyConfig[0x16C:])

	// Sylverant sets these to enable all team rewards? Not sure what this means yet.
	pkt.PlayerKeyConfig.TeamRewards[0] = 0xFFFFFFFF
	pkt.PlayerKeyConfig.TeamRewards[1] = 0xFFFFFFFF

	return c.Send(pkt)
}

// Handle the character select/preview request.
//
// For the preview request, this method will either send info about a character given
// a particular slot in an 0xE5 response or ack the selection with an 0xE4 (also used
// for an empty slot). The client will send one of these preview request packets for
// each of the character slots (i.e. 4 times).
//
// The client also sends this packet when  a character has been selected from the menu
// (or after the dressing room or recreate), as indicated by the Selecting flag.
func (s *Server) handleCharacterSelect(ctx context.Context, c *client.Client, pkt *packets.CharacterSelection) error {
	resp, err := s.shipgateClient.FindCharacter(ctx, &shipgate.CharacterRequest{
		AccountId: c.Account.Id,
		Slot:      pkt.Slot,
	})
	if err != nil {
		return fmt.Errorf("error selecting character: %w", err)
	}

	if pkt.Selecting == 0x01 {
		if !resp.Exists {
			return fmt.Errorf("attempted to select nonexistent character in slot: %d", pkt.Slot)
		}
		// They've selected a character from the menu.
		c.Config.SlotNum = uint8(pkt.Slot)
		return s.sendCharacterAck(c, pkt.Slot, 1)
	}

	if resp.Exists {
		// They have a character in that slot; send the character preview.
		return s.sendCharacterPreview(c, resp.Character)
	}
	// We don't have a character for this slot.
	return s.sendCharacterAck(c, pkt.Slot, 2)
}

// Send the character acknowledgement packet in response to the action taken. Setting flag
// to 0 indicates a creation ack, 1 acks a selected character, and 2 indicates that a character
// doesn't exist in the slot requested via preview request.
func (s *Server) sendCharacterAck(c *client.Client, slotNum uint32, flag uint32) error {
	return c.Send(&packets.CharacterAck{
		Header: packets.BBHeader{Type: packets.LoginCharAckType},
		Slot:   slotNum,
		Flag:   flag,
	})
}

// send the preview packet containing basic details about a character in the selected slot.
func (s *Server) sendCharacterPreview(c *client.Client, char *proto.Character) error {
	previewPacket := &packets.CharacterSummary{
		Header: packets.BBHeader{Type: packets.LoginCharPreviewType},
		Slot:   0,
		Character: packets.CharacterPreview{
			Experience:     char.Experience,
			Level:          char.Level,
			NameColor:      char.NameColor,
			Model:          byte(char.ModelType),
			NameColorChksm: char.NameColorChecksum,
			SectionID:      byte(char.SectionId),
			Class:          byte(char.Class),
			V2Flags:        byte(char.V2Flags),
			Version:        byte(char.Version),
			V1Flags:        char.V1Flags,
			Costume:        uint16(char.Costume),
			Skin:           uint16(char.Skin),
			Face:           uint16(char.Face),
			Head:           uint16(char.Head),
			Hair:           uint16(char.Hair),
			HairRed:        uint16(char.HairRed),
			HairGreen:      uint16(char.HairGreen),
			HairBlue:       uint16(char.HairBlue),
			PropX:          char.ProportionX,
			PropY:          char.ProportionY,
			Playtime:       char.Playtime,
		},
	}
	copy(previewPacket.Character.GuildcardStr[:], char.GuildcardStr[:])
	copy(previewPacket.Character.Name[:], char.Name[:])

	return c.Send(previewPacket)
}

// Acknowledge the checksum the client sent us. We don't actually do
// anything with it but the client won't proceed otherwise.
func (s *Server) sendChecksumAck(c *client.Client) error {
	return c.Send(&packets.ChecksumAck{
		Header: packets.BBHeader{Type: packets.LoginChecksumAckType},
		Ack:    1,
	})
}

// LoadConfig the player's saved guildcards, build the chunk data, and send the chunk header.
func (s *Server) handleGuildcardDataStart(ctx context.Context, c *client.Client) error {
	resp, err := s.shipgateClient.GetGuildcardEntries(ctx, &shipgate.GetGuildcardEntriesRequest{
		AccountId: c.Account.Id,
	})
	if err != nil {
		return fmt.Errorf("error loading guildcards: %w", err)
	}

	gcData := new(GuildcardData)
	// Maximum of 140 entries can be sent.
	for i, entry := range resp.Entries {
		// TODO: This may not actually work yet, but I haven't gotten to
		// figuring out how the other servers use it.
		pktEntry := gcData.Entries[i]
		pktEntry.Guildcard = uint32(entry.Guildcard)
		copy(pktEntry.Name[:], entry.Name)
		copy(pktEntry.TeamName[:], entry.TeamName)
		copy(pktEntry.Description[:], entry.Description)
		pktEntry.Language = uint8(entry.Language)
		pktEntry.SectionID = uint8(entry.SectionId)
		pktEntry.CharClass = uint8(entry.Class)
		copy(pktEntry.Comment[:], entry.Comment)
	}

	var size int
	c.GuildcardData, size = bytes.BytesFromStruct(gcData)
	checksum := crc32.ChecksumIEEE(c.GuildcardData)

	return s.sendGuildcardHeader(c, checksum, uint16(size))
}

// send the header containing metadata about the guildcard chunk.
func (s *Server) sendGuildcardHeader(c *client.Client, checksum uint32, dataLen uint16) error {
	return c.Send(&packets.GuildcardHeader{
		Header:   packets.BBHeader{Type: packets.LoginGuildcardHeaderType},
		Unknown:  0x00000001,
		Length:   dataLen,
		Checksum: checksum,
	})
}

// send another chunk of the client's guildcard data.
func (s *Server) handleGuildcardChunk(c *client.Client, chunkReq *packets.GuildcardChunkRequest) error {
	if chunkReq.Continue == 0x01 {
		return s.sendGuildcardChunk(c, chunkReq.ChunkRequested)
	}
	// Anything else is a request to cancel sending guildcard chunks.
	return nil
}

// send the specified chunk of guildcard data.
func (s *Server) sendGuildcardChunk(c *client.Client, chunkNum uint32) error {
	pkt := &packets.GuildcardChunk{
		Header: packets.BBHeader{Type: packets.LoginGuildcardChunkType},
		Chunk:  chunkNum,
	}

	// The client will only accept 0x6800 bytes of a chunk per packet.
	offset := uint16(chunkNum) * maxDataChunkSize
	remaining := uint16(len(c.GuildcardData)) - offset

	if remaining > maxDataChunkSize {
		pkt.Data = c.GuildcardData[offset : offset+maxDataChunkSize]
	} else {
		pkt.Data = c.GuildcardData[offset:]
	}

	return c.Send(pkt)
}

// send the header for the parameter files we're about to start sending.
func (s *Server) sendParameterHeader(c *client.Client, numEntries uint32, entries []byte) error {
	return c.Send(&packets.ParameterHeader{
		Header: packets.BBHeader{
			Type:  packets.LoginParameterHeaderType,
			Flags: numEntries,
		},
		Entries: entries,
	})
}

// Index into chunkData and send the specified chunk of parameter data.
func (s *Server) sendParameterChunk(c *client.Client, chunkData []byte, chunk uint32) error {
	return c.Send(&packets.ParameterChunk{
		Header: packets.BBHeader{Type: packets.LoginParameterChunkType},
		Chunk:  chunk,
		Data:   chunkData,
	})
}

// The client may send us flags as a result of user actions in order to indicate
// a change in state or desired behavior. For instance, setting 0x02 indicates
// that the character dressing room has been opened.
func (s *Server) setClientFlag(c *client.Client, pkt *packets.SetFlag) {
	c.Flag = c.Flag | pkt.Flag
	// Some flags are set right before the client disconnects, which means saving them
	// on the Client struct alone isn't safe since the state is lost. To fix this the
	// flags are also kept in memory to avoid bugs like accidentally recreating characters.
	s.kvCache.Put(clientFlagCacheKey(c), c.Flag, -1)
}

// Performs a create or update/delete depending on whether the user followed the
// "dressing room" or "recreate" flows (as indicated by a client flag).
func (s *Server) handleCharacterUpdate(ctx context.Context, c *client.Client, charPkt *packets.CharacterSummary) error {
	if s.hasDressingRoomFlag(c) {
		// "Dressing room"; a request to update an existing character.
		if err := s.updateCharacter(ctx, c, charPkt); err != nil {
			s.Logger.Error(err.Error())
			return err
		}
	} else {
		// The "recreate" option. This is a request to create a character in a slot and is used
		// for both creating new characters and replacing existing ones.
		if _, err := s.shipgateClient.DeleteCharacter(ctx, &shipgate.CharacterRequest{
			AccountId: c.Account.Id,
			Slot:      charPkt.Slot,
		}); err != nil {
			msg := fmt.Errorf("error deleting character for account %d in slot %d ", c.Account.Id, charPkt.Slot)
			s.Logger.Error(msg)
			return msg
		}

		p := charPkt.Character
		stats := BaseStats[p.Class]

		newCharacter := &proto.Character{
			Guildcard:         c.Account.Guildcard,
			GuildcardStr:      p.GuildcardStr[:],
			Slot:              charPkt.Slot,
			Experience:        0,
			Level:             0,
			NameColor:         p.NameColor,
			ModelType:         int32(p.Model),
			NameColorChecksum: p.NameColorChksm,
			SectionId:         int32(p.SectionID),
			Class:             int32(p.Class),
			V2Flags:           int32(p.V2Flags),
			Version:           int32(p.Version),
			V1Flags:           p.V1Flags,
			Costume:           uint32(p.Costume),
			Skin:              uint32(p.Skin),
			Face:              uint32(p.Face),
			Head:              uint32(p.Head),
			Hair:              uint32(p.Hair),
			HairRed:           uint32(p.HairRed),
			HairGreen:         uint32(p.HairGreen),
			HairBlue:          uint32(p.HairBlue),
			ProportionX:       p.PropX,
			ProportionY:       p.PropY,
			Name:              p.Name[:],
			Atp:               uint32(stats.ATP),
			Mst:               uint32(stats.MST),
			Evp:               uint32(stats.EVP),
			Hp:                uint32(stats.HP),
			Dfp:               uint32(stats.DFP),
			Ata:               uint32(stats.ATA),
			Lck:               uint32(stats.LCK),
			Meseta:            StartingMeseta,
		}
		newCharacter.ReadableName = convertReadableName(p.Name[:])

		// TODO: Add the rest of these.
		//--unsigned char keyConfig[232]; // 0x3E8 - 0x4CF;
		//--techniques blob,
		//--options blob,

		if _, err := s.shipgateClient.UpsertCharacter(ctx, &shipgate.UpsertCharacterRequest{
			AccountId: c.Account.Id,
			Character: newCharacter,
		}); err != nil {
			return err
		}
	}

	c.Config.SlotNum = uint8(charPkt.Slot)
	return s.sendCharacterAck(c, charPkt.Slot, 0)
}

func convertReadableName(name []uint8) string {
	// The string is UTF-16LE encoded; convert it from from []uint8 to a []uint16
	// slice with the bytes reversed and drops the language code prefix (0x09006900).
	cleanedName := name[4:]
	utfName := make([]uint16, 0)
	for i, j := 0, 0; i <= len(cleanedName)-2; i += 2 {
		if cleanedName[i]|cleanedName[i+1] == 0 {
			break
		}
		utfName = append(utfName, uint16(cleanedName[i])|uint16(cleanedName[i+1]<<4))
		j++
	}

	return string(utf16.Decode(utfName))
}

func (s *Server) hasDressingRoomFlag(c *client.Client) bool {
	if (c.Flag & 0x02) != 0 {
		return true
	}

	flags, found := s.kvCache.Get(clientFlagCacheKey(c))
	if found {
		return (flags.(uint32) & 0x02) != 0
	}
	return false
}

func (s *Server) updateCharacter(ctx context.Context, c *client.Client, pkt *packets.CharacterSummary) error {
	// Clear the dressing room flag so that it doesn't get stuck and cause problems.
	flags, _ := s.kvCache.Get(clientFlagCacheKey(c))
	s.kvCache.Put(clientFlagCacheKey(c), flags.(uint32)^0x02, -1)

	resp, err := s.shipgateClient.FindCharacter(ctx, &shipgate.CharacterRequest{
		AccountId: c.Account.Id,
		Slot:      pkt.Slot,
	})
	if err != nil {
		return err
	} else if !resp.Exists {
		return fmt.Errorf("character does not exist in slot %d for guildcard %d", pkt.Slot, c.Guildcard)
	}

	pc := pkt.Character
	character := resp.Character
	character.NameColor = pc.NameColor
	character.ModelType = int32(pc.Model)
	character.NameColorChecksum = pc.NameColorChksm
	character.SectionId = int32(pc.SectionID)
	character.Class = int32(pc.Class)
	character.Costume = uint32(pc.Costume)
	character.Skin = uint32(pc.Skin)
	character.Head = uint32(pc.Head)
	character.HairRed = uint32(pc.HairRed)
	character.HairGreen = uint32(pc.HairGreen)
	character.HairBlue = uint32(pc.HairBlue)
	character.ProportionX = pc.PropX
	character.ProportionY = pc.PropY
	character.Name = pc.Name[:]
	character.ReadableName = convertReadableName(pc.Name[:])

	_, err = s.shipgateClient.UpsertCharacter(ctx, &shipgate.UpsertCharacterRequest{
		AccountId: c.Account.Id,
		Character: character,
	})
	return err
}

// Player selected one of the items on the ship select screen; respond with the
// IP address and port of the ship server to  which the client will connect after
// disconnecting from this server.
func (s *Server) handleShipSelection(ctx context.Context, c *client.Client, menuSelectionPkt *packets.MenuSelection) error {
	shipList, err := s.shipgateClient.GetActiveShips(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("error retrieving ship list: %w", err)
	}

	selectedShip := menuSelectionPkt.ItemID - 1
	if selectedShip >= uint32(len(shipList.Ships)) {
		return fmt.Errorf("invalid ship selection: %d", selectedShip)
	}

	ip := net.ParseIP(shipList.Ships[selectedShip].Ip).To4()
	port, _ := strconv.Atoi(shipList.Ships[selectedShip].Port)

	return c.Send(&packets.Redirect{
		Header: packets.BBHeader{Type: packets.RedirectType},
		IPAddr: [4]uint8{ip[0], ip[1], ip[2], ip[3]},
		Port:   uint16(port),
	})
}
