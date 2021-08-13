package character

import (
	"context"
	"fmt"
	"hash/crc32"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf16"

	"github.com/spf13/viper"

	"github.com/dcrodman/archon"
	"github.com/dcrodman/archon/internal/client"
	"github.com/dcrodman/archon/internal/core/auth"
	"github.com/dcrodman/archon/internal/core/bytes"
	"github.com/dcrodman/archon/internal/core/data"
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

// Server is the CHARACTER server implementation. Clients are sent to this server
//  after authenticating with LOGIN. Each client connects to the server in four
// different phases (each one is a new connection):
// 	1. Data download (login options, guildcard, and character previews).
//  2. Character selection
//  3. (Optional) Character creation/modification (recreate and dressing room)
//  4. Confirmation and ship selection
//
// The ship list is obtained by communicating with the shipgate server since ships
// do not directly connect to this server.
type Server struct {
	name           string
	kvCache        *Cache
	shipGateClient *shipgate.Client
	shipGateAddr   string
}

func NewServer(name string, shipgateAddr string) *Server {
	return &Server{
		name:         name,
		kvCache:      NewCache(),
		shipGateAddr: shipgateAddr,
	}
}

func (s *Server) Name() string {
	return s.name
}

func (s *Server) Init(ctx context.Context) error {
	var err error
	s.shipGateClient, err = shipgate.NewClient(s.shipGateAddr)
	if err != nil {
		return err
	}

	if err := initParameterData(); err != nil {
		return err
	}

	// Start the loop that retrieves the ship list from the shipgate.
	if err := s.shipGateClient.StartShipRefreshLoop(ctx); err != nil {
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
		err = s.handleOptionsRequest(c)
	case packets.LoginCharPreviewReqType:
		var pkt packets.CharacterSelection
		bytes.StructFromBytes(data, &pkt)
		err = s.handleCharacterSelect(c, &pkt)
	case packets.LoginChecksumType:
		// Everybody else seems to ignore this, so...
		err = s.sendChecksumAck(c)
	case packets.LoginGuildcardReqType:
		err = s.handleGuildcardDataStart(c)
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
		err = s.handleCharacterUpdate(c, &charPkt)
	case packets.MenuSelectType:
		var menuSelectionPkt packets.MenuSelection
		bytes.StructFromBytes(data, &menuSelectionPkt)
		err = s.handleShipSelection(c, &menuSelectionPkt)
	case packets.DisconnectType:
		// Just wait for the client to disconnect.
		break
	default:
		archon.Log.Infof("received unknown packet %x from %s", packetHeader.Type, c.IPAddr())
	}
	return err
}

func (s *Server) handleLogin(ctx context.Context, c *client.Client, loginPkt *packets.Login) error {
	username := string(bytes.StripPadding(loginPkt.Username[:]))
	password := string(bytes.StripPadding(loginPkt.Password[:]))

	account, err := s.shipGateClient.AuthenticateAccount(ctx, username, password)
	if err != nil {
		switch err {
		case auth.ErrInvalidCredentials:
			return s.sendSecurity(c, packets.BBLoginErrorPassword)
		case auth.ErrAccountBanned:
			return s.sendSecurity(c, packets.BBLoginErrorBanned)
		default:
			sendErr := s.sendMessage(c, strings.Title(err.Error()))
			if sendErr == nil {
				return sendErr
			}
			return err
		}
	}

	if err := s.sendSecurity(c, packets.BBLoginErrorNone); err != nil {
		return err
	}

	c.TeamID = uint32(account.TeamID)
	c.Guildcard = uint32(account.Guildcard)
	c.Account = account

	// At this point, the user has chosen (or created) a character and the
	// client needs the ship list.
	if loginPkt.Phase == packets.ShipSelection {
		if err = s.sendTimestamp(c); err != nil {
			return err
		}
		if err = s.sendShipList(c); err != nil {
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
	// Constants set according to how Newserv does it.
	return c.Send(&packets.Security{
		Header:       packets.BBHeader{Type: packets.LoginSecurityType},
		ErrorCode:    errorCode,
		PlayerTag:    0x00010000,
		Guildcard:    c.Guildcard,
		TeamID:       c.TeamID,
		Config:       c.Config,
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
func (s *Server) sendShipList(c *client.Client) error {
	shipList := s.shipGateClient.GetConnectedShipList()

	pkt := &packets.ShipList{
		Header: packets.BBHeader{
			Type:  packets.LoginShipListType,
			Flags: uint32(len(shipList)),
		},
		Unknown:     0x20,
		Unknown2:    0xFFFFFFF4,
		Unknown3:    0x04,
		ShipEntries: shipList,
	}
	copy(pkt.ServerName[:], bytes.ConvertToUtf16("Archon"))

	return c.Send(pkt)
}

// send whatever scrolling message was read out of the config file for the login screen.
func (s *Server) sendScrollMessage(c *client.Client) error {
	// Returns the scroll message displayed along the top of the ship selection screen,
	// lazily computing it from the config file and storing it in a package var.
	shipSelectionScrollMessageInit.Do(func() {
		shipSelectionScrollMessage = bytes.ConvertToUtf16(
			viper.GetString("character_server.scroll_message"),
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
func (s *Server) handleOptionsRequest(c *client.Client) error {
	account := c.Account
	playerOptions, err := data.FindPlayerOptions(account)
	if err != nil {
		return err
	}

	if playerOptions == nil {
		// We don't have any saved key config - give them the defaults.
		playerOptions = &data.PlayerOptions{
			Account:   account,
			KeyConfig: make([]byte, 420),
		}
		copy(playerOptions.KeyConfig, BaseKeyConfig[:])

		if err = data.CreatePlayerOptions(playerOptions); err != nil {
			return err
		}
	}

	return s.sendOptions(c, playerOptions.KeyConfig)
}

// send the client's configuration options. keyConfig should be 420 bytes long and either
// point to the default keys array or loaded from the database.
func (s *Server) sendOptions(c *client.Client, keyConfig []byte) error {
	if len(keyConfig) != 420 {
		return fmt.Errorf("Received keyConfig of length %d; should be 420", len(keyConfig))
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

// Handle the character select/preview request. For the preview request, this
//method will either send info about a character given a particular slot in an
// 0xE5 response or ack the selection with an 0xE4 (also used for an empty slot).
// The client will send one of these preview request packets for each of the character
// slots (i.e. 4 times). The client also sends this packet when a character has
// been selected from the list and the Selecting flag will be set.
func (s *Server) handleCharacterSelect(c *client.Client, pkt *packets.CharacterSelection) error {
	account := c.Account
	char, err := data.FindCharacter(account, int(pkt.Slot))
	if err != nil {
		return err
	}

	if pkt.Selecting == 0x01 {
		if char == nil {
			return fmt.Errorf("attempted to select nonexistent character in slot: %d", pkt.Slot)
		}
		// They've selected a character from the menu.
		c.Config.SlotNum = uint8(pkt.Slot)
		return s.sendCharacterAck(c, pkt.Slot, 1)
	} else {
		if char == nil {
			// We don't have a character for this slot.
			return s.sendCharacterAck(c, pkt.Slot, 2)
		}
		// They have a character in that slot; send the character preview.
		return s.sendCharacterPreview(c, char)
	}
}

// Send the character acknowledgement packet. Setting flag to 0 indicates a creation
// ack, 1 acks a selected character, and 2 indicates that a character doesn't exist
// in the slot requested via preview request.
func (s *Server) sendCharacterAck(c *client.Client, slotNum uint32, flag uint32) error {
	return c.Send(&packets.CharacterAck{
		Header: packets.BBHeader{Type: packets.LoginCharAckType},
		Slot:   slotNum,
		Flag:   flag,
	})
}

// send the preview packet containing basic details about a character in the selected slot.
func (s *Server) sendCharacterPreview(c *client.Client, char *data.Character) error {
	previewPacket := &packets.CharacterSummary{
		Header: packets.BBHeader{Type: packets.LoginCharPreviewType},
		Slot:   0,
		Character: packets.CharacterPreview{
			Experience:     char.Experience,
			Level:          char.Level,
			NameColor:      char.NameColor,
			Model:          char.ModelType,
			NameColorChksm: char.NameColorChecksum,
			SectionID:      char.SectionID,
			Class:          char.Class,
			V2Flags:        char.V2Flags,
			Version:        char.Version,
			V1Flags:        char.V1Flags,
			Costume:        char.Costume,
			Skin:           char.Skin,
			Face:           char.Face,
			Head:           char.Head,
			Hair:           char.Hair,
			HairRed:        char.HairRed,
			HairGreen:      char.HairGreen,
			HairBlue:       char.HairBlue,
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
func (s *Server) handleGuildcardDataStart(c *client.Client) error {
	account := c.Account
	guildcards, err := data.FindGuildcardEntries(account)
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
	s.kvCache.Set(clientFlagKey(c), c.Flag, -1)
}

func clientFlagKey(c *client.Client) string {
	return fmt.Sprintf("client-flags-%d", c.Account.ID)
}

// Performs a create or update/delete depending on whether the user followed the
// "dressing room" or "recreate" flows (as indicated by a client flag).
func (s *Server) handleCharacterUpdate(c *client.Client, charPkt *packets.CharacterSummary) error {
	if s.hasDressingRoomFlag(c) {
		// "Dressing room"; a request to update an existing character.
		if err := s.updateCharacter(c, charPkt); err != nil {
			archon.Log.Error(err.Error())
			return err
		}
	} else {
		// The "recreate" option. This is a request to create a character in a slot and is used
		// for both creating new characters and replacing existing ones.
		account := c.Account
		existingCharacter, err := data.FindCharacter(account, int(charPkt.Slot))
		if err != nil {
			msg := fmt.Errorf("failed to locate character in slot %d for account %d", charPkt.Slot, account.ID)
			archon.Log.Error(msg)
			return msg
		}
		if existingCharacter != nil {
			if err := data.DeleteCharacter(existingCharacter); err != nil {
				archon.Log.Error(err.Error())
				return err
			}
		}

		p := charPkt.Character
		stats := BaseStats[p.Class]

		newCharacter := &data.Character{
			Account:           account,
			Guildcard:         account.Guildcard,
			GuildcardStr:      p.GuildcardStr[:],
			Slot:              charPkt.Slot,
			Experience:        0,
			Level:             0,
			NameColor:         p.NameColor,
			ModelType:         p.Model,
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
			Meseta:            StartingMeseta,
		}
		// The string is UTF-16LE encoded and it needs to be converted from []uint8 to
		// a []uint16 slice with the bytes reversed.
		// Also drops what is presumably the language code (0x09006900) off of the front.
		cleanedName := p.Name[4:]
		utfName := make([]uint16, 0)
		for i, j := 0, 0; i <= len(cleanedName)-2; i += 2 {
			if cleanedName[i]|cleanedName[i+1] == 0 {
				break
			}
			utfName = append(utfName, uint16(cleanedName[i])|uint16(cleanedName[i+1]<<4))
			j++
		}
		newCharacter.ReadableName = string(utf16.Decode(utfName))

		// TODO: Add the rest of these.
		//--unsigned char keyConfig[232]; // 0x3E8 - 0x4CF;
		//--techniques blob,
		//--options blob,

		if err := data.CreateCharacter(newCharacter); err != nil {
			return err
		}
	}

	c.Config.SlotNum = uint8(charPkt.Slot)
	return s.sendCharacterAck(c, charPkt.Slot, 0)
}

func (s *Server) hasDressingRoomFlag(c *client.Client) bool {
	if (c.Flag & 0x02) != 0 {
		return true
	}

	flags, found := s.kvCache.Get(clientFlagKey(c))
	if found {
		return (flags.(uint32) & 0x02) != 0
	}
	return false
}

func (s *Server) updateCharacter(c *client.Client, pkt *packets.CharacterSummary) error {
	// Clear the dressing room flag so that it doesn't get stuck and cause problems.
	flags, _ := s.kvCache.Get(clientFlagKey(c))
	s.kvCache.Set(clientFlagKey(c), flags.(uint32)^0x02, -1)

	account := c.Account
	char, err := data.FindCharacter(account, int(pkt.Slot))
	if err != nil {
		return err
	} else if char == nil {
		return fmt.Errorf("character does not exist in slot %d for guildcard %d", pkt.Slot, c.Guildcard)
	}

	p := pkt.Character
	char.NameColor = p.NameColor
	char.ModelType = p.Model
	char.NameColorChecksum = p.NameColorChksm
	char.SectionID = p.SectionID
	char.Class = p.Class
	char.Costume = p.Costume
	char.Skin = p.Skin
	char.Head = p.Head
	char.HairRed = p.HairRed
	char.HairGreen = p.HairGreen
	char.HairBlue = p.HairBlue
	char.ProportionX = p.PropX
	char.ProportionY = p.PropY
	copy(char.Name, p.Name[:])

	return data.UpdateCharacter(char)
}

// Player selected one of the items on the ship select screen; respond with the
// IP address and port of the ship server to  which the client will connect after
// disconnecting from this server.
func (s *Server) handleShipSelection(c *client.Client, menuSelectionPkt *packets.MenuSelection) error {
	selectedShip := menuSelectionPkt.ItemID - 1
	ip, port, err := s.shipGateClient.GetSelectedShipAddress(selectedShip)
	if err != nil {
		return fmt.Errorf("could not get selected ship: %d", selectedShip)
	}
	return c.Send(&packets.Redirect{
		Header: packets.BBHeader{Type: packets.RedirectType},
		IPAddr: [4]uint8{ip[0], ip[1], ip[2], ip[3]},
		Port:   uint16(port),
	})
}
