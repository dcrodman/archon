package block

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/dcrodman/archon/internal/character"
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
	Logger *zap.SugaredLogger

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
	case packets.CharacterDataType:
		// TODO: Probably have some data to copy in here.
		err = s.sendPacket67(c)
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

const (
	// Values stolen from Tethealla, though it almost certainly doesn't matter.
	NameColorNormal = 0xFFFFFFFF
	NameColorGM     = 0xFF1D94F7
)

func (s *Server) fetchAndSendCharacter(ctx context.Context, c *client.Client) error {
	resp, err := s.shipgateClient.FindCharacter(ctx, &shipgate.CharacterRequest{
		AccountId: c.Account.Id,
		Slot:      c.ActiveSlot,
	})
	if err != nil {
		return fmt.Errorf("error loading selected character: %v", err)
	}
	dbCharacter := resp.Character

	charPkt := &packets.FullCharacter{
		Header: packets.BBHeader{Type: packets.FullCharacterType},
		// NumInventoryItems: 0,
		HPMaterials:    uint8(dbCharacter.HpMaterialsUsed),
		TPMaterials:    uint8(dbCharacter.TpMaterialsUsed),
		Language:       0,
		ATP:            uint16(dbCharacter.Atp),
		MST:            uint16(dbCharacter.Mst),
		EVP:            uint16(dbCharacter.Evp),
		HP:             uint16(dbCharacter.Hp),
		DFP:            uint16(dbCharacter.Dfp),
		ATA:            uint16(dbCharacter.Ata),
		LCK:            uint16(dbCharacter.Lck),
		Level:          uint16(dbCharacter.Level),
		Experience:     dbCharacter.Experience,
		SkinID:         uint16(dbCharacter.ModelType),
		SectionID:      uint8(dbCharacter.SectionId),
		Class:          uint8(dbCharacter.Class),
		SkinFlag:       uint8(dbCharacter.V2Flags),
		Costume:        uint16(dbCharacter.Costume),
		Skin:           uint16(dbCharacter.Skin),
		Face:           uint16(dbCharacter.Face),
		Head:           uint16(dbCharacter.Head),
		Hair:           uint16(dbCharacter.Hair),
		HairColorRed:   uint16(dbCharacter.HairRed),
		HairColorGreen: uint16(dbCharacter.HairGreen),
		HairColorBlue:  uint16(dbCharacter.HairBlue),
		ProportionX:    uint32(dbCharacter.ProportionX),
		ProportionY:    uint32(dbCharacter.ProportionY),
		PlayTime:       dbCharacter.Playtime,
	}
	copy(charPkt.GuildcardStr[:], dbCharacter.GuildcardStr)
	copy(charPkt.Name[:], dbCharacter.Name)

	charPkt.NameColor = NameColorNormal
	if c.IsGm {
		charPkt.NameColor = NameColorGM
	}

	// TODO: Tethealla doesn't really support editing this either, so will need to figure out
	// how to save this and return it to the player rather than using the default.
	copy(charPkt.KeyConfig[:], character.BaseKeyConfig[:])

	// TODO: Copy the techniques and inventory here.

	return c.Send(charPkt)
}

func (s *Server) sendFullCharacterEnd(c *client.Client) error {
	// Acts as an EOF for the full character data.
	return c.Send(&packets.BBHeader{
		Type: packets.FullCharacterEndType,
	})
}

func (s *Server) sendPacket67(c *client.Client) error {
	return c.Send(&packets.Packet67{
		Header: packets.BBHeader{
			Type: 0x67,
		},
		Unknown1:  0x01,
		PlayerTag: 0x00010000,
		// Something: ,
	})
}
