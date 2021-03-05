package character

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dcrodman/archon"
	"github.com/dcrodman/archon/internal/server/internal"
	"github.com/dcrodman/archon/internal/server/shipgate/api"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Server's internal representation of Ship connection information for
// the ship selection screen.
type ship struct {
	id   int
	name []byte
	ip   string
	port string
}

type shipgateClient struct {
	shipgateAddress string

	ships      []ship
	shipsMutex sync.RWMutex
}

// Return a list of the currently active ships.
func (sc *shipgateClient) getActiveShips() []ship {
	sc.shipsMutex.RLock()
	defer sc.shipsMutex.RUnlock()

	shipsCopy := make([]ship, len(sc.ships))
	copy(shipsCopy, sc.ships)
	return shipsCopy
}

// Starts a loop that makes an API request to the shipgate server over an interval
// in order to query the list of active ships. The result is parsed and stored in
// the Server's ships field.
func (sc *shipgateClient) startShipListRefreshLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second * 10):
			if err := sc.refreshShipList(); err != nil {
				archon.Log.Errorf(err.Error())
			}
		}
	}
}

func (sc *shipgateClient) refreshShipList() error {
	activeShips, err := sc.requestActiveShipList()
	if err != nil {
		return fmt.Errorf("shipgateClient: failed to connect to shipgate: %s", err)
	}

	sc.shipsMutex.Lock()
	sc.ships = activeShips
	sc.shipsMutex.Unlock()

	return nil
}

func (sc *shipgateClient) requestActiveShipList() ([]ship, error) {
	creds, err := credentials.NewClientTLSFromFile(viper.GetString("shipgate_certificate_file"), "")
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate file for shipgate: %s", err)
	}

	conn, err := grpc.Dial(sc.shipgateAddress, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to shipgate: %s", err)
	}

	defer conn.Close()

	shipgateClient := api.NewShipInfoServiceClient(conn)
	response, err := shipgateClient.GetActiveShips(context.Background(), &empty.Empty{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ships from shipgate: %s", err)
	}

	ships := make([]ship, 0)
	for _, s := range response.Ships {
		ships = append(ships, ship{
			id:   int(s.Id),
			name: internal.ConvertToUtf16(s.Name),
			ip:   s.Ip,
			port: s.Port,
		})
	}

	return ships, nil
}
