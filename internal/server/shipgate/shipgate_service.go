package shipgate

import (
	"context"
	"sync"

	"github.com/dcrodman/archon"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/dcrodman/archon/internal/server/shipgate/api"
)

type ship struct {
	id   int
	name string
	ip   string
	port string
	// TODO: Need a way to deregister these reliably. Ships will probably need
	// their own gRPC service so I'm deferring the heartbeating problem until then.
	active bool
}

// shipgateServiceServer implements the SHIPGATE server logic, which never directly
// interacts with the client. Instead it is responsible for coordinating information
// transfer between the CHARACTER, SHIP, and BLOCK servers.
type shipgateServiceServer struct {
	api.UnimplementedShipgateServiceServer

	connectedShips      map[string]*ship
	connectedShipsMutex sync.RWMutex
}

func (s *shipgateServiceServer) GetActiveShips(ctx context.Context, _ *emptypb.Empty) (*api.ShipList, error) {
	s.connectedShipsMutex.RLock()
	defer s.connectedShipsMutex.RUnlock()

	ships := make([]*api.ShipList_Ship, 0)
	for _, connectedShip := range s.connectedShips {
		ships = append(ships, &api.ShipList_Ship{
			Id:   int32(connectedShip.id),
			Name: connectedShip.name,
			Ip:   connectedShip.ip,
			Port: connectedShip.port,
		})
	}

	return &api.ShipList{Ships: ships}, nil
}

func (s *shipgateServiceServer) RegisterShip(ctx context.Context, req *api.RegistrationRequest) (*emptypb.Empty, error) {
	s.connectedShipsMutex.Lock()
	defer s.connectedShipsMutex.Unlock()

	// Ships are never cleared from the map so that we can keep the IDs relatively
	// stable and allow for brief interruptions while preserving idempotency.
	if _, ok := s.connectedShips[req.Name]; ok {
		if !s.connectedShips[req.Name].active {
			archon.Log.Infof("SHIPGATE reactivated ship %s at %s:%s", req.Name, req.Address, req.Port)
		}
		s.connectedShips[req.Name].active = true
		s.connectedShips[req.Name].ip = req.Address
		s.connectedShips[req.Name].port = req.Port
	} else {
		s.connectedShips[req.Name] = &ship{
			id:   len(s.connectedShips) + 1,
			name: req.Name,
			ip:   req.Address,
			port: req.Port,
		}
		archon.Log.Infof("SHIPGATE registered ship %s at %s:%s", req.Name, req.Address, req.Port)
	}
	return &emptypb.Empty{}, nil
}
