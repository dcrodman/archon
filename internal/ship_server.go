package internal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"net"
	"sync"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/dcrodman/archon/internal/commands"
	"github.com/dcrodman/archon/internal/encryption"
	"github.com/dcrodman/archon/internal/shipgate"
)

type GameServer struct {
	Name string

	lobbies []*Lobby

	gamesMtx sync.Mutex
	games    []*Game
}

func (s *GameServer) Identifier() string {
	return "SHIP:GAME"
}

func (s *GameServer) Init(ctx context.Context) error {
	// Create the available lobbies.
	s.lobbies = make([]*Lobby, Config.ShipServer.NumLobbies)
	for i := range Config.ShipServer.NumLobbies {
		s.lobbies[i] = NewLobby(uint8(i))
	}
	// Lobby IDs are uint8s and since this is just an array of pointers we can set
	// the capacity and spare ourselves from having to deal with expanding it.
	s.games = make([]*Game, math.MaxUint8)
	return nil
}

func (s *GameServer) Handshake(ctx context.Context, c *Client) error {
	c.CryptoSession = encryption.NewBlueBurstCryptoSession()

	pkt := &commands.Welcome{
		Header:       commands.BBHeader{Type: commands.LoginWelcomeType, Size: 0xC8},
		Copyright:    [96]byte{},
		ServerVector: [48]byte{},
		ClientVector: [48]byte{},
	}
	copy(pkt.Copyright[:], GameCopyright)
	copy(pkt.ServerVector[:], c.CryptoSession.ServerVector())
	copy(pkt.ClientVector[:], c.CryptoSession.ClientVector())

	return c.SendRaw(ctx, pkt)
}

func (s *GameServer) Handle(ctx context.Context, c *Client, data []byte) error {
	var cmdHeader commands.BBHeader
	UnmarshalStruct(data[:commands.BBHeaderSize], &cmdHeader)

	var err error
	switch cmdHeader.Type {
	case commands.LoginType:
		var loginCmd commands.Login
		UnmarshalStruct(data, &loginCmd)
		err = s.handleLogin(ctx, c, &loginCmd)
	case commands.DisconnectType:
		// Ignore and allow the upstream call to handleDisconnectedClient to clean up the client.
	case commands.ShipListRequestType:
		var shipListCmd commands.ShipList
		UnmarshalStruct(data, &shipListCmd)
		err = s.handleShipListRequest(ctx, c, &shipListCmd)
	case commands.MenuSelectionType:
		err = s.handleMenuSelection(ctx, c, data)
	case commands.LobbySelectType:
		var lobbySelectCmd commands.LobbySelect
		UnmarshalStruct(data, &lobbySelectCmd)
		err = s.handleLobbySelection(ctx, c, lobbySelectCmd)
	case commands.GameListType:
		err = s.handleGameList(ctx, c)
	case commands.CreateGameType:
		var createCmd commands.CreateGame
		UnmarshalStruct(data, &createCmd)
		err = s.handleCreateGame(ctx, c, createCmd)
	case commands.LeaveGameType:
		var playerData commands.PlayerData
		UnmarshalStruct(data, &playerData)
		s.handleLeaveGame(ctx, c, playerData)
	case commands.BroadcastType:
		var broadcastCmd commands.Broadcast
		UnmarshalStruct(data, &broadcastCmd)
		s.handleBroadcastCommand(ctx, c, broadcastCmd)
	case commands.RoomNameType:
		err = s.handleRoomNameRequest(ctx, c)
	case commands.SyncCharacterType:
		var syncCommand commands.SyncCharacter
		UnmarshalStruct(data, &syncCommand)
		err = s.handleSyncCharacter(ctx, c, &syncCommand)
	default:
		Logger.Infof("received unknown command %x from %s", cmdHeader.Type, c.IPAddr)
	}
	return err
}

// cleanupDisconnectedClient is invoked when clients disconnect from the game server.
func (s *GameServer) cleanupDisconnectedClient(ctx context.Context, c *Client) {
	// Remove the client from the lobby they were in.
	c.State.Lock()
	room := c.State.Room
	c.State.Unlock()
	if room != nil {
		room.RemoveClient(ctx, c)
	}
}

func (s *GameServer) handleLogin(ctx context.Context, c *Client, loginPkt *commands.Login) error {
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
	c.Config.SlotNum = loginPkt.Slot

	character, err := shipgate.Shipgate.FindCharacter(ctx, c.Account.ID, uint32(loginPkt.Slot))
	if err != nil {
		return fmt.Errorf("error loading selected character: %v", err)
	}
	c.State.Lock()
	c.State.Character = character
	c.State.Unlock()

	if err := SendSecurity(ctx, c, commands.BBLoginErrorNone); err != nil {
		return err
	}
	if err := SendSyncCharacter(ctx, c); err != nil {
		return err
	}
	if err := SendLobbyMenu(ctx, c); err != nil {
		return err
	}

	// TODO: Send C5.

	// Newserv notes that the client may ignore this packet if it's sent while the client
	// is still joining the lobby (aka "bursting") so add a delay before we send this.
	time.AfterFunc(2*time.Second, func() {
		if err := SendLobbyArrowUpdate(ctx, c, c.State.Room.(*Lobby)); err != nil {
			Logger.Warnf("error sending lobby update to client %v: %v", c.IPAddr, err)
		}
	})
	return s.addClientToLobby(ctx, c, -1)
}

func SendMessage(ctx context.Context, c *Client, message string) error {
	return c.Send(ctx, &commands.ClientMessage{
		Header:   commands.BBHeader{Type: commands.ClientMessageType},
		Language: 0x00450009,
		Message:  EncodeUTF16LEString(message),
	})
}

const (
	// Values stolen from Tethealla, though it almost certainly doesn't matter.
	NameColorNormal = 0xFFFFFFFF
	NameColorGM     = 0xFF1D94F7
)

func SendSyncCharacter(ctx context.Context, c *Client) error {
	cmd := &commands.SyncCharacter{
		Header:    commands.BBHeader{Type: commands.SyncCharacterType},
		Character: *c.State.Character,
	}

	cmd.Character.DisplayData.Visual.NameColor = NameColorNormal
	if c.Account.GM {
		cmd.Character.DisplayData.Visual.NameColor = NameColorGM
	}

	return c.Send(ctx, cmd)
}

func SendLobbyMenu(ctx context.Context, c *Client) error {
	lobbyEntries := make([]commands.LobbyListEntry, Config.ShipServer.NumLobbies)
	for i := 0; i < Config.ShipServer.NumLobbies; i++ {
		lobbyEntries[i].MenuID = 0x001A0001
		lobbyEntries[i].LobbyID = uint32(i)
	}

	return c.Send(ctx, &commands.LobbyMenu{
		Header: commands.BBHeader{
			Type:  commands.LobbyMenuType,
			Flags: 0x0F, // PSOBB expects this to always contain 15 entries.
		},
		Lobbies: lobbyEntries,
	})
}

func SendLobbyArrowUpdate(ctx context.Context, c *Client, l *Lobby) error {
	entries := l.BuildLobbyArrowEntries()
	return c.Send(ctx, &commands.LobbyArrowUpdate{
		Header: commands.BBHeader{
			Type:  commands.LobbyArrowUpdateType,
			Flags: uint32(len(entries)),
		},
		Entries: entries,
	})
}

// Find an available lobby and add the client to it, sending the appropriate notifications to
// both the joining client and all existing clients in the lobby.
func (s *GameServer) addClientToLobby(ctx context.Context, c *Client, preferredLobbyID int) error {
	// Try to add the player to the lobby the client selected. If we're unable to do so (for
	// instance because the lobby has filled), fall through to the usual lobby selection logic.
	if preferredLobbyID >= 0 {
		lobby := s.lobbies[preferredLobbyID]
		if err := lobby.AddClient(ctx, c); err == nil {
			return nil
		}
	}

	for _, lobby := range s.lobbies {
		if lobby.IsFull() {
			continue
		}

		err := lobby.AddClient(ctx, c)
		switch {
		case err != nil:
			return err
		case err == ErrLobbyFull:
			continue
		default:
			return nil
		}
	}
	return nil
}

func (s *GameServer) handleShipListRequest(ctx context.Context, c *Client, _ *commands.ShipList) error {
	return SendShipList(ctx, c)
}

func (s *GameServer) handleMenuSelection(ctx context.Context, c *Client, data []byte) error {
	var cmd commands.MenuSelection
	UnmarshalStruct(data, &cmd)

	switch cmd.MenuID {
	case ShipListMenuID:
		return s.handleShipSelection(ctx, c, cmd)
	case GameListMenuID:
		return s.handleGameSelection(ctx, c, data, cmd)
	default:
		return fmt.Errorf("unrecognized menu ID: %v", cmd.MenuID)
	}
}

// Player selected one of the items on the ship select screen; respond with the
// IP address and port of the ship server to  which the client will connect after
// disconnecting from this server.
func (s *GameServer) handleShipSelection(ctx context.Context, c *Client, cmd commands.MenuSelection) error {
	availableShips := shipgate.Shipgate.GetAvailableShips(ctx)

	selectedShip := cmd.ItemID - 1
	if selectedShip >= uint32(len(availableShips)) {
		return fmt.Errorf("invalid ship selection: %d", selectedShip)
	}

	var ip [4]uint8
	copy(ip[:], net.ParseIP(availableShips[selectedShip].Address).To4())
	port := uint16(availableShips[selectedShip].Port)

	return SendRedirect(ctx, c, ip, port)
}

// Player has selected a game from the kiosk (after receiving the game list).
func (s *GameServer) handleGameSelection(ctx context.Context, c *Client, data []byte, cmd commands.MenuSelection) error {
	s.gamesMtx.Lock()
	game := s.games[cmd.ItemID-1]
	s.gamesMtx.Unlock()

	if game == nil {
		return SendLobbyMessageBox(ctx, c, "$C7This game no longer exists.")
	}

	// If the game was password-protected, the game will have prompted the player for one and included
	// the input in this command. This check is valid even if the password is empty.
	if game.HasPassword() {
		if cmd.Header.Size < 0x30 || !bytes.Equal(game.Password[:], data[16:cmd.Header.Size]) {
			return SendLobbyMessageBox(ctx, c, "$C7Incorrect password.")
		}
	}

	// TODO: Like when creating a game, check the player's level (or do it in game) relative to difficulty.

	// Transfer them to the game they requested to join.
	c.State.Lock()
	currentRoom := c.State.Room
	c.State.Unlock()
	if currentRoom != nil {
		currentRoom.RemoveClient(ctx, c)
	}

	err := game.AddClient(ctx, c)
	switch err {
	case nil:
		return nil
	case ErrGameAbandoned:
		return SendLobbyMessageBox(ctx, c, "$C7This game no longer exists.")
	case ErrLobbyFull:
		return SendLobbyMessageBox(ctx, c, "$C7This game is full.")
	default:
		Logger.Warnw("client %v encountered error joining game %s: %v", c.IPAddr, DecodeUTF16LE(game.Name[:]), err)
		return SendLobbyMessageBox(ctx, c, "$C7You cannot join this game.")
	}
}

// Player has selected a new lobby from the teleporter or just left a game to return to the lobby.
func (s *GameServer) handleLobbySelection(ctx context.Context, c *Client, cmd commands.LobbySelect) error {
	return s.addClientToLobby(ctx, c, int(cmd.LobbyID))
}

const GameListMenuID = 0x22222222

func (s *GameServer) handleGameList(ctx context.Context, c *Client) error {
	cmd := commands.Menu{Entries: make([]commands.MenuEntry, 1)}
	// Like for the ship menu, the client expects a placeholder first entry.
	cmd.Entries[0].MenuID = GameListMenuID
	copy(cmd.Entries[0].Name[:], EncodeUTF16LEString("archon")[:])

	s.gamesMtx.Lock()
	var games uint32
	for i, game := range s.games {
		if game == nil {
			continue
		}
		entry := commands.MenuEntry{
			MenuID: GameListMenuID,
			ItemID: uint32(i) + 1,
			// Shamelessly ripping off newserv here but the difficulty needs 0x22 added for some reason.
			Difficulty: uint8(game.Difficulty) + 0x22,
			NumPlayers: uint8(game.NumPlayers()),
			Episode:    uint8(game.Episode),
		}
		copy(entry.Name[:], game.Name[:])

		if game.HasPassword() {
			entry.Flags |= 0x02
		}
		if game.IsFull() {
			entry.Flags |= 0x04
		}
		switch game.Mode {
		case BattleMode:
			entry.Flags |= 0x10
		case ChallengeMode:
			entry.Flags |= 0x20
		}
		cmd.Entries = append(cmd.Entries, entry)
		games++
	}
	s.gamesMtx.Unlock()

	cmd.Header = commands.BBHeader{
		Type:  commands.GameListType,
		Flags: games,
	}
	return c.Send(ctx, cmd)
}

// handleCreateGame sets up a new Game in response to a player creating one from the lobby kiosk.
func (s *GameServer) handleCreateGame(ctx context.Context, c *Client, cmd commands.CreateGame) error {
	// TODO: Check that the current character's level qualifies for the difficulty mode.

	game := &Game{
		Episode:    GameEpisode(cmd.Episode),
		Difficulty: GameDifficulty(cmd.Difficulty),
		RandomSeed: rand.Uint32(),
	}
	copy(game.Name[:], cmd.Name[:])
	copy(game.Password[:], cmd.Password[:])

	switch {
	case cmd.BattleMode == 1:
		game.Mode = BattleMode
	case cmd.ChallengeMode == 1:
		game.Mode = ChallengeMode
	case cmd.SoloMode == 1:
		game.Mode = SoloMode
	}

	var assigned bool
	s.gamesMtx.Lock()
	// Put the game in the first available slot.
	for i := range s.games {
		if s.games[i] == nil {
			s.games[i] = game
			game.ID = uint8(i)
			assigned = true
			break
		}
	}
	s.gamesMtx.Unlock()
	if !assigned {
		// TODO: Send the player a notification rather than erroring.
		return errors.New("block is full")
	}

	// Move the player out of their current lobby and into the game they just created.
	c.State.Lock()
	lobby := c.State.Room
	c.State.Unlock()
	lobby.RemoveClient(ctx, c)
	if err := game.AddClient(ctx, c); err != nil {
		return fmt.Errorf("creating game and joining: %v", err)
	}

	// TODO: Send 1D

	return nil
}

func (s *GameServer) handleLeaveGame(ctx context.Context, c *Client, cmd commands.PlayerData) {
	c.State.Lock()
	game := c.State.Room
	c.State.Unlock()

	// Disconnect the player from the current lobby, but don't add them to another one -
	// that will be handled by the 84 command.
	if game != nil {
		game.RemoveClient(ctx, c)
	}

	c.State.Lock()
	defer c.State.Unlock()

	// Update the client's Character with the contents of this command, with the exception
	// of display data and inventory since that is maintained server-side.
	copy(c.State.Character.ChallengeRecords[:], cmd.Records.ChallengeRecords[:])
	copy(c.State.Character.BattleRecords[:], cmd.Records.BattleRecords[:])
	copy(c.State.Character.InfoBoard[:], cmd.InfoBoard[:])
	copy(c.State.Character.ChoiceSearch[:], cmd.ChoiceSearch[:])

	// TODO: Skipping auto reply for now since I don't yet know what to do with that.
	// TODO: Blocked guildcards, which may require revisiting the guildcard data model (see character server).

	// Persist the character data we just updated, which was expected in the original game.
	if err := shipgate.Shipgate.UpsertCharacter(ctx, c.Account.ID, uint32(c.Config.SlotNum), c.State.Character); err != nil {
		Logger.Warnf("error saving character data for client %v: %v", c.IPAddr, err)
	}
}

func SendLobbyMessageBox(ctx context.Context, c *Client, msg string) error {
	cmd := commands.LobbyMessageBox{
		Header: commands.BBHeader{
			Type: commands.LobbyMessageBoxType,
		},
		Message: EncodeUTF16LEString(msg),
	}
	if len(cmd.Message) > 0x200 {
		return errors.New("message must not exceed 0x200 bytes")
	}
	return c.Send(ctx, cmd)
}

func (s *GameServer) handleBroadcastCommand(ctx context.Context, c *Client, cmd commands.Broadcast) {
	c.State.Lock()
	room := c.State.Room
	c.State.Unlock()
	if room != nil {
		room.Broadcast(ctx, c, cmd)
	}
}

func (s *GameServer) handleRoomNameRequest(ctx context.Context, c *Client) error {
	c.State.Lock()
	room := c.State.Room
	c.State.Unlock()
	// This shouldn't ever really happen but the command appears to be safe to ignore.
	if room == nil {
		return nil
	}

	cmd := struct {
		Header  commands.BBHeader
		Message []uint8
	}{
		Header: commands.BBHeader{
			Type: commands.RoomNameType,
		},
		Message: room.RoomName(),
	}
	return c.Send(ctx, cmd)
}

// Client has requested to save the current game state, so flush it to the database.
func (s *GameServer) handleSyncCharacter(ctx context.Context, c *Client, _ *commands.SyncCharacter) error {
	c.State.Lock()
	charData := *c.State.Character
	c.State.Unlock()

	// Based on my youth of hacking the s*** out of this game, I'm opting to ignore the contents of the
	// sync command from the client in favor of flushing the server-side state.

	return shipgate.Shipgate.UpsertCharacter(ctx, c.Account.ID, uint32(c.Config.SlotNum), &charData)
}
