package shipgate

import (
	"github.com/dcrodman/archon/internal/core/data"
	"github.com/dcrodman/archon/internal/core/proto"
)

func characterToProto(character *data.Character) *proto.Character {
	protoCharacter := &proto.Character{
		Id:                character.ID,
		Guildcard:         uint64(character.Guildcard),
		GuildcardStr:      character.GuildcardStr,
		Slot:              character.Slot,
		Experience:        character.Experience,
		Level:             character.Level,
		NameColor:         character.NameColor,
		ModelType:         int32(character.ModelType),
		NameColorChecksum: character.NameColorChecksum,
		SectionId:         int32(character.SectionID),
		Class:             int32(character.Class),
		V2Flags:           int32(character.V2Flags),
		Version:           int32(character.Version),
		V1Flags:           character.V1Flags,
		Costume:           uint32(character.Costume),
		Skin:              uint32(character.Skin),
		Face:              uint32(character.Face),
		Head:              uint32(character.Head),
		Hair:              uint32(character.Hair),
		HairRed:           uint32(character.HairRed),
		HairGreen:         uint32(character.HairGreen),
		HairBlue:          uint32(character.HairBlue),
		ProportionX:       character.ProportionX,
		ProportionY:       character.ProportionY,
		Name:              character.Name,
		ReadableName:      character.ReadableName,
		Playtime:          character.Playtime,
		Atp:               uint32(character.ATP),
		Mst:               uint32(character.MST),
		Evp:               uint32(character.EVP),
		Hp:                uint32(character.HP),
		Dfp:               uint32(character.DFP),
		Ata:               uint32(character.ATA),
		Lck:               uint32(character.LCK),
		Meseta:            character.Meseta,
	}
	return protoCharacter
}

func characterFromProto(character *proto.Character) *data.Character {
	dbCharacter := &data.Character{
		Guildcard:         character.Guildcard,
		GuildcardStr:      character.GuildcardStr,
		Slot:              character.Slot,
		Experience:        character.Experience,
		Level:             character.Level,
		NameColor:         character.NameColor,
		ModelType:         byte(character.ModelType),
		NameColorChecksum: character.NameColorChecksum,
		SectionID:         byte(character.SectionId),
		Class:             byte(character.Class),
		V2Flags:           byte(character.V2Flags),
		Version:           byte(character.Version),
		V1Flags:           character.V1Flags,
		Costume:           uint16(character.Costume),
		Skin:              uint16(character.Skin),
		Face:              uint16(character.Face),
		Head:              uint16(character.Head),
		Hair:              uint16(character.Hair),
		HairRed:           uint16(character.HairRed),
		HairGreen:         uint16(character.HairGreen),
		HairBlue:          uint16(character.HairBlue),
		ProportionX:       character.ProportionX,
		ProportionY:       character.ProportionY,
		ReadableName:      character.ReadableName,
		Name:              character.Name,
		Playtime:          character.Playtime,
		ATP:               uint16(character.Atp),
		MST:               uint16(character.Mst),
		EVP:               uint16(character.Evp),
		HP:                uint16(character.Hp),
		DFP:               uint16(character.Dfp),
		ATA:               uint16(character.Ata),
		LCK:               uint16(character.Lck),
		Meseta:            character.Meseta,
	}
	return dbCharacter
}

func guildcardEntryToProto(gcEntry *data.GuildcardEntry) *proto.GuildcardEntry {
	return &proto.GuildcardEntry{
		Guildcard:       gcEntry.Guildcard,
		FriendGuildcard: uint64(gcEntry.FriendGuildcard),
		Name:            gcEntry.Name,
		TeamName:        gcEntry.TeamName,
		Description:     gcEntry.Description,
		Language:        uint32(gcEntry.Language),
		SectionId:       uint32(gcEntry.SectionID),
		Class:           uint32(gcEntry.Class),
		Comment:         gcEntry.Comment,
	}
}

func playerOptionsToProto(playerOptions *data.PlayerOptions) *proto.PlayerOptions {
	return &proto.PlayerOptions{
		Id:        uint32(playerOptions.ID),
		KeyConfig: playerOptions.KeyConfig,
	}
}

func playerOptionsFromProto(playerOptions *proto.PlayerOptions) *data.PlayerOptions {
	return &data.PlayerOptions{
		KeyConfig: playerOptions.KeyConfig,
	}
}
