package internal

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/dcrodman/archon/internal/commands"
)

var ErrGameAbandoned = errors.New("all players have left the game")

const MaxPlayersPerGame = 4

type GameEpisode uint8

const (
	Episode1 GameEpisode = 1
	Episode2 GameEpisode = 2
	Episode4 GameEpisode = 4
)

type GameMode uint8

const (
	NormalMode GameMode = iota
	BattleMode
	ChallengeMode
	SoloMode
)

type GameDifficulty uint8

const (
	NormalDifficulty = iota
	Hard
	VeryHard
	Ultimate
)

type Game struct {
	ID uint8

	Name       [32]uint8
	Password   [32]uint8
	Episode    GameEpisode
	Mode       GameMode
	Difficulty GameDifficulty
	RandomSeed uint32

	// Lock must be held when accessing any of the remaining fields.
	sync.Mutex
	clients        [MaxPlayersPerGame]*Client
	currentClients int

	leaderID  uint8
	sectionID uint8

	inQuest   bool
	abandoned bool
}

// RoomName returns the Game's name. Name may be accessed directly above, and this is mainly
// for consistency with the Room interface.
func (g *Game) RoomName() []byte {
	return g.Name[:]
}

func (g *Game) NumPlayers() int {
	g.Lock()
	defer g.Unlock()
	return g.currentClients
}

func (g *Game) IsFull() bool {
	return g.NumPlayers() >= MaxPlayersPerGame
}

func (g *Game) HasPassword() bool {
	return g.Password[0] != 0
}

// AddClient adds a client to a lobby and sends the appropriate lobby join notifications
// to both the player being added and all other players in the lobby. The client's
// LobbyID (i.e. its index/slot in the lobby) is set if they were successfully added,
// otherwise an error is returned if the lobby is full or the client could not be added.
func (g *Game) AddClient(ctx context.Context, c *Client) error {
	g.Lock()
	switch {
	case g.currentClients >= MaxPlayersPerGame:
		g.Unlock()
		return ErrLobbyFull
	case g.abandoned:
		g.Unlock()
		return ErrGameAbandoned
	}

	// Now assign the client to the first available slot in the lobby and update its state.
	var lobbySlotID uint8
	for i := range g.clients {
		if g.clients[i] != nil {
			continue
		}
		lobbySlotID = uint8(i)

		g.clients[i] = c
		g.currentClients++
		break
	}
	g.Unlock()

	c.State.Lock()
	c.State.Room = g
	c.State.LobbySlotID = lobbySlotID
	c.State.Unlock()

	// Inform the client of its new lobby assignment.
	if err := SendJoinGame(ctx, g, c, lobbySlotID); err != nil {
		return fmt.Errorf("assigning player to lobby: %v", err)
	}

	// Notify the existing clients in the lobby that a player joined.
	SendJoinGameNotifications(ctx, g, c)

	return nil
}

// SendJoinGame sends a joining client the current state of all other players in the game.
func SendJoinGame(ctx context.Context, g *Game, c *Client, lobbySlotID uint8) error {
	joinCmd := &commands.JoinGame{
		Header: commands.BBHeader{
			Type: commands.JoinGameType,
		},
		// Variations: ,
		ClientID:   lobbySlotID,
		DisableUDP: 1,
		Difficulty: uint8(g.Difficulty),
		RandomSeed: g.RandomSeed,
		Episode:    uint8(g.Episode),
	}
	switch g.Mode {
	case BattleMode:
		joinCmd.BattleMode = 1
	case ChallengeMode:
		joinCmd.ChallengeMode = 1
	case SoloMode:
		joinCmd.SoloMode = 1
	}

	g.Lock()
	joinCmd.LeaderID = g.leaderID
	if g.inQuest {
		joinCmd.InQuest = 1
	}

	// Build the full set of entries for all other clients in the lobby (excluding the joining player).
	for _, oc := range g.clients {
		if oc == nil {
			continue
		}
		oc.Lock()
		playerEntry := commands.PlayerLobbyEntry{
			PlayerLobbyData: commands.PlayerLobbyData{
				PlayerTag: 0x00010000,
				Guildcard: uint32(oc.Account.Guildcard),
				// TODO: Will need to set this once teams are supported.
				// TMGuildcard: ,
				TeamID:         uint32(oc.Account.TeamID),
				ClientID:       uint32(oc.LobbySlotID),
				HideHelpPrompt: 1,
			},
			Inventory:   oc.Character.Inventory,
			DisplayData: oc.Character.DisplayData,
		}
		copy(playerEntry.Name[:], oc.Character.GuildCard.Name[:])
		oc.Unlock()
	}
	joinCmd.Header.Flags = uint32(g.currentClients)
	g.Unlock()

	return c.Send(ctx, joinCmd)
}

func buildPlayerLobbyEntry(c *Client) commands.PlayerLobbyEntry {
	c.State.Lock()
	defer c.State.Unlock()

	entry := commands.PlayerLobbyEntry{
		PlayerLobbyData: commands.PlayerLobbyData{
			PlayerTag: 0x00010000,
			Guildcard: uint32(c.Account.Guildcard),
			// TODO: Will need to set this once teams are supported.
			// TMGuildcard: ,
			TeamID:         uint32(c.Account.TeamID),
			ClientID:       uint32(c.State.LobbySlotID),
			HideHelpPrompt: 1,
		},
		Inventory:   c.State.Character.Inventory,
		DisplayData: c.State.Character.DisplayData,
	}
	copy(entry.Name[:], c.State.Character.GuildCard.Name[:])
	return entry
}

// SendJoinGameNotifications is used to inform all other players in the game that a new
// player has joined.
func SendJoinGameNotifications(ctx context.Context, g *Game, c *Client) {
	c.State.Lock()
	lobbySlotID := c.State.LobbySlotID
	c.State.Unlock()

	joinCmd := &commands.JoinLobby{
		Header: commands.BBHeader{
			Type:  commands.AddPlayerToGame,
			Flags: 1, // 1 indicates that this only contains one entry (the joining client).
		},
		ClientID:         lobbySlotID,
		DisableUDP:       1,
		LobbyNumber:      g.ID,
		BlockNumber:      0, // Always going to be 0 unless we support more blocks.
		EnableBattleMode: 1,
		// Event:            0,
		EnableVoiceChat: 1,
	}

	// Entries works differently than it does for lobbies in that the player data is
	// filled in according to their slot positions.
	joinCmd.Entries = make([]commands.PlayerLobbyEntry, MaxPlayersPerGame)
	joinCmd.Entries[lobbySlotID] = commands.PlayerLobbyEntry{
		PlayerLobbyData: commands.PlayerLobbyData{
			PlayerTag: 0x00010000,
			Guildcard: uint32(c.Account.Guildcard),
			// TODO: Will need to set this once teams are supported.
			// TMGuildcard: ,
			TeamID:         uint32(c.Account.TeamID),
			ClientID:       uint32(c.LobbySlotID),
			HideHelpPrompt: 1,
		},
		Inventory:   c.Character.Inventory,
		DisplayData: c.Character.DisplayData,
	}
	copy(joinCmd.Entries[lobbySlotID].Name[:], c.Character.GuildCard.Name[:])

	g.Lock()
	joinCmd.LeaderID = g.leaderID
	otherClients := make([]*Client, len(g.clients))
	copy(otherClients[:], g.clients[:])
	g.Unlock()

	for _, oc := range otherClients {
		if oc == nil || oc == c {
			continue
		}
		if err := oc.Send(ctx, joinCmd); err != nil {
			Logger.Warnf("error sending game join notification to client %v: %v", oc.IPAddr, err)
		}
	}
}

// RemoveClient removes a client from a lobby, resetting the Client's LobbySlotID. If the player
// being removed was the current leader, a new leader is selected from the remaining set of players.
func (g *Game) RemoveClient(ctx context.Context, c *Client) {
	c.State.Lock()
	currentLobbySlotID := c.State.LobbySlotID
	// This is probably redundant and is technically still a valid slot.
	c.State.LobbySlotID = 0
	c.State.Room = nil
	c.State.Unlock()

	g.Lock()
	g.clients[currentLobbySlotID] = nil
	g.currentClients--

	if g.currentClients == 0 {
		// All players have left the game; prevent any new players from joining and notify the
		// ship server that it can be cleaned up.
		g.abandoned = true
		g.Unlock()
		return
	}

	// Select a new leader starting from the lowest index
	if g.leaderID == currentLobbySlotID {
		g.leaderID = 0
		for i := range g.clients {
			if g.clients[i] != nil {
				g.leaderID = uint8(i)

				g.clients[i].State.Lock()
				g.sectionID = g.clients[i].State.Character.DisplayData.Visual.SectionID
				g.clients[i].State.Unlock()

				break
			}
		}
	}

	// TODO: This is where we'll need to trigger updating the drop tables to match the new leader.
	// Newserv supports disabling this behavior and only ever using the drop tables (and thus
	// section ID) of the leader that created the game, but for now we just do it.

	g.Unlock()

	// Notify the other players that a player has left the lobby.
	SendLeaveGameNotifications(ctx, g, c, currentLobbySlotID)
}

func SendLeaveGameNotifications(ctx context.Context, g *Game, c *Client, departingSlotID uint8) {
	g.Lock()
	lobbyLeaderID := g.leaderID
	clients := make([]*Client, len(g.clients))
	copy(clients[:], g.clients[:])
	g.Unlock()

	for _, oc := range clients {
		if oc == c || oc == nil {
			continue
		}
		oc.State.Lock()
		lobbySlotID := oc.State.LobbySlotID
		oc.State.Unlock()

		if err := oc.Send(ctx, &commands.LeaveLobby{
			Header: commands.BBHeader{
				Type:  commands.RemovePlayerFromGame,
				Flags: uint32(departingSlotID),
			},
			ClientID:   lobbySlotID,
			LeaderID:   lobbyLeaderID,
			DisableUDP: 1,
		}); err != nil {
			Logger.Warnf("error sending lobby leave notification to client %v: %v", oc.IPAddr, err)
		}
	}
}

// Broadcast sends cmd to all players in the lobby except for sender.
func (g *Game) Broadcast(ctx context.Context, sender *Client, cmd commands.Broadcast) {
	g.Lock()
	clients := make([]*Client, len(g.clients))
	copy(clients[:], g.clients[:])
	g.Unlock()

	for _, c := range clients {
		if c == nil || c == sender {
			continue
		}
		if err := c.Send(ctx, cmd); err != nil {
			Logger.Warnf("error sending broadcast command to client %v: %v", c.IPAddr, err)
		}
	}
}
