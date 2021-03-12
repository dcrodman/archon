package data

import (
	"errors"
	"gorm.io/gorm"
)

type GuildcardEntry struct {
	gorm.Model

	AccountID int

	Guildcard       int
	FriendGuildcard int
	Name            []byte
	TeamName        []byte
	Description     []byte
	Language        byte
	SectionID       byte
	Class           byte
	Comment         []byte
}

// FindGuildcardEntries returns all the GuildcardEntry rows associated with an Account.
func FindGuildcardEntries(account *Account) ([]GuildcardEntry, error) {
	var guildcardEntries []GuildcardEntry
	err := db.Model(&account).Association("GuildcardEntry").Find(&guildcardEntries)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return guildcardEntries, nil
}
