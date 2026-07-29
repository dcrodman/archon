package shipgate

import (
	"context"
	"fmt"

	"github.com/dcrodman/archon/internal/core/data"
	"github.com/dcrodman/archon/internal/core/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

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
	if err := data.UpsertCharacter(s.db, character); err != nil {
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
