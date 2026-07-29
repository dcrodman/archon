package shipgate

import (
	"context"

	"github.com/dcrodman/archon/internal/core/proto"
	"google.golang.org/protobuf/types/known/emptypb"
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

func (s *service) GetActiveShips(ctx context.Context, _ *emptypb.Empty) (*ShipList, error) {
	s.logger.Debug("GetActiveShips")
	s.connectedShipsMutex.RLock()
	defer s.connectedShipsMutex.RUnlock()

	var shipList ShipList
	for _, connectedShip := range s.connectedShips {
		shipList.Ships = append(shipList.Ships, &proto.Ship{
			Id:   int32(connectedShip.id),
			Name: connectedShip.name,
			Ip:   connectedShip.ip,
			Port: connectedShip.port,
		})
	}

	return &shipList, nil
}

func (s *service) RegisterShip(ctx context.Context, req *RegisterShipRequest) (*emptypb.Empty, error) {
	s.logger.Debug("RegisterShip")
	s.connectedShipsMutex.Lock()
	defer s.connectedShipsMutex.Unlock()

	// Ships are never cleared from the map so that we can keep the IDs relatively
	// stable and allow for brief interruptions while preserving idempotency.
	if _, ok := s.connectedShips[req.Name]; ok {
		if !s.connectedShips[req.Name].active {
			s.logger.Infof("[SHIPGATE] reactivated ship %s at %s:%s", req.Name, req.Address, req.Port)
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
		s.logger.Infof("[SHIPGATE] registered ship %s at %s:%s", req.Name, req.Address, req.Port)
	}
	return &emptypb.Empty{}, nil
}
