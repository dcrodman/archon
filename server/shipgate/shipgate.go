package shipgate

import (
	"context"
	"github.com/dcrodman/archon/server/shipgate/api"
	"github.com/golang/protobuf/ptypes/empty"
)

type shipServiceServer struct{}

func (s *shipServiceServer) GetActiveShips(ctx context.Context, _ *empty.Empty) (*api.ShipList, error) {
	return &api.ShipList{
		Ships: []*api.ShipList_Ship{
			{Id: 1, Name: "Test", Ip: "192.168.1.4", Port: "15001"},
		},
	}, nil
}
