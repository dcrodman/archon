package block

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/dcrodman/archon/internal/core"
	"github.com/dcrodman/archon/internal/core/bytes"
	"github.com/dcrodman/archon/internal/core/client"
	"github.com/dcrodman/archon/internal/packets"
	"github.com/dcrodman/archon/internal/shipgate"
)

var loginCopyright = []byte("Phantasy Star Online Blue Burst Game Server. Copyright 1999-2004 SONICTEAM.")

type Server struct {
	Name   string
	Config *core.Config
	Logger *logrus.Logger

	shipgateClient shipgate.Shipgate
}

func (s *Server) Identifier() string {
	return s.Name
}

// Init connects to the shipgate.
func (s *Server) Init(ctx context.Context) error {
	s.shipgateClient = shipgate.NewRPCClient(s.Config)
	return nil
}

func (s *Server) SetUpClient(c *client.Client) {
	c.CryptoSession = client.NewBlueBurstCryptoSession()
	c.DebugTags["server_type"] = "block"
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
	default:
		s.Logger.Infof("received unknown packet %x from %s", packetHeader.Type, c.IPAddr())
	}
	return err
}

func (s *Server) handleLogin(ctx context.Context, c *client.Client, loginPkt *packets.Login) error {
	username := string(bytes.StripPadding(loginPkt.Username[:]))
	password := string(bytes.StripPadding(loginPkt.Password[:]))

	account, err := s.shipgateClient.AuthenticateAccount(ctx, &shipgate.AuthenticateAccountRequest{
		Username: username,
		Password: password,
	})
	if err != nil {
		switch err {
		case shipgate.ErrInvalidCredentials:
			return s.sendSecurity(c, packets.BBLoginErrorPassword)
		case shipgate.ErrAccountBanned:
			return s.sendSecurity(c, packets.BBLoginErrorBanned)
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

	if err := s.sendSecurity(c, packets.BBLoginErrorNone); err != nil {
		return err
	}
	if err := s.sendLobbyList(c); err != nil {
		return err
	}
	if err := s.fetchAndSendCharacter(ctx, c); err != nil {
		return err
	}

	return s.sendFullCharacterEnd(c)
}

func (s *Server) sendSecurity(c *client.Client, errorCode uint32) error {
	cfg := packets.ClientConfig{
		Magic:        c.Config.Magic,
		CharSelected: c.Config.CharSelected,
		SlotNum:      c.Config.SlotNum,
		Flags:        c.Config.Flags,
	}
	copy(cfg.Ports[:], c.Config.Ports[:])
	copy(cfg.Unused[:], c.Config.Unused[:])
	copy(cfg.Unused2[:], c.Config.Unused2[:])

	return c.Send(&packets.Security{
		Header:       packets.BBHeader{Type: packets.LoginSecurityType},
		ErrorCode:    errorCode,
		PlayerTag:    0x00010000,
		Guildcard:    c.Guildcard,
		TeamID:       c.TeamID,
		Config:       cfg,
		Capabilities: 0x00000102,
	})
}

func (s *Server) sendMessage(c *client.Client, message string) error {
	return c.Send(&packets.LoginClientMessage{
		Header:   packets.BBHeader{Type: packets.LoginClientMessageType},
		Language: 0x00450009,
		Message:  bytes.ConvertToUtf16(message),
	})
}

func (s *Server) sendLobbyList(c *client.Client) error {
	lobbyEntries := make([]packets.LobbyListEntry, s.Config.BlockServer.NumLobbies)
	for i := 0; i < s.Config.BlockServer.NumLobbies; i++ {
		lobbyEntries[i].MenuID = 0x001A0001
		lobbyEntries[i].LobbyID = uint32(i)
	}

	return c.Send(&packets.LobbyList{
		Header: packets.BBHeader{
			Type:  packets.LobbyListType,
			Flags: 0x0F,
		},
		Lobbies: lobbyEntries,
	})
}

func (s *Server) fetchAndSendCharacter(ctx context.Context, c *client.Client) error {
	resp, err := s.shipgateClient.FindCharacter(ctx, &shipgate.CharacterRequest{
		AccountId: c.Account.Id,
		Slot:      c.ActiveSlot,
	})
	if err != nil {
		return fmt.Errorf("error loading selected character: %v", err)
	}
	character := resp.Character

	charPkt := &packets.FullCharacter{
		Header: packets.BBHeader{Type: packets.FullCharacterType},
		// TODO: All of these.
		// NumInventoryItems uint8
		// HPMaterials       uint8
		// TPMaterials       uint8
		// Language          uint8
		// Inventory         [30]InventorySlot
		ATP:        uint16(character.Atp),
		MST:        uint16(character.Mst),
		EVP:        uint16(character.Evp),
		HP:         uint16(character.Hp),
		DFP:        uint16(character.Dfp),
		ATA:        uint16(character.Ata),
		LCK:        uint16(character.Lck),
		Level:      uint16(character.Level),
		Experience: character.Experience,
		Meseta:     character.Meseta,
		// NameColorBlue
		// NameColorGreen
		// NameColorRed
		// NameColorTransparency
		// SkinID
		SectionID: uint8(character.SectionId),
		Class:     uint8(character.Class),
		// SkinFlag
		Costume:        uint16(character.Costume),
		Skin:           uint16(character.Skin),
		Face:           uint16(character.Face),
		Head:           uint16(character.Head),
		Hair:           uint16(character.Hair),
		HairColorRed:   uint16(character.HairRed),
		HairColorGreen: uint16(character.HairGreen),
		HairColorBlue:  uint16(character.HairBlue),
		ProportionX:    uint32(character.ProportionX),
		ProportionY:    uint32(character.ProportionY),
		PlayTime:       character.Playtime,
	}
	copy(charPkt.GuildcardStr[:], character.GuildcardStr)
	copy(charPkt.Name[:], character.Name)
	// copy(charPkt.KeyConfig[:], character.)
	// copy(charPkt.Techniques[:], character.)
	// copy(charPkt.Options[:], )
	// copy(charPkt.QuestData[:], )

	return c.Send(charPkt)
}

func (s *Server) sendFullCharacterEnd(c *client.Client) error {
	// Acts as an EOF for the full character data.
	return c.Send(&packets.BBHeader{
		Type: packets.FullCharacterEndType,
	})
}
