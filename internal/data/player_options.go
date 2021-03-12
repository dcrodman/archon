package data

import (
	"errors"
	"gorm.io/gorm"
)

type PlayerOptions struct {
	gorm.Model

	AccountID int

	KeyConfig []byte
}

// FindPlayerOptions returns all of hte PlayerOptions associated with an Account.
func FindPlayerOptions(account *Account) (*PlayerOptions, error) {
	var playerOptions PlayerOptions
	err := db.Model(&account).Association("PlayerOptions").Find(&playerOptions)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &playerOptions, nil
}

// UpdatePlayerOptions updates the PlayerOptions row with the contents in po.
func CreatePlayerOptions(account *Account, po *PlayerOptions) error {
	return db.Model(account).Association("PlayerOptions").Replace(&po)
}
