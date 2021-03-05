package shipgate

import (
	"context"

	"github.com/dcrodman/archon/internal/server/shipgate/api"
	"google.golang.org/protobuf/types/known/emptypb"
)

type shipgateServiceServer struct {
	api.UnimplementedShipgateServiceServer
}

func (s *shipgateServiceServer) GetActiveShips(ctx context.Context, _ *emptypb.Empty) (*api.ShipList, error) {
	return &api.ShipList{
		Ships: []*api.ShipList_Ship{
			{Id: 1, Name: "Test 1", Ip: "192.168.1.4", Port: "15001"},
			{Id: 2, Name: "Test 2", Ip: "192.168.1.4", Port: "15002"},
		},
	}, nil
}

func (s *shipgateServiceServer) RegisterShip(ctx context.Context, req *api.RegistrationRequest) (*emptypb.Empty, error) {
	panic("not implemented") // TODO: Implement
}
