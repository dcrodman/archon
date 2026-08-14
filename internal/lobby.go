package internal

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/dcrodman/archon/internal/commands"
)

const MaxLobbyPlayers = 12

var ErrLobbyFull = errors.New("lobby contains the maximum number of players")

type Lobby struct {
	ID uint8
	// mtx must be held when accessing any state variables.
	mtx            sync.RWMutex
	clients        []*Client
	currentClients int
	leaderID       uint8
}

func NewLobby(id uint8) *Lobby {
	return &Lobby{
		ID:      id,
		clients: make([]*Client, MaxLobbyPlayers),
	}
}

func (l *Lobby) IsFull() bool {
	l.mtx.RLock()
	defer l.mtx.RUnlock()
	return l.currentClients >= MaxLobbyPlayers
}

// AddClient adds a client to a lobby and sends the appropriate lobby join notifications
// to both the player being added and all other players in the lobby. The client's
// LobbyID (i.e. its index/slot in the lobby) is set if they were successfully added,
// otherwise an error is returned if the lobby is full or the client could not be added.
func (l *Lobby) AddClient(ctx context.Context, c *Client) error {
	l.mtx.Lock()

	if l.currentClients >= MaxLobbyPlayers {
		l.mtx.Unlock()
		return ErrLobbyFull
	}

	// Now assign the client to a slot in the lobby and update its state.
	var lobbySlotID uint8
	for i := range l.clients {
		if l.clients[i] != nil {
			continue
		}
		lobbySlotID = uint8(i)

		l.clients[i] = c
		l.currentClients++
		break
	}
	l.mtx.Unlock()

	c.Lock()
	c.LobbySlotID = lobbySlotID
	c.Unlock()

	// Inform the client of its new lobby assignment.
	if err := SendJoinLobby(ctx, l, c, lobbySlotID); err != nil {
		return fmt.Errorf("assigning player to lobby: %v", err)
	}

	// Notify the existing clients in the lobby that a player joined.
	SendLobbyJoinNotification(ctx, l, c)

	return nil
}

// RemoveClient removes a client from a lobby, resetting the Client's LobbySlotID. If the player
// being removed was the current leader, a new leader is selected from the remaining set of players.
func (l *Lobby) RemoveClient(ctx context.Context, c *Client) {
	c.Lock()
	currentLobbySlotID := c.LobbySlotID
	// This is probably redundant and is technically still a valid slot.
	c.LobbySlotID = 0
	c.Unlock()

	l.mtx.Lock()
	l.clients[currentLobbySlotID] = nil
	l.currentClients--

	// Select a new leader starting from the lowest index
	if l.leaderID == currentLobbySlotID {
		l.leaderID = 0
		for i := range l.clients {
			if l.clients[i] != nil {
				l.leaderID = uint8(i)
				break
			}
		}
	}
	l.mtx.Unlock()

	// TODO: Send the appropriate packets.
}

func SendJoinLobby(ctx context.Context, l *Lobby, c *Client, lobbySlotID uint8) error {
	joinCmd := &commands.JoinLobby{
		Header: commands.BBHeader{
			Type: commands.JoinLobbyType,
		},
		ClientID:         lobbySlotID,
		DisableUDP:       1,
		LobbyNumber:      l.ID,
		BlockNumber:      0, // Always going to be 0 unless we support more blocks.
		EnableBattleMode: 1,
		// Event:            0,
		EnableVoiceChat: 1,
	}

	otherClients := make([]*Client, len(l.clients))
	l.mtx.Lock()
	joinCmd.LeaderID = l.leaderID
	copy(otherClients, l.clients)
	l.mtx.Unlock()

	// Build the full set of entries for all other clients in the lobby (excluding the joining player).
	for _, oc := range otherClients {
		if oc == nil {
			continue
		}
		oc.Lock()
		playerEntry := commands.LobbyEntry{
			PlayerTag: 0x00010000,
			Guildcard: uint32(oc.Account.Guildcard),
			// TODO: Will need to set this once teams are supported.
			// TMGuildcard: ,
			TeamID:         uint32(oc.Account.TeamID),
			ClientID:       uint32(oc.LobbySlotID),
			HideHelpPrompt: 1,
			Inventory:      oc.Character.Inventory,
			DisplayData:    oc.Character.DisplayData,
		}
		copy(playerEntry.Name[:], oc.Character.GuildCard.Name[:])
		oc.Unlock()

		joinCmd.Entries = append(joinCmd.Entries, playerEntry)
	}

	joinCmd.Header.Flags = uint32(len(joinCmd.Entries))
	return c.Send(ctx, joinCmd)
}

func SendLobbyJoinNotification(ctx context.Context, l *Lobby, joiningClient *Client) {
	clients := make([]*Client, len(l.clients))
	l.mtx.Lock()
	lobbyLeaderID := l.leaderID
	copy(clients, l.clients)
	l.mtx.Unlock()

	baseCmd := commands.JoinLobby{
		Header: commands.BBHeader{
			Type:  commands.AddPlayerToLobby,
			Flags: 1,
		},
		DisableUDP:       1,
		LobbyNumber:      l.ID,
		BlockNumber:      0, // Always going to be 0 unless we support more blocks.
		EnableBattleMode: 1,
		LeaderID:         lobbyLeaderID,
		// Event:            0,
		EnableVoiceChat: 1,
		Entries: []commands.LobbyEntry{
			{
				PlayerTag: 0x00010000,
				Guildcard: uint32(joiningClient.Account.Guildcard),
				// TODO: Will need to set this once teams are supported.
				// TMGuildcard: ,
				TeamID:         uint32(joiningClient.Account.TeamID),
				ClientID:       uint32(joiningClient.LobbySlotID),
				HideHelpPrompt: 1,
				Inventory:      joiningClient.Character.Inventory,
				DisplayData:    joiningClient.Character.DisplayData,
			},
		},
	}
	for _, oc := range clients {
		if oc == joiningClient || oc == nil {
			continue
		}

		joinCmd := baseCmd
		joinCmd.ClientID = oc.LobbySlotID

		if err := oc.Send(ctx, joinCmd); err != nil {
			Logger.Warnf("error sending lobby join notification to client %v", oc.IPAddr)
		}
	}
}
