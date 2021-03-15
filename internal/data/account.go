package data

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// Account contains the login information specific to each registered user.
type Account struct {
	gorm.Model

	Username         string `gorm:"unique; not null"`
	Password         string `gorm:"not null"`
	Email            string `gorm:"unique"`
	RegistrationDate time.Time
	Guildcard        int  `gorm:"AUTO_INCREMENT"`
	GM               bool `gorm:"default:false"`
	Banned           bool `gorm:"default:false"`
	Active           bool `gorm:"default:true"`
	TeamID           int
	PrivilegeLevel   byte
}

// FindCharacterInSlot returns the Character associated with the account in
// the given slot or nil if none exists.
func (a *Account) FindCharacterInSlot(slot int) (*Character, error) {
	var character Character
	err := db.Where("slot = ? AND account_id = ?", slot, &a.ID).First(&character).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &character, nil
}

// FindAccount searches for an account with the specified username, returning the
// *Account instance if found or nil if there is no match.
func FindAccount(username string) (*Account, error) {
	var account Account
	err := db.Where("username = ?", username).First(&account).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &account, nil
}

// FindUnscopedAccount searches for a potentially soft-deleted account with the
// specified username, returning the *Account instance if found or nil if
// there is no match.
func FindUnscopedAccount(username string) (*Account, error) {
	var account Account
	err := db.Unscoped().Where("username = ?", username).First(&account).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &account, nil
}

// CreateAccount persists the Account record to the database.
func CreateAccount(account *Account) error {
	return db.Create(account).Error
}

// DeleteAccount soft-deletes an Account record from the database.
func DeleteAccount(account *Account) error {
	return db.Delete(account).Error
}

// PermanentlyDeleteAccount permanently deletes an Account record from the database.
func PermanentlyDeleteAccount(account *Account) error {
	return db.Unscoped().Delete(account).Error
}
