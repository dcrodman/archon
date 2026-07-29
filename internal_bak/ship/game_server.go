package ship

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/dcrodman/archon/internal/character"
	"github.com/dcrodman/archon/internal/commands"
	"github.com/dcrodman/archon/internal/core"
	"github.com/dcrodman/archon/internal/core/bytes"
	"github.com/dcrodman/archon/internal/core/client"
	"github.com/dcrodman/archon/internal/shipgate"
)

var loginCopyright = []byte("Phantasy Star Online Blue Burst Game Server. Copyright 1999-2004 SONICTEAM.")

type GameServer struct {
	Name   string
	Config *core.Config
	Logger *zap.SugaredLogger

	shipgateClient shipgate.Shipgate
}

func (s *GameServer) Identifier() string {
	return "SHIP:GAME"
}

// Init connects to the shipgate.
func (s *GameServer) Init(ctx context.Context) error {
	s.shipgateClient = shipgate.NewClient(s.Config)
	return nil
}

func (s *GameServer) SetUpClient(c *client.Client) {
	c.CryptoSession = client.NewBlueBurstCryptoSession()
}

func (s *GameServer) Handshake(c *client.Client) error {
	pkt := &commands.Welcome{
		Header:       commands.BBHeader{Type: commands.LoginWelcomeType, Size: 0xC8},
		Copyright:    [96]byte{},
		ServerVector: [48]byte{},
		ClientVector: [48]byte{},
	}
	copy(pkt.Copyright[:], loginCopyright)
	copy(pkt.ServerVector[:], c.CryptoSession.ServerVector())
	copy(pkt.ClientVector[:], c.CryptoSession.ClientVector())

	return c.SendRaw(pkt)
}

func (s *GameServer) Handle(ctx context.Context, c *client.Client, data []byte) error {
	var cmdHeader commands.BBHeader
	bytes.StructFromBytes(data[:commands.BBHeaderSize], &cmdHeader)

	var err error
	switch cmdHeader.Type {
	case commands.LoginType:
		var loginPkt commands.Login
		bytes.StructFromBytes(data, &loginPkt)
		err = s.handleLogin(ctx, c, &loginPkt)
	default:
		s.Logger.Infof("received unknown command %x from %s", cmdHeader.Type, c.IPAddr())
	}
	return err
}

func (s *GameServer) handleLogin(ctx context.Context, c *client.Client, loginPkt *commands.Login) error {
	username := string(bytes.StripPadding(loginPkt.Username[:]))
	password := string(bytes.StripPadding(loginPkt.Password[:]))

	account, err := s.shipgateClient.AuthenticateAccount(ctx, &shipgate.AuthenticateAccountRequest{
		Username: username,
		Password: password,
	})
	if err != nil {
		switch err {
		case shipgate.ErrInvalidCredentials:
			return s.sendSecurity(c, commands.BBLoginErrorPassword)
		case shipgate.ErrAccountBanned:
			return s.sendSecurity(c, commands.BBLoginErrorBanned)
		default:
			sendErr := s.sendMessage(c, cases.Title(language.English).String(err.Error()))
			if sendErr == nil {
				return sendErr
			}
			return err
		}
	}
	c.Account = account
	c.ActiveSlot = loginPkt.Slot

	if err := s.sendSecurity(c, commands.BBLoginErrorNone); err != nil {
		return err
	}
	if err := s.sendLobbyMenu(c); err != nil {
		return err
	}
	return s.fetchAndSendCharacter(ctx, c)
}

func (s *GameServer) sendSecurity(c *client.Client, errorCode uint32) error {
	cfg := commands.ClientConfig{
		Magic:        c.Config.Magic,
		CharSelected: c.Config.CharSelected,
		SlotNum:      c.Config.SlotNum,
		Flags:        c.Config.Flags,
	}
	copy(cfg.Ports[:], c.Config.Ports[:])
	copy(cfg.Unused[:], c.Config.Unused[:])
	copy(cfg.Unused2[:], c.Config.Unused2[:])

	return c.Send(&commands.Security{
		Header:       commands.BBHeader{Type: commands.LoginSecurityType},
		ErrorCode:    errorCode,
		PlayerTag:    0x00010000,
		Guildcard:    c.Guildcard,
		TeamID:       c.TeamID,
		Config:       cfg,
		Capabilities: 0x00000102,
	})
}

func (s *GameServer) sendMessage(c *client.Client, message string) error {
	return c.Send(&commands.LoginClientMessage{
		Header:   commands.BBHeader{Type: commands.LoginClientMessageType},
		Language: 0x00450009,
		Message:  bytes.ConvertToUtf16(message),
	})
}

func (s *GameServer) sendLobbyMenu(c *client.Client) error {
	lobbyEntries := make([]commands.LobbyListEntry, s.Config.ShipServer.NumLobbies)
	for i := 0; i < s.Config.ShipServer.NumLobbies; i++ {
		lobbyEntries[i].MenuID = 0x001A0001
		lobbyEntries[i].LobbyID = uint32(i)
	}

	return c.Send(&commands.LobbyList{
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

func (s *GameServer) fetchAndSendCharacter(ctx context.Context, c *client.Client) error {
	resp, err := s.shipgateClient.FindCharacter(ctx, &shipgate.CharacterRequest{
		AccountId: c.Account.Id,
		Slot:      c.ActiveSlot,
	})
	if err != nil {
		return fmt.Errorf("error loading selected character: %v", err)
	}
	dbc := resp.Character

	charPkt := &commands.SyncCharacter{
		Header: commands.BBHeader{Type: commands.SyncCharacterType},
		Inventory: commands.PlayerInventory{
			HPFromMaterials: uint8(dbc.HpMaterialsUsed),
			TPFromMaterials: uint8(dbc.TpMaterialsUsed),
			// Items: ,
		},
		DisplayData: commands.PlayerDisplayData{
			Stats: commands.PlayerStats{
				ATP: uint16(dbc.Atp),
				MST: uint16(dbc.Mst),
				EVP: uint16(dbc.Evp),
				HP:  uint16(dbc.Hp),
				DFP: uint16(dbc.Dfp),
				ATA: uint16(dbc.Ata),
				LCK: uint16(dbc.Lck),
				// TODO: ESP, AttackRange, KnockbackRange
				Level:      uint32(dbc.Level),
				Experience: dbc.Experience,
				Meseta:     dbc.Meseta,
			},
			Visual: commands.PlayerVisual{
				NameColor:      NameColorNormal,
				SkinID:         uint8(dbc.ModelType),
				SectionID:      uint8(dbc.SectionId),
				Class:          uint8(dbc.Class),
				SkinFlag:       uint8(dbc.V2Flags),
				Costume:        uint16(dbc.Costume),
				Skin:           uint16(dbc.Skin),
				Face:           uint16(dbc.Face),
				Head:           uint16(dbc.Head),
				Hair:           uint16(dbc.Hair),
				HairColorRed:   uint16(dbc.HairRed),
				HairColorGreen: uint16(dbc.HairGreen),
				HairColorBlue:  uint16(dbc.HairBlue),
				ProportionX:    uint32(dbc.ProportionX),
				ProportionY:    uint32(dbc.ProportionY),
			},
			// TODO: Tech levels.
		},
		Signature: 0xC87ED5B1,
		PlayTime:  dbc.Playtime,
	}
	copy(charPkt.GuildCard.Description[:], dbc.GuildcardStr)
	copy(charPkt.DisplayData.Visual.Name[:], dbc.Name)
	if c.IsGm {
		charPkt.DisplayData.Visual.NameColor = NameColorGM
	}
	copy(charPkt.DisplayData.DispName[:], dbc.Name)

	// TODO: Tethealla doesn't really support editing this either, so will need to figure out
	// how to save this and return it to the player rather than using the default.
	copy(charPkt.KeyConfig[:], character.BaseKeyConfig[:0x16C])
	copy(charPkt.JoystickConfig[:], character.BaseKeyConfig[0x16C:])

	// TODO: Copy the techniques and inventory here.

	return c.Send(charPkt)
}

func (s *GameServer) sendJoinLobby(c *client.Client) error {
	return c.Send(&commands.JoinLobby{
		Header: commands.BBHeader{
			Type: commands.JoinLobbyType,
		},
		DisableUDP: 1,
	})
}
