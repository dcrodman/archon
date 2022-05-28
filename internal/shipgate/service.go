package shipgate

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/dcrodman/archon/internal/core/auth"
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

// Service implements the SHIPGATE server logic, which acts as the data and coordination
// layer between the other server components. It never directly interacts with the client,
// only handling RPC requests from other trusted servers.
type service struct {
	logger *logrus.Logger

	connectedShips      map[string]*ship
	connectedShipsMutex sync.RWMutex
}

func (s *service) GetActiveShips(ctx context.Context, _ *emptypb.Empty) (*ShipList, error) {
	s.connectedShipsMutex.RLock()
	defer s.connectedShipsMutex.RUnlock()

	ships := make([]*ShipList_Ship, 0)
	for _, connectedShip := range s.connectedShips {
		ships = append(ships, &ShipList_Ship{
			Id:   int32(connectedShip.id),
			Name: connectedShip.name,
			Ip:   connectedShip.ip,
			Port: connectedShip.port,
		})
	}

	return &ShipList{Ships: ships}, nil
}

func (s *service) RegisterShip(ctx context.Context, req *RegistrationRequest) (*emptypb.Empty, error) {
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

func (s *service) AuthenticateAccount(ctx context.Context, req *AccountAuthRequest) (*AccountAuthResponse, error) {
	account, err := auth.VerifyAccount(req.GetUsername(), req.GetPassword())
	if err != nil {
		return nil, err
	}

	return &AccountAuthResponse{
		Id:               uint64(account.ID),
		Username:         account.Username,
		Email:            account.Email,
		RegistrationDate: account.RegistrationDate.Format(time.RFC3339),
		Guildcard:        int64(account.Guildcard),
		GM:               account.GM,
		Banned:           account.Banned,
		Active:           account.Active,
		TeamId:           int64(account.TeamID),
		PriviledgeLevel:  []byte{account.PrivilegeLevel},
	}, nil
}
