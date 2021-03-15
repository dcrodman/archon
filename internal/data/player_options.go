package data

import (
	"errors"
	"gorm.io/gorm"
)

type PlayerOptions struct {
	gorm.Model

	Account   *Account
	AccountID int

	KeyConfig []byte
}

// FindPlayerOptions returns all of hte PlayerOptions associated with an Account.
func FindPlayerOptions(account *Account) (*PlayerOptions, error) {
	var playerOptions PlayerOptions
	err := db.Where("account_id = ?", &account.ID).First(&playerOptions).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &playerOptions, nil
}

func CreatePlayerOptions(po *PlayerOptions) error {
	return db.Create(po).Error
}

func UpdatePlayerOptions(po *PlayerOptions) error {
	return db.Updates(&po).Error
}
