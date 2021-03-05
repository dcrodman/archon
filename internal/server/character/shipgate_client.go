package character

import (
	"context"
	"fmt"
	"time"

	"github.com/dcrodman/archon"
	"github.com/dcrodman/archon/internal/server/internal"
	"github.com/dcrodman/archon/internal/server/shipgate/api"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Character servers internal representation of Ship connection information
// for the ship selection screen.
type ship struct {
	id   int
	name []byte
	ip   string
	port string
}

func (s *Server) startShipRefreshLoop(ctx context.Context) error {
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

// Starts a loop that makes an API request to the shipgate server over an interval
// in order to query the list of active ships. The result is parsed and stored in
// the Server's ships field.
func (s *Server) startShipListRefreshLoop(ctx context.Context) {
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

func (s *Server) refreshShipList() error {
	response, err := s.shipgateClient.GetActiveShips(context.Background(), &empty.Empty{})
	if err != nil {
		return fmt.Errorf("failed to fetch ships from shipgate: %s", err)
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

	s.connectedShipsMutex.Lock()
	s.connectedShips = ships
	s.connectedShipsMutex.Unlock()
	return nil
}
