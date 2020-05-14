package data

import "github.com/jinzhu/gorm"

type GuildcardEntry struct {
	gorm.Model

	AccountID int
	Account   *Account

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
	q := db.Model(&account).Related(&guildcardEntries)

	if q.Error != nil {
		return nil, q.Error
	}

	return guildcardEntries, nil
}
