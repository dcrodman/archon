package data

import (
	"errors"

	"gorm.io/gorm"
)

type GuildcardEntry struct {
	gorm.Model

	Account   *Account
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
	err := db.Where("account_id = ?", &account.ID).Find(&guildcardEntries).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return guildcardEntries, nil
}
