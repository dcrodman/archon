package shipgate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/gorm"

	"github.com/dcrodman/archon/internal/core/data"
	"github.com/dcrodman/archon/internal/core/proto"
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
	logger              *logrus.Logger
	db                  *gorm.DB
	connectedShips      map[string]*ship
	connectedShipsMutex sync.RWMutex
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

var (
	ErrUnknown            = errors.New("an unexpected error occurred, please contact your server administrator")
	ErrInvalidCredentials = errors.New("username/combination password not found")
	ErrAccountBanned      = errors.New("this account has been suspended")
)

func (s *service) AuthenticateAccount(ctx context.Context, req *AuthenticateAccountRequest) (*proto.Account, error) {
	s.logger.Debug("AuthenticateAccount")
	account, err := data.FindAccountByUsername(s.db, req.Username)
	if err != nil {
		return nil, ErrUnknown
	}

	if account == nil || account.Password != HashPassword(req.Password) {
		return nil, ErrInvalidCredentials
	} else if account.Banned {
		return nil, ErrAccountBanned
	}

	return &proto.Account{
		Id:               uint64(account.ID),
		Username:         account.Username,
		Email:            account.Email,
		RegistrationDate: account.RegistrationDate.Format(time.RFC3339),
		Guildcard:        uint64(account.Guildcard),
		Gm:               account.GM,
		Banned:           account.Banned,
		Active:           account.Active,
		TeamId:           int64(account.TeamID),
		PriviledgeLevel:  []byte{account.PrivilegeLevel},
	}, nil
}

// HashPassword returns a version of password with Archon's chosen hashing strategy.
func HashPassword(password string) string {
	hash := sha256.New()
	if _, err := hash.Write(stripPadding([]byte(password))); err != nil {
		panic(fmt.Errorf("error generating password hash: %v", err))
	}
	return hex.EncodeToString(hash.Sum(nil)[:])
}

func stripPadding(b []byte) []byte {
	for i := len(b) - 1; i >= 0; i-- {
		if b[i] != 0 {
			return b[:i+1]
		}
	}
	return b
}

func (s *service) FindCharacter(ctx context.Context, req *CharacterRequest) (*FindCharacterResponse, error) {
	s.logger.Debug("FindCharacter")

	character, err := data.FindCharacter(s.db, uint(req.AccountId), req.Slot)
	if err != nil {
		return nil, fmt.Errorf("error retrieving character for account %d slot %d: %w", req.AccountId, req.Slot, err)
	}

	resp := &FindCharacterResponse{
		Exists:    false,
		Character: &proto.Character{},
	}
	if character != nil {
		resp.Exists = true
		resp.Character = characterToProto(character)
	}
	return resp, nil
}

func (s *service) UpsertCharacter(ctx context.Context, req *UpsertCharacterRequest) (*emptypb.Empty, error) {
	s.logger.Debug("UpsertCharacter")

	character := characterFromProto(req.Character)
	character.AccountID = req.AccountId
	if err := data.UpdateCharacter(s.db, character); err != nil {
		return nil, fmt.Errorf("error updating character for account %d slot %d: %w", req.AccountId, req.Character.Slot, err)
	}
	return &emptypb.Empty{}, nil
}

func (s *service) DeleteCharacter(ctx context.Context, req *CharacterRequest) (*emptypb.Empty, error) {
	s.logger.Debug("DeleteCharacter")

	if err := data.DeleteCharacter(s.db, uint(req.AccountId), req.Slot); err != nil {
		return nil, fmt.Errorf("error deleting character for account %d slot %d: %w", req.AccountId, req.Slot, err)
	}
	return &emptypb.Empty{}, nil
}

func (s *service) GetGuildcardEntries(ctx context.Context, req *GetGuildcardEntriesRequest) (*GetGuildcardEntriesResponse, error) {
	s.logger.Debug("GetGuildcardEntries")

	entries, err := data.FindGuildcardEntries(s.db, req.AccountId)
	if err != nil {
		return nil, fmt.Errorf("error retrieving guildcard entries for account %d: %w", req.AccountId, err)
	}
	resp := &GetGuildcardEntriesResponse{}
	for _, entry := range entries {
		resp.Entries = append(resp.Entries, guildcardEntryToProto(&entry))
	}
	return resp, nil
}

func (s *service) GetPlayerOptions(ctx context.Context, req *GetPlayerOptionsRequest) (*GetPlayerOptionsResponse, error) {
	s.logger.Debug("GetPlayerOptions")

	playerOptions, err := data.FindPlayerOptions(s.db, req.AccountId)
	if err != nil {
		return nil, fmt.Errorf("error retrieving player options for account %d: %w", req.AccountId, err)
	}

	resp := &GetPlayerOptionsResponse{
		Exists:        false,
		PlayerOptions: &proto.PlayerOptions{},
	}
	if playerOptions != nil {
		resp.Exists = true
		resp.PlayerOptions = playerOptionsToProto(playerOptions)
	}
	return resp, nil
}

func (s *service) UpsertPlayerOptions(ctx context.Context, req *UpsertPlayerOptionsRequest) (*emptypb.Empty, error) {
	s.logger.Debug("UpsertPlayerOptions")

	playerOptions := playerOptionsFromProto(req.PlayerOptions)
	playerOptions.Account = &data.Account{
		ID: req.AccountId,
	}
	if err := data.CreatePlayerOptions(s.db, playerOptions); err != nil {
		return nil, fmt.Errorf("error creating player options: %v", err)
	}
	return &emptypb.Empty{}, nil
}
