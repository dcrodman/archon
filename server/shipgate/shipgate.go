package shipgate

import (
	"context"
	"github.com/dcrodman/archon/server/shipgate/api"
	"github.com/golang/protobuf/ptypes/empty"
)

type shipServiceServer struct{}

func (s *shipServiceServer) GetActiveShips(ctx context.Context, empty *empty.Empty) (*api.ShipList, error) {
	return &api.ShipList{}, nil
}
