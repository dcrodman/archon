package shipgate

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/dcrodman/archon"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/dcrodman/archon/internal/packets"
	"github.com/dcrodman/archon/internal/server/internal"
	"github.com/dcrodman/archon/internal/server/shipgate/api"
)

type ShipListClient struct {
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

func NewShipListClient(shipgateAddress string) *ShipListClient {
	return &ShipListClient{shipgateAddress: shipgateAddress}
}

func (s *ShipListClient) StartShipRefreshLoop(ctx context.Context) error {
	creds, err := credentials.NewClientTLSFromFile(viper.GetString("shipgate_certificate_file"), "")
	if err != nil {
		return fmt.Errorf("failed to load certificate file for shipgate: %s", err)
	}

	conn, err := grpc.Dial(s.shipgateAddress, grpc.WithTransportCredentials(creds))
	if err != nil {
		return fmt.Errorf("failed to connect to shipgate: %s", err)
	}
	// Lazy, but just leave the connection open until the server shuts down.

	s.shipgateClient = api.NewShipgateServiceClient(conn)

	// The first set is fetched synchronously so that the ship list will start populated.
	// Also gives us a chance to validate that the shipgate address is valid.
	if err := s.refreshShipList(); err != nil {
		return err
	}
	go s.startShipListRefreshLoop(ctx)

	return nil
}

func (s *ShipListClient) GetConnectedShipList() []packets.ShipListEntry {
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

func (s *ShipListClient) GetSelectedShipAddress(selectedShip uint32) (net.IP, int, error) {
	s.connectedShipsMutex.Lock()
	defer s.connectedShipsMutex.Unlock()

	if selectedShip >= uint32(len(s.connectedShips)) {
		return nil, 0, fmt.Errorf("invalid ship selection: %d", selectedShip)
	}

	shipIP := net.ParseIP(s.connectedShips[selectedShip].ip).To4()
	shipPort, _ := strconv.Atoi(s.connectedShips[selectedShip].port)
	return shipIP, shipPort, nil
}

// Starts a loop that makes an API request to the shipgate server over an interval
// in order to query the list of active ships. The result is parsed and stored in
// the Server's ships field.
func (s *ShipListClient) startShipListRefreshLoop(ctx context.Context) {
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

func (s *ShipListClient) refreshShipList() error {
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
