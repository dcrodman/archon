/*
 * The BLOCK and SHIP server logic.
 */
package main

import (
	"github.com/dcrodman/archon/util"
	"net"
)

type BlockServer struct {
	name string
	port string

	lobbyPkt LobbyListPacket
}

func (server *BlockServer) Name() string { return server.name }

func (server *BlockServer) Port() string { return server.port }

func (server *BlockServer) Init() error {
	// Precompute our lobby list since this won't change once the server has started.
	server.lobbyPkt.Header.Size = BBHeaderSize
	server.lobbyPkt.Header.Type = LobbyListType
	server.lobbyPkt.Header.Flags = uint32(Config.BlockServer.NumLobbies)
	for i := 0; i <= Config.BlockServer.NumLobbies; i++ {
		server.lobbyPkt.Lobbies = append(server.lobbyPkt.Lobbies, struct {
			MenuId  uint32
			LobbyId uint32
			Padding uint32
		}{
			MenuId:  0x1A0001,
			LobbyId: uint32(i),
			Padding: 0,
		})
		server.lobbyPkt.Header.Size += 12
	}
	return nil
}

func (server *BlockServer) NewClient(conn *net.TCPConn) (*Client, error) {
	return NewShipClient(conn)
}

func (server *BlockServer) Handle(c *Client) error {
	var hdr BBHeader
	util.StructFromBytes(c.Data()[:BBHeaderSize], &hdr)

	var err error
	switch hdr.Type {
	case LoginType:
		err = server.HandleShipLogin(c)
	default:
		Log.Infof("Received unknown packet %02x from %s", hdr.Type, c.IPAddr())
	}
	return err
}

func (server *BlockServer) HandleShipLogin(c *Client) error {
	if _, err := VerifyAccount(c); err != nil {
		return err
	}
	if err := server.sendSecurity(c, BBLoginErrorNone, c.guildcard, c.teamId); err != nil {
		return err
	}
	if err := server.sendBlockList(c); err != nil {
		return err
	}
	return server.sendLobbyList(c)
}

func (server *BlockServer) sendSecurity(client *Client, errorCode BBLoginError,
	guildcard uint32, teamId uint32) error {
	// Constants set according to how Newserv does it.
	pkt := &SecurityPacket{
		Header:       BBHeader{Type: LoginSecurityType},
		ErrorCode:    uint32(errorCode),
		PlayerTag:    0x00010000,
		Guildcard:    guildcard,
		TeamId:       teamId,
		Config:       &client.config,
		Capabilities: 0x00000102,
	}

	Log.Debug("Sending Security Packet")
	return EncryptAndSend(client, pkt)
}

// Send the client the block list on the selection screen.
func (server *BlockServer) sendBlockList(client *Client) error {
	//data, size := util.BytesFromStruct(pkt)
	Log.Debug("Sending Block Packet - NOT IMPLEMENTED")
	//return client.SendEncrypted(data, size)
	return nil
}

// Send the client the lobby list on the selection screen.
func (server *BlockServer) sendLobbyList(client *Client) error {
	Log.Debug("Sending Lobby List Packet")
	return EncryptAndSend(client, server.lobbyPkt)
}
