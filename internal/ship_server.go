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
}

func (s *GameServer) Identifier() string {
	return "SHIP:GAME"
}

// Init connects to the shipgate.
func (s *GameServer) Init(ctx context.Context) error {
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
	c.ActiveSlot = loginPkt.Slot

	if err := SendSecurity(ctx, c, commands.BBLoginErrorNone); err != nil {
		return err
	}
	if err := SendLobbyMenu(ctx, c); err != nil {
		return err
	}
	return s.fetchAndSendCharacter(ctx, c)
}

func SendMessage(ctx context.Context, c *Client, message string) error {
	return c.Send(ctx, &commands.LoginClientMessage{
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

	return c.Send(ctx, &commands.LobbyList{
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

func (s *GameServer) fetchAndSendCharacter(ctx context.Context, c *Client) error {
	character, err := shipgate.Shipgate.FindCharacter(ctx, c.Account.ID, c.ActiveSlot)
	if err != nil {
		return fmt.Errorf("error loading selected character: %v", err)
	}

	charPkt := &commands.SyncCharacter{
		Header: commands.BBHeader{Type: commands.SyncCharacterType},
		Inventory: commands.PlayerInventory{
			HPFromMaterials: character.HPMaterialsUsed,
			TPFromMaterials: character.TPMaterialsUsed,
			// Items: ,
		},
		DisplayData: commands.PlayerDisplayData{
			Stats: commands.PlayerStats{
				ATP: character.ATP,
				MST: character.MST,
				EVP: character.EVP,
				HP:  character.HP,
				DFP: character.DFP,
				ATA: character.ATA,
				LCK: character.LCK,
				// TODO: ESP, AttackRange, KnockbackRange
				Level:      uint32(character.Level),
				Experience: character.Experience,
				Meseta:     character.Meseta,
			},
			Visual: commands.PlayerVisual{
				NameColor:      NameColorNormal,
				SkinID:         character.ModelType,
				SectionID:      character.SectionID,
				Class:          character.Class,
				SkinFlag:       character.V2Flags,
				Costume:        character.Costume,
				Skin:           character.Skin,
				Face:           character.Face,
				Head:           character.Head,
				Hair:           character.Hair,
				HairColorRed:   character.HairRed,
				HairColorGreen: character.HairGreen,
				HairColorBlue:  character.HairBlue,
				ProportionX:    uint32(character.ProportionX),
				ProportionY:    uint32(character.ProportionY),
			},
			// TODO: Tech levels.
		},
		Signature: 0xC87ED5B1,
		PlayTime:  character.Playtime,
	}
	copy(charPkt.GuildCard.Description[:], character.GuildcardStr)
	copy(charPkt.DisplayData.Visual.Name[:], character.Name)
	if c.IsGm {
		charPkt.DisplayData.Visual.NameColor = NameColorGM
	}
	copy(charPkt.DisplayData.DispName[:], character.Name)

	// TODO: Tethealla doesn't really support editing this either, so will need to figure out
	// how to save this and return it to the player rather than using the default.
	copy(charPkt.KeyConfig[:], BaseKeyConfig[:0x16C])
	copy(charPkt.JoystickConfig[:], BaseKeyConfig[0x16C:])

	// TODO: Copy the techniques and inventory here.

	return c.Send(ctx, charPkt)
}

func SendJoinLobby(ctx context.Context, c *Client) error {
	return c.Send(ctx, &commands.JoinLobby{
		Header: commands.BBHeader{
			Type: commands.JoinLobbyType,
		},
		DisableUDP: 1,
	})
}
