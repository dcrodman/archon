package data

import (
	"errors"

	"gorm.io/gorm"
)

type PlayerOptions struct {
	ID uint64 `gorm:"primaryKey"`

	Account   *Account
	AccountID int

	KeyConfig []byte
}

// FindPlayerOptions returns all of hte PlayerOptions associated with an Account.
func FindPlayerOptions(db *gorm.DB, accountId uint64) (*PlayerOptions, error) {
	var playerOptions PlayerOptions
	err := db.Where("account_id = ?", accountId).First(&playerOptions).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &playerOptions, nil
}

func CreatePlayerOptions(db *gorm.DB, po *PlayerOptions) error {
	return db.Create(po).Error
}

func UpdatePlayerOptions(db *gorm.DB, po *PlayerOptions) error {
	return db.Updates(&po).Error
}
