package shipgate

import (
	"errors"

	"gorm.io/gorm"

	"github.com/dcrodman/archon/internal/data"
)

// findGuildcardEntries returns all the GuildcardEntry rows associated with an Account.
func findGuildcardEntries(db *gorm.DB, accountId uint64) ([]data.GuildcardEntry, error) {
	var guildcardEntries []data.GuildcardEntry
	err := db.Where("account_id = ?", accountId).Find(&guildcardEntries).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return guildcardEntries, nil
}
