/*
 * The BLOCK and SHIP server logic.
 */
package block

//
//import (
//	"fmt"
//	"github.com/dcrodman/archon"
//	"github.com/dcrodman/archon/server"
//	"github.com/dcrodman/archon/server/ship"
//	"github.com/dcrodman/archon/util"
//	"net"
//	"strconv"
//)
//
//type BlockServer struct {
//	name string
//	port string
//
//	lobbyPkt archon.LobbyListPacket
//}
//
//func NewServer(name string, port int64) server.Server {
//	return &BlockServer{
//		name: name,
//		port: strconv.FormatInt(port, 10),
//	}
//}
//
//func (server *BlockServer) Name() string { return server.name }
//
//func (server *BlockServer) Port() string { return server.port }
//
//func (server *BlockServer) Init() error {
//	// Precompute our lobby list since this won't change once the server has started.
//	server.lobbyPkt.Header.Size = archon.BBHeaderSize
//	server.lobbyPkt.Header.Type = archon.LobbyListType
//	server.lobbyPkt.Header.Flags = uint32(archon.Config.BlockServer.NumLobbies)
//	for i := 0; i <= archon.Config.BlockServer.NumLobbies; i++ {
//		server.lobbyPkt.Lobbies = append(server.lobbyPkt.Lobbies, struct {
//			MenuId  uint32
//			LobbyId uint32
//			Padding uint32
//		}{
//			MenuId:  0x1A0001,
//			LobbyId: uint32(i),
//			Padding: 0,
//		})
//		server.lobbyPkt.Header.Size += 12
//	}
//	return nil
//}
//
//func (server *BlockServer) NewClient(conn *net.TCPConn) (*server.Client, error) {
//	return ship.NewShipClient(conn)
//}
//
//func (server *BlockServer) Handle(c *server.Client) error {
//	var hdr archon.BBHeader
//	util.StructFromBytes(c.Data()[:archon.BBHeaderSize], &hdr)
//
//	var err error
//	switch hdr.Type {
//	case archon.LoginType:
//		err = server.HandleShipLogin(c)
//	default:
//		archon.Log.Infof("Received unknown packet %02x from %s", hdr.Type, c.IPAddr())
//	}
//	return err
//}
//
//func (server *BlockServer) HandleShipLogin(c *server.Client) error {
//	if _, err := archon.VerifyAccount(c); err != nil {
//		return err
//	}
//	if err := server.sendSecurity(c, archon.BBLoginErrorNone, c.guildcard, c.teamId); err != nil {
//		return err
//	}
//	if err := server.sendBlockList(c); err != nil {
//		return err
//	}
//	return server.sendLobbyList(c)
//}
//
//func (server *BlockServer) sendSecurity(client *server.Client, errorCode archon.BBLoginError,
//	guildcard uint32, teamId uint32) error {
//	// Constants set according to how Newserv does it.
//	pkt := &archon.SecurityPacket{
//		Header:       archon.BBHeader{Type: archon.LoginSecurityType},
//		ErrorCode:    uint32(errorCode),
//		PlayerTag:    0x00010000,
//		Guildcard:    guildcard,
//		TeamId:       teamId,
//		Config:       &client.config,
//		Capabilities: 0x00000102,
//	}
//
//	archon.Log.Debug("Sending Security Packet")
//	return archon.EncryptAndSend(client, pkt)
//}
//
//// send the client the block list on the selection screen.
//func (server *BlockServer) sendBlockList(client *server.Client) error {
//	//data, size := util.BytesFromStruct(pkt)
//	archon.Log.Debug("Sending Block Packet - NOT IMPLEMENTED")
//	//return client.SendEncrypted(data, size)
//	return nil
//}
//
//// send the client the lobby list on the selection screen.
//func (server *BlockServer) sendLobbyList(client *server.Client) error {
//	archon.Log.Debug("Sending Lobby List Packet")
//	return archon.EncryptAndSend(client, server.lobbyPkt)
//}
