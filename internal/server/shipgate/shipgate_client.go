package shipgate

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dcrodman/archon/internal/auth"
	"github.com/dcrodman/archon/internal/data"
	"github.com/dcrodman/archon/internal/server/client"
	"google.golang.org/grpc/metadata"
	"gorm.io/gorm"

	"github.com/dcrodman/archon"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/dcrodman/archon/internal/packets"
	"github.com/dcrodman/archon/internal/server/internal"
	"github.com/dcrodman/archon/internal/server/shipgate/api"
)

type Client struct {
	shipgateAddress string
	shipgateClient  api.ShipgateServiceClient

	connectedShipsMutex sync.RWMutex
	connectedShips      []shipInfo
}

// Character servers internal representation of Ship connection information
// for the ship selection screen.
type shipInfo struct {
	id   int
	name []byte
	ip   string
	port string
}

func NewClient(shipgateAddress string) (*Client, error) {
	creds, err := credentials.NewClientTLSFromFile(viper.GetString("shipgate_certificate_file"), "")
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate file for shipgate: %s", err)
	}

	conn, err := grpc.Dial(shipgateAddress, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to shipgate: %s", err)
	}

	return &Client{
		shipgateAddress: shipgateAddress,
		// Lazy, but just leave the connection open until the server shuts down.
		shipgateClient: api.NewShipgateServiceClient(conn),
	}, nil
}

func (s *Client) StartShipRefreshLoop(ctx context.Context) error {
	// The first set is fetched synchronously so that the ship list will start populated.
	// Also gives us a chance to validate that the shipgate address is valid.
	if err := s.refreshShipList(); err != nil {
		return err
	}
	go s.startShipListRefreshLoop(ctx)

	return nil
}

func (s *Client) GetConnectedShipList() []packets.ShipListEntry {
	s.connectedShipsMutex.RLock()
	defer s.connectedShipsMutex.RUnlock()

	shipList := make([]packets.ShipListEntry, 0)
	for i, ship := range s.connectedShips {
		entry := packets.ShipListEntry{
			MenuID:   uint16(i + 1),
			ShipID:   uint32(ship.id),
			ShipName: [36]byte{},
		}
		copy(entry.ShipName[:], ship.name)
		shipList = append(shipList, entry)
	}
	if len(shipList) == 0 {
		// A "No Ships!" entry is shown if we either can't connect to the shipgate or
		// the shipgate doesn't report any connected ships.
		shipList = append(shipList, packets.ShipListEntry{
			MenuID: 0xFF, ShipID: 0xFF, ShipName: [36]byte{},
		})
		copy(shipList[0].ShipName[:], internal.ConvertToUtf16("No Ships!")[:])
	}
	return shipList
}

func (s *Client) GetSelectedShipAddress(selectedShip uint32) (net.IP, int, error) {
	s.connectedShipsMutex.Lock()
	defer s.connectedShipsMutex.Unlock()

	if selectedShip >= uint32(len(s.connectedShips)) {
		return nil, 0, fmt.Errorf("invalid ship selection: %d", selectedShip)
	}

	shipIP := net.ParseIP(s.connectedShips[selectedShip].ip).To4()
	shipPort, _ := strconv.Atoi(s.connectedShips[selectedShip].port)
	return shipIP, shipPort, nil
}

func (s *Client) AuthenticateAccount(ctx context.Context, c *client.Client, username, password string) (*data.Account, error) {
	md := metadata.New(map[string]string{
		"authorization": password,
	})

	accountpb, err := s.shipgateClient.AuthenticateAccount(
		metadata.NewOutgoingContext(ctx, md),
		&api.AccountAuthRequest{Username: username},
	)
	if err != nil {
		switch err {
		case auth.ErrInvalidCredentials:
			return nil, s.sendSecurity(c, packets.BBLoginErrorPassword)
		case auth.ErrAccountBanned:
			return nil, s.sendSecurity(c, packets.BBLoginErrorBanned)
		default:
			sendErr := s.sendMessage(c, strings.Title(err.Error()))
			if sendErr == nil {
				return nil, sendErr
			}
			return nil, err
		}
	}

	if err := s.sendSecurity(c, packets.BBLoginErrorNone); err != nil {
		return nil, err
	}

	rd, err := time.Parse(time.RFC3339, accountpb.GetRegistrationDate())
	if err != nil {
		return nil, err
	}

	var pl byte
	if b := accountpb.GetPriviledgeLevel(); len(b) != 0 {
		pl = b[0]
	}

	return &data.Account{
		Model: gorm.Model{
			ID: uint(accountpb.Id),
		},
		Username:         accountpb.GetUsername(),
		Password:         password,
		Email:            accountpb.GetEmail(),
		RegistrationDate: rd,
		Guildcard:        int(accountpb.GetGuildcard()),
		GM:               accountpb.GetGM(),
		Banned:           accountpb.GetBanned(),
		Active:           accountpb.GetActive(),
		TeamID:           int(accountpb.TeamId),
		PrivilegeLevel:   pl,
	}, nil
}

// Starts a loop that makes an API request to the shipgate server over an interval
// in order to query the list of active ships. The result is parsed and stored in
// the Server's ships field.
func (s *Client) startShipListRefreshLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second * 10):
			if err := s.refreshShipList(); err != nil {
				archon.Log.Errorf(err.Error())
			}
		}
	}
}

func (s *Client) refreshShipList() error {
	response, err := s.shipgateClient.GetActiveShips(context.Background(), &empty.Empty{})
	if err != nil {
		return fmt.Errorf("failed to fetch ships from shipgate: %s", err)
	}

	ships := make([]shipInfo, 0)
	for _, s := range response.Ships {
		ships = append(ships, shipInfo{
			id:   int(s.Id),
			name: internal.ConvertToUtf16(s.Name),
			ip:   s.Ip,
			port: s.Port,
		})
	}

	s.connectedShipsMutex.Lock()
	s.connectedShips = ships
	s.connectedShipsMutex.Unlock()
	return nil
}

func (s *Client) sendSecurity(c *client.Client, errorCode uint32) error {
	return c.Send(&packets.Security{
		Header:       packets.BBHeader{Type: packets.LoginSecurityType},
		ErrorCode:    errorCode,
		PlayerTag:    0x00010000,
		Guildcard:    c.Guildcard,
		TeamID:       c.TeamID,
		Config:       c.Config,
		Capabilities: 0x00000102,
	})
}

func (s *Client) sendMessage(c *client.Client, message string) error {
	return c.Send(&packets.LoginClientMessage{
		Header:   packets.BBHeader{Type: packets.LoginClientMessageType},
		Language: 0x00450009,
		Message:  internal.ConvertToUtf16(message),
	})
}
