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

	// Lock must be held when accessing any state variables.
	sync.RWMutex
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

// RoomName returns the Lobby's name. Mainly for consistency with the Room interface.
func (l *Lobby) RoomName() []byte {
	lobbyName := fmt.Sprintf("LOBBY %02d", l.ID)
	return EncodeUTF16LEString(lobbyName)
}

func (l *Lobby) IsFull() bool {
	l.RLock()
	defer l.RUnlock()
	return l.currentClients >= MaxLobbyPlayers
}

// AddClient adds a client to a lobby and sends the appropriate lobby join notifications
// to both the player being added and all other players in the lobby. The client's
// LobbyID (i.e. its index/slot in the lobby) is set if they were successfully added,
// otherwise an error is returned if the lobby is full or the client could not be added.
func (l *Lobby) AddClient(ctx context.Context, c *Client) error {
	l.Lock()
	if l.currentClients >= MaxLobbyPlayers {
		l.Unlock()
		return ErrLobbyFull
	}

	// Now assign the client to the first available slot in the lobby and update its state.
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
	l.Unlock()

	c.State.Lock()
	c.State.Room = l
	c.State.LobbySlotID = lobbySlotID
	c.State.Unlock()

	// Inform the client of its new lobby assignment.
	if err := SendJoinLobby(ctx, l, c, lobbySlotID); err != nil {
		return fmt.Errorf("assigning player to lobby: %v", err)
	}

	// Notify the existing clients in the lobby that a player joined.
	SendLobbyJoinNotification(ctx, l, c)

	return nil
}

// SendJoinLobby sends the current state of the lobby to the joining player.
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

	l.Lock()
	joinCmd.LeaderID = l.leaderID
	otherClients := make([]*Client, len(l.clients))
	copy(otherClients, l.clients)
	l.Unlock()

	// Build the full set of entries for all other clients in the lobby (excluding the joining player).
	for _, oc := range otherClients {
		if oc == nil {
			continue
		}
		joinCmd.Entries = append(joinCmd.Entries, buildPlayerLobbyEntry(c))
	}

	joinCmd.Header.Flags = uint32(len(joinCmd.Entries))
	return c.Send(ctx, joinCmd)
}

// SendLobbyJoinNotification is used to send notifications to all other players
// when a client joins a lobby.
func SendLobbyJoinNotification(ctx context.Context, l *Lobby, joiningClient *Client) {
	l.Lock()
	lobbyLeaderID := l.leaderID
	clients := make([]*Client, len(l.clients))
	copy(clients, l.clients)
	l.Unlock()

	baseCmd := commands.JoinLobby{
		Header: commands.BBHeader{
			Type:  commands.AddPlayerToLobby,
			Flags: 1, // 1 indicates that this only contains one entry (the joining client).
		},
		DisableUDP:       1,
		LobbyNumber:      l.ID,
		BlockNumber:      0, // Always going to be 0 unless we support more blocks.
		EnableBattleMode: 1,
		LeaderID:         lobbyLeaderID,
		// Event:            0,
		EnableVoiceChat: 1,
		Entries: []commands.PlayerLobbyEntry{
			buildPlayerLobbyEntry(joiningClient),
		},
	}

	for _, oc := range clients {
		if oc == joiningClient || oc == nil {
			continue
		}

		joinCmd := baseCmd
		oc.State.Lock()
		joinCmd.ClientID = oc.State.LobbySlotID
		oc.State.Unlock()

		if err := oc.Send(ctx, joinCmd); err != nil {
			Logger.Warnf("error sending lobby join notification to client %v: %v", oc.IPAddr, err)
		}
	}
}

// RemoveClient removes a client from a lobby, resetting the Client's LobbySlotID. If the player
// being removed was the current leader, a new leader is selected from the remaining set of players.
func (l *Lobby) RemoveClient(ctx context.Context, c *Client) {
	c.State.Lock()
	currentLobbySlotID := c.State.LobbySlotID
	// This is probably redundant and is technically still a valid slot.
	c.State.LobbySlotID = 0
	c.State.Room = nil
	c.State.Unlock()

	l.Lock()
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
	l.Unlock()

	// Notify the other players that a player has left the lobby.
	SendLeaveLobbyNotifications(ctx, l, c, currentLobbySlotID)
}

func SendLeaveLobbyNotifications(ctx context.Context, l *Lobby, c *Client, departingSlotID uint8) {
	l.Lock()
	lobbyLeaderID := l.leaderID
	clients := make([]*Client, len(l.clients))
	copy(clients, l.clients)
	l.Unlock()

	for _, oc := range clients {
		if oc == c || oc == nil {
			continue
		}
		oc.State.Lock()
		cmd := &commands.LeaveLobby{
			Header: commands.BBHeader{
				Type:  commands.RemovePlayerFromLobbyType,
				Flags: uint32(departingSlotID),
			},
			ClientID:   oc.State.LobbySlotID,
			LeaderID:   lobbyLeaderID,
			DisableUDP: 1,
		}
		oc.State.Unlock()

		if err := oc.Send(ctx, cmd); err != nil {
			Logger.Warnf("error sending lobby leave notification to client %v: %v", oc.IPAddr, err)
		}
	}
}

// Broadcast sends cmd to all players in the lobby except for sender.
func (l *Lobby) Broadcast(ctx context.Context, sender *Client, cmd commands.Broadcast) {
	l.Lock()
	clients := make([]*Client, len(l.clients))
	copy(clients, l.clients)
	l.Unlock()

	for _, c := range clients {
		if c == nil || c == sender {
			continue
		}
		if err := c.Send(ctx, cmd); err != nil {
			Logger.Warnf("error sending broadcast command to client %v: %v", c.IPAddr, err)
		}
	}
}

// BuildLobbyArrowEntries calculates the arrow colors for all players currently in the lobby
// and returns the entries for the LobbyArrowUpdate command.
func (l *Lobby) BuildLobbyArrowEntries() []commands.LobbyArrowUpdateEntry {
	l.Lock()
	defer l.Unlock()

	entries := make([]commands.LobbyArrowUpdateEntry, l.currentClients)
	i := 0
	for _, c := range l.clients {
		if c == nil {
			continue
		}
		var arrowColor uint32
		// GMs get orange arrows by default.
		if c.Account.GM {
			arrowColor = 7
		}
		c.State.Lock()
		entries[i] = commands.LobbyArrowUpdateEntry{
			PlayerTag:  0x00010000,
			Guildcard:  c.State.Character.GuildCard.GuildCardNumber,
			ArrowColor: arrowColor,
		}
		c.State.Unlock()
		i++
	}
	return entries
}
