package internal

import (
	"context"
	"fmt"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/dcrodman/archon/internal/commands"
	"github.com/dcrodman/archon/internal/encryption"
	"github.com/dcrodman/archon/internal/shipgate"
)

type GameServer struct {
	Name string

	lobbies []*Lobby
}

func (s *GameServer) Identifier() string {
	return "SHIP:GAME"
}

func (s *GameServer) Init(ctx context.Context) error {
	// Create the game lobbies.
	s.lobbies = make([]*Lobby, Config.ShipServer.NumLobbies)
	for i := range Config.ShipServer.NumLobbies {
		s.lobbies[i] = NewLobby(uint8(i))
	}
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
		var loginPkt commands.Login
		UnmarshalStruct(data, &loginPkt)
		err = s.handleLogin(ctx, c, &loginPkt)
	case commands.SyncCharacterType:
		var syncCommand commands.SyncCharacter
		UnmarshalStruct(data, &syncCommand)
		s.handleSyncCharacter(ctx, c, &syncCommand)
	case commands.DisconnectType:
		// Ignore and allow the upstream call to handleDisconnectedClient to clean up the client.
	default:
		Logger.Infof("received unknown command %x from %s", cmdHeader.Type, c.IPAddr)
	}
	return err
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

	character, err := shipgate.Shipgate.FindCharacter(ctx, c.Account.ID, loginPkt.Slot)
	if err != nil {
		return fmt.Errorf("error loading selected character: %v", err)
	}
	c.Lock()
	c.Character = character
	c.Unlock()

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
	return s.addClientToLobby(ctx, c)
}

func SendMessage(ctx context.Context, c *Client, message string) error {
	return c.Send(ctx, &commands.ClientMessage{
		Header:   commands.BBHeader{Type: commands.ClientMessageType},
		Language: 0x00450009,
		Message:  ConvertToUtf16(message),
	})
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

const (
	// Values stolen from Tethealla, though it almost certainly doesn't matter.
	NameColorNormal = 0xFFFFFFFF
	NameColorGM     = 0xFF1D94F7
)

func SendSyncCharacter(ctx context.Context, c *Client) error {
	cmd := &commands.SyncCharacter{
		Header:    commands.BBHeader{Type: commands.SyncCharacterType},
		Character: *c.Character,
	}

	cmd.Character.DisplayData.Visual.NameColor = NameColorNormal
	if c.Account.GM {
		cmd.Character.DisplayData.Visual.NameColor = NameColorGM
	}

	return c.Send(ctx, cmd)
}

// Find an available lobby and add the client to it, sending the appropriate notifications to
// both the joining client and all existing clients in the lobby.
func (s *GameServer) addClientToLobby(ctx context.Context, c *Client) error {
	for _, lobby := range s.lobbies {
		if lobby.IsFull() {
			continue
		}

		err := lobby.AddClient(ctx, c)
		switch err {
		case nil:
			return nil
		case ErrLobbyFull:
			continue
		default:
			return err
		}
	}
	return nil
}

// Client has requested to save the current game state, so flush it to the database.
func (s *GameServer) handleSyncCharacter(ctx context.Context, c *Client, syncCommand *commands.SyncCharacter) error {
	c.Lock()
	charData := *c.Character
	c.Unlock()

	// Based on my youth of hacking the s*** out of this game, I'm opting to ignore the contents of the
	// sync command from the client in favor of flushing the server-side state.

	return shipgate.Shipgate.UpsertCharacter(ctx, c.Account.ID, uint32(c.Config.SlotNum), &charData)
}

// handleDisconnectedClient is invoked when clients disconnect from the game server.
func (s *GameServer) handleDisconnectedClient(ctx context.Context, c *Client) {
	// Remove the client from the lobby they were in.
	lobby := s.lobbies[c.LobbyID]
	lobby.RemoveClient(ctx, c)
}
