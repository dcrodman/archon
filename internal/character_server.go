package internal

import (
	"context"
	"fmt"
	"hash/crc32"
	"net"
	"sync"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/dcrodman/archon/internal/commands"
	"github.com/dcrodman/archon/internal/data"
	"github.com/dcrodman/archon/internal/encryption"
	"github.com/dcrodman/archon/internal/shipgate"
)

var (
	copyright = []byte("Phantasy Star Online Blue Burst Game Server. Copyright 1999-2004 SONICTEAM.")

	// Scrolling message that appears across the top of the ship selection screen.
	shipSelectionScrollMessage     []byte
	shipSelectionScrollMessageInit sync.Once
)

// Server contains the bulk of the implementation for character management and
// selection. Clients are redirected here after authenticating with [GameAuthServer].
// Each client can connect to this server in up to four different phases, with
// each phase as a new connection:
//
//  1. Data download (login options, guildcard, and character previews).
//  2. Character selection
//  3. (Optional) Character creation/modification (recreate and dressing room)
//  4. Confirmation and ship selection
//
// The ship list is obtained by communicating with the shipgate server since ships
// do not directly connect to this server.
type CharacterServer struct {
	numParameterFiles int
}

func (s *CharacterServer) Identifier() string {
	return "CHARACTER:DATA"
}

func (s *CharacterServer) Init(ctx context.Context) error {
	var err error
	if s.numParameterFiles, err = initParameterData(); err != nil {
		return err
	}
	return nil
}

func (s *CharacterServer) Handshake(ctx context.Context, c *Client) error {
	c.CryptoSession = encryption.NewBlueBurstCryptoSession()

	pkt := &commands.Welcome{
		Header:       commands.BBHeader{Type: commands.LoginWelcomeType, Size: 0xC8},
		Copyright:    [96]byte{},
		ServerVector: [48]byte{},
		ClientVector: [48]byte{},
	}
	copy(pkt.Copyright[:], copyright)
	copy(pkt.ServerVector[:], c.CryptoSession.ServerVector())
	copy(pkt.ClientVector[:], c.CryptoSession.ClientVector())

	return c.SendRaw(ctx, pkt)
}

func (s *CharacterServer) Handle(ctx context.Context, c *Client, data []byte) error {
	var cmdHeader commands.BBHeader
	UnmarshalStruct(data[:commands.BBHeaderSize], &cmdHeader)

	var err error
	switch cmdHeader.Type {
	case commands.LoginType:
		var loginPkt commands.Login
		UnmarshalStruct(data, &loginPkt)
		err = s.handleLogin(ctx, c, &loginPkt)
	case commands.OptionsRequestType:
		err = s.handleOptionsRequest(ctx, c)
	case commands.CharacterSelectionType:
		var pkt commands.CharacterSelection
		UnmarshalStruct(data, &pkt)
		err = s.handleCharacterSelect(ctx, c, &pkt)
	case commands.ChecksumType:
		// Everybody else seems to ignore this, so...
		err = SendChecksumAck(ctx, c)
	case commands.GuildcardRequestType:
		err = s.handleGuildcardDataStart(ctx, c)
	case commands.GuildcardChunkReqType:
		var chunkReq commands.GuildcardChunkRequest
		UnmarshalStruct(data, &chunkReq)
		err = s.handleGuildcardChunk(ctx, c, &chunkReq)
	case commands.ParameterHeaderReqType:
		err = SendParameterHeader(ctx, c, uint32(s.numParameterFiles), paramHeaderData)
	case commands.ParameterChunkReqType:
		var pkt commands.BBHeader
		UnmarshalStruct(data, &pkt)
		err = SendParameterChunk(ctx, c, paramChunkData[int(pkt.Flags)], pkt.Flags)
	case commands.SetFlagType:
		// Ignored.
	case commands.CharacterSummaryType:
		var charPkt commands.CharacterSummary
		UnmarshalStruct(data, &charPkt)
		err = s.handleCharacterUpdate(ctx, c, &charPkt)
	case commands.MenuSelectionType:
		var menuSelectionPkt commands.MenuSelection
		UnmarshalStruct(data, &menuSelectionPkt)
		err = s.handleShipSelection(ctx, c, &menuSelectionPkt)
	case commands.DisconnectType:
		// Just wait for the client to disconnect.
		break
	default:
		Logger.Infof("received unknown command %x from %s", cmdHeader.Type, c.IPAddr)
	}
	return err
}

func (s *CharacterServer) handleLogin(ctx context.Context, c *Client, loginPkt *commands.Login) error {
	username := string(StripPadding(loginPkt.Username[:]))
	password := string(StripPadding(loginPkt.Password[:]))

	account, err := shipgate.Shipgate.AuthenticateAccount(ctx, username, password)
	if err != nil {
		switch err {
		case shipgate.ErrInvalidCredentials:
			return SendSecurity(ctx, c, commands.BBLoginErrorPassword)
		case shipgate.ErrAccountBanned:
			return SendSecurity(ctx, c, commands.BBLoginErrorBanned)
		default:
			sendErr := SendMessage(ctx, c, cases.Title(language.English).String(err.Error()))
			if sendErr == nil {
				return sendErr
			}
			return err
		}
	}
	c.Account = account
	c.LoginPhase = loginPkt.Phase

	if err := SendSecurity(ctx, c, commands.BBLoginErrorNone); err != nil {
		return err
	}

	// At this point, the user has chosen (or created) a character and the
	// client needs the ship list.
	if c.LoginPhase == commands.ShipSelection {
		if err = SendTimestamp(ctx, c); err != nil {
			return err
		}
		if err = SendShipList(ctx, c); err != nil {
			return err
		}
		if err = SendScrollMessage(ctx, c); err != nil {
			return err
		}
	}

	return nil
}

// Expected format of the timestamp sent to the client.
const B1TimeFormat = "2006:01:02: 15:05:05.000"

// Send a timestamp command in order to indicate the server's current time.
func SendTimestamp(ctx context.Context, c *Client) error {
	pkt := &commands.Timestamp{
		Header:    commands.BBHeader{Type: commands.TimestampType},
		Timestamp: [28]uint8{},
	}
	stamp := time.Now().Format(B1TimeFormat)
	copy(pkt.Timestamp[:], stamp)

	return c.Send(ctx, pkt)
}

// Send the menu items for the ship select screen. Since we're only supporting
// a single (bundled) ship server for the moment, the entries are hardcoded
// rather than bothering with anything fancy like retrieving a list of active
// ships from the shipgate, etc.

const ShipListMenuID = 0x11111111

func SendShipList(ctx context.Context, c *Client) error {
	entries := []commands.MenuEntry{
		// The first item is ignored and just used for the menu title.
		{MenuID: ShipListMenuID, ItemID: 0},
	}
	copy(entries[0].Name[:], EncodeUTF16LEString("Archon"))

	availableShips := shipgate.Shipgate.GetAvailableShips(ctx)

	// Append our active ship list.
	for i, ship := range availableShips {
		var entry commands.MenuEntry
		entry.ItemID = uint32(i) + 1
		copy(entry.Name[:], EncodeUTF16LEString(ship.Name))
		entries = append(entries, entry)
	}

	return c.Send(ctx, &commands.Menu{
		Header: commands.BBHeader{
			Type:  commands.ShipMenuType,
			Flags: uint32(len(availableShips)),
		},
		Entries: entries,
	})
}

// send whatever scrolling message was read out of the config file for the login screen.
func SendScrollMessage(ctx context.Context, c *Client) error {
	// Returns the scroll message displayed along the top of the ship selection screen,
	// lazily computing it from the config file and storing it in a package var.
	shipSelectionScrollMessageInit.Do(func() {
		shipSelectionScrollMessage = EncodeUTF16LEString(
			Config.CharacterServer.ScrollMessage,
		)
		// The end of the message appears to be garbled unless there is an extra byte...?
		shipSelectionScrollMessage = append(shipSelectionScrollMessage, 0x00)
	})

	return c.Send(ctx, &commands.ScrollMessage{
		Header:  commands.BBHeader{Type: commands.ScrollMessageType},
		Message: shipSelectionScrollMessage,
	})
}

// Player selected one of the items on the ship select screen; respond with the
// IP address and port of the ship server to  which the client will connect after
// disconnecting from this server.
func (s *CharacterServer) handleShipSelection(ctx context.Context, c *Client, menuSelectionPkt *commands.MenuSelection) error {
	availableShips := shipgate.Shipgate.GetAvailableShips(ctx)

	selectedShip := menuSelectionPkt.ItemID - 1
	if selectedShip >= uint32(len(availableShips)) {
		return fmt.Errorf("invalid ship selection: %d", selectedShip)
	}

	var ip [4]uint8
	copy(ip[:], net.ParseIP(availableShips[selectedShip].Address).To4())
	port := uint16(availableShips[selectedShip].Port)

	return SendRedirect(ctx, c, ip, port)
}

// LoadConfig key config and other option data from the database or provide defaults for new accounts.
func (s *CharacterServer) handleOptionsRequest(ctx context.Context, c *Client) error {
	playerOptions, err := shipgate.Shipgate.GetPlayerOptions(ctx, c.Account.ID)
	if err != nil {
		return fmt.Errorf("error handling options request: %w", err)
	}

	if playerOptions == nil {
		// We don't have any saved key config - give them the defaults.
		playerOptions = &data.PlayerOptions{
			KeyConfig: make([]byte, 420),
		}
		copy(playerOptions.KeyConfig, BaseKeyConfig[:])

		err = shipgate.Shipgate.UpsertPlayerOptions(ctx, c.Account.ID, playerOptions)
		if err != nil {
			return fmt.Errorf("error creating player options: %w", err)
		}
	}

	return SendOptions(ctx, c, playerOptions.KeyConfig)
}

// send the client's configuration options. keyConfig should be 420 bytes long and either
// point to the default keys array or loaded from the database.
func SendOptions(ctx context.Context, c *Client, keyConfig []byte) error {
	if len(keyConfig) != 420 {
		return fmt.Errorf("received keyConfig of length %d; should be 420", len(keyConfig))
	}

	pkt := &commands.Options{
		Header: commands.BBHeader{Type: commands.OptionsType},
	}
	pkt.PlayerKeyConfig.Guildcard = uint32(c.Account.Guildcard)
	copy(pkt.PlayerKeyConfig.KeyConfig[:], keyConfig[:0x16C])
	copy(pkt.PlayerKeyConfig.JoystickConfig[:], keyConfig[0x16C:])

	// Sylverant sets these to enable all team rewards? Not sure what this means yet.
	pkt.PlayerKeyConfig.TeamRewards[0] = 0xFFFFFFFF
	pkt.PlayerKeyConfig.TeamRewards[1] = 0xFFFFFFFF

	return c.Send(ctx, pkt)
}

// Handle the character select/preview request.
//
// For the preview request, this method will either send info about a character given
// a particular slot in an 0xE5 response or ack the selection with an 0xE4 (also used
// for an empty slot). The client will send one of these preview request commands for
// each of the character slots (i.e. 4 times).
//
// The client also sends this command when  a character has been selected from the menu
// (or after the dressing room or recreate), as indicated by the Selecting flag.
func (s *CharacterServer) handleCharacterSelect(ctx context.Context, c *Client, pkt *commands.CharacterSelection) error {
	character, err := shipgate.Shipgate.FindCharacter(ctx, c.Account.ID, pkt.Slot)
	if err != nil {
		return fmt.Errorf("error selecting character: %w", err)
	}

	if pkt.Selecting == 0x01 {
		if character == nil {
			return fmt.Errorf("attempted to select nonexistent character in slot: %d", pkt.Slot)
		}
		// They've selected a character from the menu.
		c.Config.SlotNum = uint8(pkt.Slot)
		return SendCharacterAck(ctx, c, pkt.Slot, 1)
	}

	if character != nil {
		// They have a character in that slot; send the character preview.
		return SendCharacterPreview(ctx, c, character)
	}
	// We don't have a character for this slot.
	return SendCharacterAck(ctx, c, pkt.Slot, 2)
}

// Send the character acknowledgement command in response to the action taken. Setting flag
// to 0 indicates a creation ack, 1 acks a selected character, and 2 indicates that a character
// doesn't exist in the slot requested via preview request.
func SendCharacterAck(ctx context.Context, c *Client, slotNum uint32, flag uint32) error {
	return c.Send(ctx, &commands.CharacterSelectionAck{
		Header: commands.BBHeader{Type: commands.CharacterSelectionAckType},
		Slot:   slotNum,
		Flag:   flag,
	})
}

// send the preview command containing basic details about a character in the selected slot.
func SendCharacterPreview(ctx context.Context, c *Client, char *commands.CharacterData) error {
	previewCommand := &commands.CharacterSummary{
		Header:      commands.BBHeader{Type: commands.CharacterSummaryType},
		Slot:        0,
		Experience:  char.DisplayData.Stats.Experience,
		Level:       char.DisplayData.Stats.Level,
		Visual:      char.DisplayData.Visual,
		PlaytimeSec: char.PlayTimeSeconds,
	}
	copy(previewCommand.Name[:], char.GuildCard.Name[:])

	return c.Send(ctx, previewCommand)
}

// Acknowledge the checksum the client sent us. We don't actually do
// anything with it but the client won't proceed otherwise.
func SendChecksumAck(ctx context.Context, c *Client) error {
	return c.Send(ctx, &commands.ChecksumAck{
		Header: commands.BBHeader{Type: commands.ChecksumAckType},
		Ack:    1,
	})
}

// GuildcardData is the per-player guildcard data chunk.
type GuildcardData struct {
	Unknown  [0x114]uint8
	Blocked  [0x1DE8]uint8 //This should be a struct once implemented
	Unknown2 [0x78]uint8
	Entries  [104]GuildcardDataEntry
	Unknown3 [0x1BC]uint8
}

// GuildcardDataEntry is the per-player friend guildcard entries.
type GuildcardDataEntry struct {
	Guildcard   uint32
	Name        [48]byte
	TeamName    [32]byte
	Description [176]byte
	Reserved    uint8
	Language    uint8
	SectionID   uint8
	CharClass   uint8
	Padding     uint32
	Comment     [176]byte
}

// LoadConfig the player's saved guildcards, build the chunk data, and send the chunk header.
func (s *CharacterServer) handleGuildcardDataStart(ctx context.Context, c *Client) error {
	entries, err := shipgate.Shipgate.GetGuildcardEntries(ctx, c.Account.ID)
	if err != nil {
		return fmt.Errorf("error loading guildcards: %w", err)
	}

	gcData := new(GuildcardData)
	// Maximum of 140 entries can be sent.
	for i, entry := range entries {
		// TODO: This may not actually work yet, but I haven't gotten to
		// figuring out how the other servers use it.
		pktEntry := gcData.Entries[i]
		pktEntry.Guildcard = uint32(entry.Guildcard)
		copy(pktEntry.Name[:], entry.Name)
		copy(pktEntry.TeamName[:], entry.TeamName)
		copy(pktEntry.Description[:], entry.Description)
		pktEntry.Language = uint8(entry.Language)
		pktEntry.SectionID = uint8(entry.SectionID)
		pktEntry.CharClass = uint8(entry.Class)
		copy(pktEntry.Comment[:], entry.Comment)
	}

	var size int
	c.GuildcardData, size = MarshalStruct(gcData)
	checksum := crc32.ChecksumIEEE(c.GuildcardData)

	return SendGuildcardHeader(ctx, c, checksum, uint16(size))
}

// send the header containing metadata about the guildcard chunk.
func SendGuildcardHeader(ctx context.Context, c *Client, checksum uint32, dataLen uint16) error {
	return c.Send(ctx, &commands.GuildcardHeader{
		Header:   commands.BBHeader{Type: commands.GuildcardHeaderType},
		Unknown:  0x00000001,
		Length:   dataLen,
		Checksum: checksum,
	})
}

// send another chunk of the client's guildcard data.
func (s *CharacterServer) handleGuildcardChunk(ctx context.Context, c *Client, chunkReq *commands.GuildcardChunkRequest) error {
	if chunkReq.Continue == 0x01 {
		return SendGuildcardChunk(ctx, c, chunkReq.ChunkRequested)
	}
	// Anything else is a request to cancel sending guildcard chunks.
	return nil
}

// Maximum size of a block of parameter or guildcard data.
const maxDataChunkSize = 0x6800

// send the specified chunk of guildcard data.
func SendGuildcardChunk(ctx context.Context, c *Client, chunkNum uint32) error {
	pkt := &commands.GuildcardChunk{
		Header: commands.BBHeader{Type: commands.GuildcardChunkType},
		Chunk:  chunkNum,
	}

	// The client will only accept 0x6800 bytes of a chunk per command.
	offset := uint16(chunkNum) * maxDataChunkSize
	remaining := uint16(len(c.GuildcardData)) - offset

	if remaining > maxDataChunkSize {
		pkt.Data = c.GuildcardData[offset : offset+maxDataChunkSize]
	} else {
		pkt.Data = c.GuildcardData[offset:]
	}

	return c.Send(ctx, pkt)
}

// send the header for the parameter files we're about to start sending.
func SendParameterHeader(ctx context.Context, c *Client, numEntries uint32, entries []byte) error {
	return c.Send(ctx, &commands.ParameterHeader{
		Header: commands.BBHeader{
			Type:  commands.ParameterHeaderType,
			Flags: numEntries,
		},
		Entries: entries,
	})
}

// Index into chunkData and send the specified chunk of parameter data.
func SendParameterChunk(ctx context.Context, c *Client, chunkData []byte, chunk uint32) error {
	return c.Send(ctx, &commands.ParameterChunk{
		Header: commands.BBHeader{Type: commands.ParameterChunkType},
		Chunk:  chunk,
		Data:   chunkData,
	})
}

// Performs a create or update/delete depending on whether the user followed the
// "dressing room" or "recreate" flows (as indicated by a client flag).
func (s *CharacterServer) handleCharacterUpdate(ctx context.Context, c *Client, charPkt *commands.CharacterSummary) error {
	if c.LoginPhase == commands.CharacterUpdate {
		// "Dressing room"; a request to update an existing character.
		if err := s.updateCharacter(ctx, c, charPkt); err != nil {
			Logger.Error(err.Error())
			return err
		}
	} else {
		// The "recreate" option. This is a request to create a character in a slot and is used
		// for both creating new characters and replacing existing ones.
		err := shipgate.Shipgate.DeleteCharacter(ctx, c.Account.ID, charPkt.Slot)
		if err != nil {
			msg := fmt.Errorf("error deleting character for account %d in slot %d ", c.Account.ID, charPkt.Slot)
			Logger.Error(msg)
			return msg
		}

		stats := BaseStats[charPkt.Visual.Class]
		newCharacter := &commands.CharacterData{
			DisplayData: commands.CharacterDisplayData{
				// Set the base stats using our parameter file.
				Stats: commands.CharacterStats{
					ATP:        stats.ATP,
					MST:        stats.MST,
					EVP:        stats.EVP,
					HP:         stats.HP,
					DFP:        stats.DFP,
					ATA:        stats.ATA,
					LCK:        stats.LCK,
					Level:      0,
					Experience: 0,
					Meseta:     StartingMeseta,
				},
				// Set the character's attributes directly from the preview packet.
				Visual: charPkt.Visual,
			},
			Signature:   0xC87ED5B1,
			OptionFlags: 0x00040058,
		}
		copy(newCharacter.GuildCard.Name[:], charPkt.Name[:])
		copy(newCharacter.DisplayData.DispName[:], charPkt.Name[:])

		cf := CharacterClassFlags(charPkt.Visual.ClassFlags)

		visualConfig := DefaultVisualConfigHunterRanger
		if cf.IsForce() {
			visualConfig = DefaultVisualConfigForce
		}
		copy(newCharacter.DisplayData.Config[:], visualConfig)

		// Set up the default configuration.
		copy(newCharacter.SymbolChats[:], BaseSymbolChats[:])
		copy(newCharacter.KeyConfig[:], BaseKeyConfig[:])

		// Set up the starting inventory.
		copied := copy(newCharacter.Inventory.Items[:], DefaultInventory[:])
		copied += copy(newCharacter.Inventory.Items[copied:], DefaultWeaponsByClass[int(charPkt.Visual.Class)])

		colorIdx := charPkt.Visual.Costume
		// Androids don't have costumes, so we need to use their skin instead.
		if cf.IsAndroid() {
			colorIdx = charPkt.Visual.Skin
		}

		// Set up the starting Mag.
		mag := DefaultMag
		mag.Item.Data2[3] = DefaultMagColors[charPkt.Visual.Class][colorIdx]

		copied += copy(newCharacter.Inventory.Items[copied:], []commands.CharacterInventoryItem{mag})
		newCharacter.Inventory.NumItems = uint8(copied)

		// All characters start with no techniques.
		techLevels := [20]uint8{}
		for i := range techLevels {
			// disabled
			techLevels[i] = 0xFF
		}

		// Forces, however, start with Foie.
		if cf.IsForce() {
			techLevels[0] = 0x00
		}
		copy(newCharacter.DisplayData.TechLevels[:], techLevels[:])

		// We're done, persist the new character.
		err = shipgate.Shipgate.UpsertCharacter(ctx, c.Account.ID, charPkt.Slot, newCharacter)
		if err != nil {
			return err
		}
	}

	c.Config.SlotNum = uint8(charPkt.Slot)
	return SendCharacterAck(ctx, c, charPkt.Slot, 0)
}

func (s *CharacterServer) updateCharacter(ctx context.Context, c *Client, pkt *commands.CharacterSummary) error {
	character, err := shipgate.Shipgate.FindCharacter(ctx, c.Account.ID, pkt.Slot)
	if err != nil {
		return err
	} else if character == nil {
		return fmt.Errorf("character does not exist in slot %d for account %d", pkt.Slot, c.Account.ID)
	}

	character.DisplayData.Visual = pkt.Visual

	return shipgate.Shipgate.UpsertCharacter(ctx, c.Account.ID, pkt.Slot, character)
}

// Character class flags specify attributes of different character classes.
type CharacterClassFlags uint32

func (c CharacterClassFlags) hasBit(pos int) bool {
	val := c & (1 << pos)
	return val > 0
}

func (c CharacterClassFlags) IsForce() bool {
	return c.hasBit(7)
}

func (c CharacterClassFlags) IsRanger() bool {
	return c.hasBit(6)
}

func (c CharacterClassFlags) IsHunter() bool {
	return c.hasBit(5)
}

func (c CharacterClassFlags) IsAndroid() bool {
	return c.hasBit(4)
}

func (c CharacterClassFlags) IsNewman() bool {
	return c.hasBit(3)
}

func (c CharacterClassFlags) IsHuman() bool {
	return c.hasBit(2)
}

func (c CharacterClassFlags) IsFemale() bool {
	return c.hasBit(1)
}

func (c CharacterClassFlags) IsMale() bool {
	return c.hasBit(0)
}
