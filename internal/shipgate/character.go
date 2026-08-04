package shipgate

import (
	"errors"

	"github.com/dcrodman/archon/internal/data"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// findCharacter returns the Character associated with the account in
// the given slot or nil if none exists.
func findCharacter(db *gorm.DB, accountID uint64, slot uint32) (*data.Character, error) {
	var character data.Character
	err := db.
		Where("slot = ? AND account_id = ?", slot, &accountID).
		Preload("Account").
		First(&character).
		Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &character, nil
}

// upsertCharacter updates an existing Character row with the contents of character.
func upsertCharacter(db *gorm.DB, character *data.Character) error {
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "account_id"}, {Name: "slot"}},
		UpdateAll: true,
	}).Create(&character).Error
}

// deleteCharacter soft-deletes a character record from the database.
func deleteCharacter(db *gorm.DB, accountID uint64, slot uint32) error {
	character, err := findCharacter(db, accountID, slot)
	if err != nil {
		return err
	} else if character != nil {
		return db.Delete(character).Error
	}
	return nil
}

// permanentlyDeleteCharacter permanently deletes a character record from the database.
func permanentlyDeleteCharacter(db *gorm.DB, character *data.Character) error {
	return db.Unscoped().Delete(character).Error
}
