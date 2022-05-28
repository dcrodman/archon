package block

import (
	"context"

	"github.com/sirupsen/logrus"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/dcrodman/archon/internal/core"
	"github.com/dcrodman/archon/internal/core/auth"
	"github.com/dcrodman/archon/internal/core/bytes"
	"github.com/dcrodman/archon/internal/core/client"
	"github.com/dcrodman/archon/internal/core/data"
	"github.com/dcrodman/archon/internal/packets"
	"github.com/dcrodman/archon/internal/shipgate"
)

var loginCopyright = []byte("Phantasy Star Online Blue Burst Game Server. Copyright 1999-2004 SONICTEAM.")

type Server struct {
	Name   string
	Config *core.Config
	Logger *logrus.Logger

	shipgateClient shipgate.Shipgate
	shipListClient *shipgate.ShipRegistrationClient
}

func (s *Server) Identifier() string {
	return s.Name
}

// Init connects to the shipgate.
func (s *Server) Init(ctx context.Context) error {
	s.shipgateClient = shipgate.NewRPCClient(s.Config)
	s.shipListClient = &shipgate.ShipRegistrationClient{
		Logger:         s.Logger,
		ShipgateClient: s.shipgateClient,
	}
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

	account, err := s.shipgateClient.AuthenticateAccount(ctx, &shipgate.AccountAuthRequest{
		Username: username,
		Password: password,
	})
	if err != nil {
		switch err {
		case auth.ErrInvalidCredentials:
			return s.sendSecurity(c, packets.BBLoginErrorPassword)
		case auth.ErrAccountBanned:
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

	if err := s.sendSecurity(c, packets.BBLoginErrorNone); err != nil {
		return err
	}
	if err := s.sendLobbyList(c); err != nil {
		return err
	}
	if err := s.fetchAndSendCharacter(c); err != nil {
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

func (s *Server) fetchAndSendCharacter(c *client.Client) error {
	// TODO: Load this from shipgate
	character := &data.Character{}

	charPkt := &packets.FullCharacter{
		Header: packets.BBHeader{Type: packets.FullCharacterType},
		// TODO: All of these.
		// NumInventoryItems uint8
		// HPMaterials       uint8
		// TPMaterials       uint8
		// Language          uint8
		// Inventory         [30]InventorySlot
		ATP:        character.ATP,
		MST:        character.MST,
		EVP:        character.EVP,
		HP:         character.HP,
		DFP:        character.DFP,
		ATA:        character.ATA,
		LCK:        character.LCK,
		Level:      uint16(character.Level),
		Experience: character.Experience,
		Meseta:     character.Meseta,
		// NameColorBlue
		// NameColorGreen
		// NameColorRed
		// NameColorTransparency
		// SkinID
		SectionID: character.SectionID,
		Class:     character.Class,
		// SkinFlag
		Costume:        character.Costume,
		Skin:           character.Skin,
		Face:           character.Face,
		Head:           character.Head,
		Hair:           character.Hair,
		HairColorRed:   character.HairRed,
		HairColorGreen: character.HairGreen,
		HairColorBlue:  character.HairBlue,
		// ProportionX:    uint32(character.ProportionX),
		// ProportionY:    uint32(character.ProportionY),
		PlayTime: character.Playtime,
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
