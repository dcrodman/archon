package login

import (
	"errors"
	"github.com/dcrodman/archon"
	"github.com/dcrodman/archon/server"
	"github.com/dcrodman/archon/util"
	crypto "github.com/dcrodman/archon/util/encryption"
	"net"
)

// Struct for caching the parameter chunk data and header so
// that the param files aren't re-read every time.
type parameterEntry struct {
	Size     uint32
	Checksum uint32
	Offset   uint32
	Filename [0x40]uint8
}

// Create and initialize a new Login client so long as we're able
// to send the welcome packet to begin encryption.
func NewLoginClient(conn *net.TCPConn) (*server.Client, error) {
	var err error
	cCrypt := crypto.NewBBCrypt()
	sCrypt := crypto.NewBBCrypt()
	lc := server.NewClient(conn, archon.BBHeaderSize, cCrypt, sCrypt)

	if archon.SendWelcome(lc) != nil {
		err = errors.New("Error sending welcome packet to: " + lc.IPAddr())
		lc = nil
	}
	return lc, err
}

type LoginServer struct {
	name             string
	port             string
	charRedirectPort uint16
}

//func NewServer(name, port, characterPort string) server.Server {
//	charPort, _ := strconv.ParseUint(characterPort, 10, 16)
//	return &LoginServer{name: name, port: port, charRedirectPort: uint16(charPort)}
//}

func (server LoginServer) Name() string       { return server.name }
func (server LoginServer) Port() string       { return server.port }
func (server LoginServer) HeaderSize() uint16 { return archon.BBHeaderSize }

//
//func (server LoginServer) AcceptClient(cs *server.ConnectionState) (server.Client2, error) {
//	return NewLoginClient(conn)
//}

func (server LoginServer) Handle(c server.Client2) error {
	var hdr archon.BBHeader
	util.StructFromBytes(c.ConnectionState().Data()[:archon.BBHeaderSize], &hdr)

	var err error
	switch hdr.Type {
	case archon.LoginType:
		err = server.HandleLogin(c)
	case archon.DisconnectType:
		// Just wait until we recv 0 from the client to d/c.
		break
	default:
		archon.Log.Infof("Received unknown packet %x from %s", hdr.Type, c.ConnectionState().IPAddr())
	}
	return err
}

func (server *LoginServer) HandleLogin(c server.Client2) error {
	//_, err := archon.VerifyAccount(client)
	//if err != nil {
	//	return err
	//}

	// The first time we receive this packet the client will have included the
	// version string in the security data; check it.
	/* if ClientVersionString != string(util.StripPadding(loginPkt.Security[:])) {
		SendSecurity(client, BBLoginErrorPatch, 0, 0)
		return errors.New("Incorrect version string")
	} */

	// Newserv sets this field when the client first connects. I think this is
	// used to indicate that the client has made it through the LOGIN server,
	// but for now we'll just set it and leave it alone.
	//client.config.Magic = 0x48615467

	//ipAddr := archon.BroadcastIP()
	//archon.SendSecurity(client, archon.BBLoginErrorNone, client.guildcard, client.teamId)
	//return archon.SendRedirect(client, ipAddr[:], server.charRedirectPort)
	return nil
}
