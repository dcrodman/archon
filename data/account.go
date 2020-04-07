package data

import (
	"github.com/jinzhu/gorm"
	"time"
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

// FindAccount searches for an account with the specified username, returning the
// *Account instance if found or nil of there is no match.
func FindAccount(username string) (*Account, error) {
	var account Account
	err := db.Where("username = ?", username).Find(&account).Error

	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
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
