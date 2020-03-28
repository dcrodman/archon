package login

import (
	"errors"
	"github.com/dcrodman/archon"
	"github.com/dcrodman/archon/server"
	"net"
	"strconv"

	"github.com/dcrodman/archon/util"
	crypto "github.com/dcrodman/archon/util/encryption"
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

// Login sub-server definition.
type LoginServer struct {
	// Cached and parsed representation of the character port.
	charRedirectPort uint16
}

func NewServer() *LoginServer {
	return &LoginServer{}
}

func (server LoginServer) Name() string { return "LOGIN" }

func (server LoginServer) Port() string { return archon.Config.LoginServer.LoginPort }

func (server *LoginServer) Init() error {
	charPort, _ := strconv.ParseUint(archon.Config.LoginServer.CharacterPort, 10, 16)
	server.charRedirectPort = uint16(charPort)
	return nil
}

func (server *LoginServer) NewClient(conn *net.TCPConn) (*server.Client, error) {
	return NewLoginClient(conn)
}

func (server *LoginServer) Handle(c *server.Client) error {
	var hdr archon.BBHeader
	util.StructFromBytes(c.Data()[:archon.BBHeaderSize], &hdr)

	var err error
	switch hdr.Type {
	case archon.LoginType:
		err = server.HandleLogin(c)
	case archon.DisconnectType:
		// Just wait until we recv 0 from the client to d/c.
		break
	default:
		archon.Log.Infof("Received unknown packet %x from %s", hdr.Type, c.IPAddr())
	}
	return err
}

func (server *LoginServer) HandleLogin(client *server.Client) error {
	//loginPkt, err := VerifyAccount(client)
	_, err := archon.VerifyAccount(client)
	if err != nil {
		return err
	}

	// The first time we receive this packet the client will have included the
	// version string in the security data; check it.
	/* if ClientVersionString != string(util.StripPadding(loginPkt.Security[:])) {
		SendSecurity(client, BBLoginErrorPatch, 0, 0)
		return errors.New("Incorrect version string")
	} */

	// Newserv sets this field when the client first connects. I think this is
	// used to indicate that the client has made it through the LOGIN server,
	// but for now we'll just set it and leave it alone.
	client.config.Magic = 0x48615467

	ipAddr := archon.BroadcastIP()
	archon.SendSecurity(client, archon.BBLoginErrorNone, client.guildcard, client.teamId)
	return archon.SendRedirect(client, ipAddr[:], server.charRedirectPort)
}
