package shipgate

import (
	"errors"

	"github.com/dcrodman/archon/internal/data"
	"gorm.io/gorm"
)

// findPlayerOptions returns all of hte PlayerOptions associated with an Account.
func findPlayerOptions(db *gorm.DB, accountId uint64) (*data.PlayerOptions, error) {
	var playerOptions data.PlayerOptions
	err := db.Where("account_id = ?", accountId).Preload("Account").First(&playerOptions).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &playerOptions, nil
}

func createPlayerOptions(db *gorm.DB, po *data.PlayerOptions) error {
	return db.Create(po).Error
}

func updatePlayerOptions(db *gorm.DB, po *data.PlayerOptions) error {
	return db.Updates(&po).Error
}
