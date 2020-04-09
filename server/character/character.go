// The CHARACTER server logic.
package character

import (
	"errors"
	"fmt"
	"github.com/dcrodman/archon"
	"github.com/dcrodman/archon/data"
	"github.com/dcrodman/archon/internal/auth"
	crypto "github.com/dcrodman/archon/internal/encryption"
	"github.com/dcrodman/archon/server"
	"github.com/dcrodman/archon/server/internal"
	"github.com/dcrodman/archon/server/internal/cache"
	"github.com/dcrodman/archon/server/internal/relay"
	"github.com/dcrodman/archon/server/shipgate"
	"github.com/spf13/viper"
	"hash/crc32"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf16"
)

const (
	// Maximum size of a block of parameter or guildcard data.
	MaxDataChunkSize = 0x6800
	// Expected format of the timestamp sent to the client.
	TimeFormat = "2006:01:02: 15:05:05"
	// Id sent in the menu selection packet to tell the client
	// that the selection was made on the ship menu.
	ShipSelectionMenuId uint16 = 0x13
)

var (
	loginCopyright = []byte("Phantasy Star Online Blue Burst Game Server. Copyright 1999-2004 SONICTEAM.")

	cachedScrollMessage     []byte
	cachedScrollMessageInit sync.Once
)

func GetScrollMessage() []byte {
	cachedScrollMessageInit.Do(func() {
		cachedScrollMessage = internal.ConvertToUtf16(viper.GetString("character_server.scroll_message"))
	})
	return cachedScrollMessage
}

// Entry in the available ships lis on the ship selection menu.
type ShipMenuEntry struct {
	MenuId  uint16
	ShipId  uint32
	Padding uint16

	ShipName [23]byte
}

type CharacterServer struct {
	name string
	port string

	kvCache *cache.Cache

	// Connected ships. Each Ship's id corresponds to its position in the array - 1.
	shipList []shipgate.Ship
}

func NewServer(name, port string) server.Server {
	initParameterData()
	return &CharacterServer{
		name:    name,
		port:    port,
		kvCache: cache.New(),
	}
}

func (s *CharacterServer) Name() string       { return s.name }
func (s *CharacterServer) Port() string       { return s.port }
func (s *CharacterServer) HeaderSize() uint16 { return archon.BBHeaderSize }

func (s *CharacterServer) AcceptClient(cs *server.ConnectionState) (server.Client2, error) {
	c := &Client{
		cs:          cs,
		serverCrypt: crypto.NewBBCrypt(),
		clientCrypt: crypto.NewBBCrypt(),
	}

	if err := s.SendWelcome(c); err != nil {
		return nil, fmt.Errorf("error sending welcome packet to %s: %s", cs.IPAddr(), err)
	}
	return c, nil
}

func (s *CharacterServer) SendWelcome(c *Client) error {
	pkt := &archon.WelcomePkt{
		Header:       archon.BBHeader{Type: archon.LoginWelcomeType, Size: 0xC8},
		Copyright:    [96]byte{},
		ServerVector: [48]byte{},
		ClientVector: [48]byte{},
	}
	copy(pkt.Copyright[:], loginCopyright)
	copy(pkt.ServerVector[:], c.serverVector())
	copy(pkt.ClientVector[:], c.clientVector())

	return c.sendRaw(pkt)
}

func (s *CharacterServer) Handle(client server.Client2) error {
	c := client.(*Client)

	var packetHeader archon.BBHeader
	internal.StructFromBytes(c.ConnectionState().Data()[:archon.BBHeaderSize], &packetHeader)

	var err error
	switch packetHeader.Type {
	case archon.LoginType:
		err = s.handleLogin(c)
	case archon.LoginOptionsRequestType:
		err = s.handleOptionsRequest(c)
	case archon.LoginCharPreviewReqType:
		err = s.handleCharacterSelect(c)
	case archon.LoginChecksumType:
		// Everybody else seems to ignore this, so...
		err = s.sendChecksumAck(c)
	case archon.LoginGuildcardReqType:
		err = s.HandleGuildcardDataStart(c)
	case archon.LoginGuildcardChunkReqType:
		err = s.handleGuildcardChunk(c)
	case archon.LoginParameterHeaderReqType:
		err = s.sendParameterHeader(c, uint32(len(paramFiles)), paramHeaderData)
	case archon.LoginParameterChunkReqType:
		var pkt archon.BBHeader
		internal.StructFromBytes(c.ConnectionState().Data(), &pkt)
		err = s.sendParameterChunk(c, paramChunkData[int(pkt.Flags)], pkt.Flags)
	case archon.LoginSetFlagType:
		s.setClientFlag(c)
	case archon.LoginCharPreviewType:
		err = s.handleCharacterUpdate(c)
	case archon.MenuSelectType:
		err = s.handleShipSelection(c)
	case archon.DisconnectType:
		// Just wait until we recv 0 from the client to disconnect.
		break
	default:
		archon.Log.Infof("Received unknown packet %x from %s", packetHeader.Type, c.ConnectionState().IPAddr())
	}
	return err
}

func (s *CharacterServer) handleLogin(c *Client) error {
	var loginPkt archon.LoginPkt
	internal.StructFromBytes(c.ConnectionState().Data(), &loginPkt)

	account, err := auth.VerifyAccount(
		string(internal.StripPadding(loginPkt.Username[:])),
		string(internal.StripPadding(loginPkt.Password[:])),
	)

	if err != nil {
		switch err {
		case auth.ErrInvalidCredentials:
			return s.sendSecurity(c, archon.BBLoginErrorPassword)
		case auth.ErrAccountBanned:
			return s.sendSecurity(c, archon.BBLoginErrorBanned)
		default:
			sendErr := s.sendMessage(c, strings.Title(err.Error()))
			if sendErr == nil {
				return sendErr
			}
			return err
		}
	}

	c.account = account
	c.TeamId = uint32(account.TeamID)
	c.Guildcard = uint32(account.Guildcard)

	if err = s.sendSecurity(c, archon.BBLoginErrorNone); err != nil {
		return err
	}

	// At this point, if we've chosen (or created) a character then the
	// client will send us the slot number and the corresponding phase.
	if loginPkt.SlotNum >= 0 && loginPkt.Phase == 4 {
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
func (s *CharacterServer) sendSecurity(c *Client, errorCode uint32) error {
	// Constants set according to how Newserv does it.
	return c.send(&archon.SecurityPacket{
		Header:       archon.BBHeader{Type: archon.LoginSecurityType},
		ErrorCode:    errorCode,
		PlayerTag:    0x00010000,
		Guildcard:    c.Guildcard,
		TeamId:       c.TeamId,
		Config:       &c.Config,
		Capabilities: 0x00000102,
	})
}

// Sends a message to the client. In this case whatever message is sent
// here will be displayed in a dialog box after the patch screen.
func (s *CharacterServer) sendMessage(c *Client, message string) error {
	return c.send(&archon.LoginClientMessagePacket{
		Header:   archon.BBHeader{Type: archon.LoginClientMessageType},
		Language: 0x00450009,
		Message:  internal.ConvertToUtf16(message),
	})
}

// send a timestamp packet in order to indicate the server's current time.
func (s *CharacterServer) sendTimestamp(client *Client) error {
	pkt := &archon.TimestampPacket{
		Header:    archon.BBHeader{Type: archon.LoginTimestampType},
		Timestamp: [28]byte{},
	}

	var tv syscall.Timeval
	syscall.Gettimeofday(&tv)
	t := time.Now().Format(TimeFormat)
	stamp := fmt.Sprintf("%s.%03d", t, uint64(tv.Usec/1000))
	copy(pkt.Timestamp[:], stamp)

	return client.send(pkt)
}

// send the menu items for the ship select screen.
func (s *CharacterServer) sendShipList(c *Client) error {
	pkt := &archon.ShipListPacket{
		Header:   archon.BBHeader{Type: archon.LoginShipListType, Flags: 0x01},
		Unknown:  0x02,
		Unknown2: 0xFFFFFFF4,
		Unknown3: 0x04,
		//ShipEntries: make([]ShipMenuEntry, len(ships)),
	}
	copy(pkt.ServerName[:], "Archon")

	// TODO: Will eventually need a mutex for read.
	//for i, ship := range ships {
	//	item := &pkt.ShipEntries[i]
	//	item.MenuId = ShipSelectionMenuId
	//	item.ShipId = ship.id
	//	copy(item.ShipName[:], util.ConvertToUtf16(string(ship.name[:])))
	//}

	return c.send(pkt)
}

// send whatever scrolling message was read out of the config file for the login screen.

func (s *CharacterServer) sendScrollMessage(c *Client) error {
	pkt := &archon.ScrollMessagePacket{
		Header:  archon.BBHeader{Type: archon.LoginScrollMessageType},
		Message: GetScrollMessage(),
	}

	// The end of the message appears to be garbled unless there is a block of extra bytes
	// on the end; add an extra and let fixLength add the rest.
	pktData, size := internal.BytesFromStruct(pkt)
	pktData = append(pktData, 0x00)

	return relay.SendRaw(c, pktData, uint16(size+1))
}

// Load key config and other option data from the database or provide defaults for new accounts.
func (s *CharacterServer) handleOptionsRequest(c *Client) error {
	playerOptions, err := data.FindPlayerOptions(c.account)
	if err != nil {
		return err
	}

	if playerOptions == nil {
		// We don't have any saved key config - give them the defaults.
		playerOptions = &data.PlayerOptions{
			Account:   *c.account,
			KeyConfig: make([]byte, 420),
		}
		copy(playerOptions.KeyConfig, archon.BaseKeyConfig[:])

		if err = data.UpdatePlayerOptions(playerOptions); err != nil {
			return err
		}
	}

	return s.sendOptions(c, playerOptions.KeyConfig)
}

// send the client's configuration options. keyConfig should be 420 bytes long and either
// point to the default keys array or loaded from the database.
func (s *CharacterServer) sendOptions(c *Client, keyConfig []byte) error {
	if len(keyConfig) != 420 {
		return fmt.Errorf("Received keyConfig of length " + string(len(keyConfig)) + "; should be 420")
	}

	pkt := &archon.OptionsPacket{
		Header:          archon.BBHeader{Type: archon.LoginOptionsType},
		PlayerKeyConfig: archon.KeyTeamConfig{Guildcard: c.Guildcard},
	}
	copy(pkt.PlayerKeyConfig.KeyConfig[:], keyConfig[:0x16C])
	copy(pkt.PlayerKeyConfig.JoystickConfig[:], keyConfig[0x16C:])

	// Sylverant sets these to enable all team rewards? Not sure what this means yet.
	pkt.PlayerKeyConfig.TeamRewards[0] = 0xFFFFFFFF
	pkt.PlayerKeyConfig.TeamRewards[1] = 0xFFFFFFFF

	return c.send(pkt)
}

// Handle the character select/preview request. Will either return information
// about a character given a particular slot in an 0xE5 response or ack the
// selection with an 0xE4 (also used for an empty slot). The client will send
// one of these packets for each of the character slots (i.e. 4 times).
func (s *CharacterServer) handleCharacterSelect(c *Client) error {
	var pkt archon.CharSelectionPacket
	internal.StructFromBytes(c.ConnectionState().Data(), &pkt)

	character, err := data.FindCharacter(c.account, int(pkt.Slot))

	if character == nil {
		// We don't have a character for this slot.
		return s.sendCharacterAck(c, pkt.Slot, 2)
	} else if err != nil {
		archon.Log.Error(err.Error())
		return err
	}

	if pkt.Selecting == 0x01 {
		// They've selected a character from the menu.
		c.Config.SlotNum = uint8(pkt.Slot)
		if err := s.sendSecurity(c, archon.BBLoginErrorNone); err != nil {
			return err
		}
		return s.sendCharacterAck(c, pkt.Slot, 1)
	}

	// They have a character in that slot; send the character preview.
	return s.sendCharacterPreview(c, character)
}

// Send the character acknowledgement packet. Setting flag to 0 indicates a creation
// ack, 1 acks a selected character, and 2 indicates that a character doesn't exist
// in the slot requested via preview request.
func (s *CharacterServer) sendCharacterAck(c *Client, slotNum uint32, flag uint32) error {
	return c.send(&archon.CharAckPacket{
		Header: archon.BBHeader{Type: archon.LoginCharAckType},
		Slot:   slotNum,
		Flag:   flag,
	})
}

// send the preview packet containing basic details about a character in the selected slot.
func (s *CharacterServer) sendCharacterPreview(c *Client, character *data.Character) error {
	previewPacket := &archon.CharacterSummaryPacket{
		Header: archon.BBHeader{Type: archon.LoginCharPreviewType},
		Slot:   0,
		Character: archon.CharacterSummary{
			Experience:     character.Experience,
			Level:          character.Level,
			NameColor:      character.NameColor,
			Model:          character.ModelType,
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
		},
	}
	copy(previewPacket.Character.GuildcardStr[:], character.GuildcardStr[:])
	copy(previewPacket.Character.Name[:], character.Name[:])

	return c.send(previewPacket)
}

// Acknowledge the checksum the client sent us. We don't actually do
// anything with it but the client won't proceed otherwise.
func (s *CharacterServer) sendChecksumAck(c *Client) error {
	return c.send(&archon.ChecksumAckPacket{
		Header: archon.BBHeader{Type: archon.LoginChecksumAckType},
		Ack:    1,
	})
}

// Load the player's saved guildcards, build the chunk data, and send the chunk header.
func (s *CharacterServer) HandleGuildcardDataStart(c *Client) error {
	guildcards, err := data.FindGuildcardEntries(c.account)
	if err != nil {
		return err
	}

	gcData := new(archon.GuildcardData)
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
	c.GuildcardData, size = internal.BytesFromStruct(gcData)
	checksum := crc32.ChecksumIEEE(c.GuildcardData)

	return s.sendGuildcardHeader(c, checksum, uint16(size))
}

// send the header containing metadata about the guildcard chunk.
func (s *CharacterServer) sendGuildcardHeader(c *Client, checksum uint32, dataLen uint16) error {
	return c.send(&archon.GuildcardHeaderPacket{
		Header:   archon.BBHeader{Type: archon.LoginGuildcardHeaderType},
		Unknown:  0x00000001,
		Length:   dataLen,
		Checksum: checksum,
	})
}

// send another chunk of the client's guildcard data.
func (s *CharacterServer) handleGuildcardChunk(c *Client) error {
	var chunkReq archon.GuildcardChunkReqPacket
	internal.StructFromBytes(c.ConnectionState().Data(), &chunkReq)

	if chunkReq.Continue == 0x01 {
		return s.sendGuildcardChunk(c, chunkReq.ChunkRequested)
	}
	// Anything else is a request to cancel sending guildcard chunks.
	return nil
}

// send the specified chunk of guildcard data.
func (s *CharacterServer) sendGuildcardChunk(c *Client, chunkNum uint32) error {
	pkt := &archon.GuildcardChunkPacket{
		Header: archon.BBHeader{Type: archon.LoginGuildcardChunkType},
		Chunk:  chunkNum,
	}

	// The client will only accept 0x6800 bytes of a chunk per packet.
	offset := uint16(chunkNum) * MaxDataChunkSize
	remaining := uint16(len(c.GuildcardData)) - offset

	if remaining > MaxDataChunkSize {
		pkt.Data = c.GuildcardData[offset : offset+MaxDataChunkSize]
	} else {
		pkt.Data = c.GuildcardData[offset:]
	}

	return c.send(pkt)
}

// send the header for the parameter files we're about to start sending.
func (s *CharacterServer) sendParameterHeader(c *Client, numEntries uint32, entries []byte) error {
	return c.send(&archon.ParameterHeaderPacket{
		Header: archon.BBHeader{
			Type:  archon.LoginParameterHeaderType,
			Flags: numEntries,
		},
		Entries: entries,
	})
}

// Index into chunkData and send the specified chunk of parameter data.
func (s *CharacterServer) sendParameterChunk(c *Client, chunkData []byte, chunk uint32) error {
	return c.send(&archon.ParameterChunkPacket{
		Header: archon.BBHeader{Type: archon.LoginParameterChunkType},
		Chunk:  chunk,
		Data:   chunkData,
	})
}

// The client may send us flags as a result of user actions in order to indicate
// a change in state or desired behavior. For instance, setting 0x02 indicates
// that the character dressing room has been opened.
func (s *CharacterServer) setClientFlag(c *Client) {
	var pkt archon.SetFlagPacket
	internal.StructFromBytes(c.ConnectionState().Data(), &pkt)

	c.Flag = c.Flag | pkt.Flag
	// Some flags are set right before the client disconnects, which means saving them
	// on the Client alone isn't safe since the state is lost. To fix this the flags are
	// also kept in memory to avoid bugs like accidentally recreating characters.
	s.kvCache.Set(clientFlagKey(c), c.Flag, -1)
}

func clientFlagKey(c *Client) string {
	return fmt.Sprintf("client-flags-%d", c.account.ID)
}

// Performs a create or update/delete depending on whether the user followed the
// "dressing room" or "recreate" flows (as indicated by a client flag).
func (s *CharacterServer) handleCharacterUpdate(c *Client) error {
	var charPkt archon.CharacterSummaryPacket
	internal.StructFromBytes(c.ConnectionState().Data(), &charPkt)

	if s.hasDressingRoomFlag(c) {
		// "Dressing room"; a request to update an existing character.
		if err := s.updateCharacter(c, &charPkt); err != nil {
			archon.Log.Error(err.Error())
			return err
		}
	} else {
		// The "recreate" option. This is a request to create a character in a slot and is used
		// for both creating new characters and replacing existing ones.
		existingCharacter, err := c.account.FindCharacterInSlot(int(charPkt.Slot))
		if err != nil {
			msg := fmt.Errorf("failed to locate character in slot %d for account %d", charPkt.Slot, c.account.ID)
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
			Account:           c.account,
			Guildcard:         c.account.Guildcard,
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
			utfName = append(utfName, uint16(cleanedName[i])|uint16(cleanedName[i+1]<<8))
			j += 1
		}
		newCharacter.ReadableName = string(utf16.Decode(utfName))

		/* TODO: Add the rest of these.
		--unsigned char keyConfig[232]; // 0x3E8 - 0x4CF;
		--techniques blob,
		--options blob,
		*/

		if err := data.CreateCharacter(newCharacter); err != nil {
			return err
		}
	}

	c.Config.SlotNum = uint8(charPkt.Slot)
	return s.sendCharacterAck(c, charPkt.Slot, 0)
}

func (s *CharacterServer) hasDressingRoomFlag(c *Client) bool {
	if (c.Flag & 0x02) != 0 {
		return true
	}

	flags, found := s.kvCache.Get(clientFlagKey(c))
	if found {
		return (flags.(uint32) & 0x02) != 0
	}
	return false
}

func (s *CharacterServer) updateCharacter(c *Client, pkt *archon.CharacterSummaryPacket) error {
	// Clear the dressing room flag so that it doesn't get stuck and cause problems.
	flags, _ := s.kvCache.Get(clientFlagKey(c))
	s.kvCache.Set(clientFlagKey(c), flags.(uint32)^0x02, -1)

	character, err := c.account.FindCharacterInSlot(int(pkt.Slot))
	if err != nil {
		return err
	} else if character == nil {
		return fmt.Errorf("character does not exist in slot %d for guildcard %d", pkt.Slot, c.Guildcard)
	}

	p := pkt.Character
	character.NameColor = p.NameColor
	character.ModelType = p.Model
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

	return data.UpdateCharacter(character)
}

// Player selected one of the items on the ship select screen; respond with the
// IP address and port of the ship server to  which the client will connect after
// disconnecting from this server.
func (s *CharacterServer) handleShipSelection(c *Client) error {
	var pkt archon.MenuSelectionPacket
	internal.StructFromBytes(c.ConnectionState().Data(), &pkt)

	selectedShip := pkt.ItemId - 1
	if selectedShip < 0 || selectedShip >= uint32(len(s.shipList)) {
		return errors.New("Invalid ship selection: " + string(selectedShip))
	}

	//ship := &s.shipList[selectedShip]

	return c.send(&archon.RedirectPacket{
		Header: archon.BBHeader{Type: archon.RedirectType},
		// TODO: Ship IP address and port
		//IPAddr: [4]uint8{},
		//Port:   s.charRedirectPort,
	})
}
