package data

import (
	"errors"

	"gorm.io/gorm"
)

type GuildcardEntry struct {
	ID uint64 `gorm:"primaryKey"`

	Account   *Account
	AccountID int

	Guildcard       uint64
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
func FindGuildcardEntries(db *gorm.DB, accountId uint64) ([]GuildcardEntry, error) {
	var guildcardEntries []GuildcardEntry
	err := db.Where("account_id = ?", accountId).Find(&guildcardEntries).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return guildcardEntries, nil
}
