// The patch package contains the implementations of the PATCH and DATA servers.
//
// PATCH is extremely simple and for the most part only exists as a first point
// of contact for the client, its chief responsibility being to send clients the
// address of the DATA server.
//
// DATA is responsible for exchanging file metadata with game clients in order to
// determine whether or not the client's files match the known patch files. If any
// of the patch file checksums do not equal the checksums of their corresponding
// client files (or do not exist), the DATA server sends the full file contents
// back to the client and forces a restart.
package patch

import (
	"github.com/dcrodman/archon"
	crypto "github.com/dcrodman/archon/internal/encryption"
	"github.com/dcrodman/archon/internal/packets"
	"github.com/dcrodman/archon/internal/server"
	"github.com/dcrodman/archon/internal/server/internal"
	"github.com/spf13/viper"
	"strconv"
	"sync"
)

// Convert the welcome message to UTF-16LE and cache it. PSOBB expects this prefix to the message,
//not completely sure why. Language perhaps?

var (
	messageBytes []byte
	messageInit  sync.Once

	// Copyright message expected by the client for the patch welcome.
	copyright = []byte("Patch Server. Copyright SonicTeam, LTD. 2001")
)

func getWelcomeMessage() ([]byte, uint16) {
	messageInit.Do(func() {
		messageBytes = internal.ConvertToUtf16(viper.GetString("patch_server.welcome_message"))

		if len(messageBytes) > (1 << 16) {
			archon.Log.Warn("patch server welcome message exceeds 65,000 characters")
			messageBytes = messageBytes[:1<<16-2]
		}

		messageBytes = append([]byte{0xFF, 0xFE}, messageBytes...)
	})

	return messageBytes, uint16(len(messageBytes))
}

type Server struct {
	name string
	// Parsed representation of the login port.
	dataRedirectPort uint16
}

func NewServer(name, dataPort string) server.Backend {
	// Convert the data port to a BE uint for the redirect packet.
	parsedDataPort, _ := strconv.ParseUint(dataPort, 10, 16)
	dataRedirectPort := uint16((parsedDataPort >> 8) | (parsedDataPort << 8))

	return &Server{name: name, dataRedirectPort: dataRedirectPort}
}

func (s *Server) Name() string { return s.name }
func (s *Server) Init() error  { return nil }

func (s *Server) CreateExtension() server.ClientExtension {
	return &patchClientExtension{
		clientCrypt: crypto.NewPCCrypt(),
		serverCrypt: crypto.NewPCCrypt(),
	}
}

func (s *Server) StartSession(c *server.Client) error {
	ext := c.Extension.(*patchClientExtension)

	// Send the welcome packet to a client with the copyright message and encryption vectors.
	pkt := packets.PatchWelcome{
		Header: packets.PCHeader{Type: packets.PatchWelcomeType, Size: 0x4C},
	}
	copy(pkt.Copyright[:], copyright)
	copy(pkt.ClientVector[:], ext.clientCrypt.Vector)
	copy(pkt.ServerVector[:], ext.serverCrypt.Vector)

	return c.SendRaw(pkt)
}

func (s *Server) Handle(c *server.Client, data []byte) error {
	var header packets.PCHeader
	internal.StructFromBytes(data[:packets.PCHeaderSize], &header)

	var err error
	switch header.Type {
	case packets.PatchWelcomeType:
		err = s.sendWelcomeAck(c)
	case packets.PatchHandshakeType:
		if err := s.sendWelcomeMessage(c); err == nil {
			err = s.sendPatchRedirect(c)
		}
	default:
		archon.Log.Infof("Received unknown packet %2x from %s", header.Type, c.IPAddr())
	}
	return err
}

func (s *Server) sendWelcomeAck(c *server.Client) error {
	// PatchHandshakeType is treated as an ack in this case.
	return c.Send(&packets.PCHeader{
		Size: 0x04,
		Type: packets.PatchHandshakeType,
	})
}

// Message displayed on the patch download screen.
func (s *Server) sendWelcomeMessage(c *server.Client) error {
	message, size := getWelcomeMessage()
	pkt := &packets.PatchWelcomeMessage{
		Header: packets.PCHeader{
			Size: packets.PCHeaderSize + size,
			Type: packets.PatchMessageType,
		},
		Message: message,
	}

	return c.Send(pkt)
}

// send the redirect packet, providing the IP and port of the next server.
func (s *Server) sendPatchRedirect(c *server.Client) error {
	pkt := packets.PatchRedirect{
		Header:  packets.PCHeader{Type: packets.PatchRedirectType},
		IPAddr:  [4]uint8{},
		Port:    s.dataRedirectPort,
		Padding: 0,
	}

	hostnameBytes := archon.BroadcastIP()
	copy(pkt.IPAddr[:], hostnameBytes[:])

	return c.Send(pkt)
}
