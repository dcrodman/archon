package shipgate

import (
	"errors"

	"gorm.io/gorm"

	"github.com/dcrodman/archon/internal/data"
)

func findAccountByID(db *gorm.DB, id uint) (*data.Account, error) {
	var account data.Account
	err := db.First(&account, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &account, nil
}

// findAccountByUsername searches for an account with the specified username, returning the
// *data.Account instance if found or nil if there is no match.
func findAccountByUsername(db *gorm.DB, username string) (*data.Account, error) {
	var account data.Account
	err := db.Where("username = ?", username).First(&account).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &account, nil
}

// findUnscopedAccount searches for a potentially soft-deleted account with the
// specified username, returning the *data.Account instance if found or nil if
// there is no match.
func findUnscopedAccount(db *gorm.DB, username string) (*data.Account, error) {
	var account data.Account
	err := db.Unscoped().Where("username = ?", username).First(&account).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &account, nil
}

// createAccount persists the Account record to the database.
func createAccount(db *gorm.DB, account *data.Account) error {
	return db.Create(account).Error
}

// deleteAccount soft-deletes an Account record from the database.
func deleteAccount(db *gorm.DB, account *data.Account) error {
	return db.Delete(account).Error
}

// permanentlyDeleteAccount permanently deletes an Account record from the database.
func permanentlyDeleteAccount(db *gorm.DB, account *data.Account) error {
	return db.Unscoped().Delete(account).Error
}
