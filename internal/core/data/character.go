package data

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Character is an instance of a character in one of the slots for an account.
type Character struct {
	ID uint64 `gorm:"primaryKey"`

	Account   *Account
	AccountID uint64

	Guildcard         uint64
	GuildcardStr      []byte
	Slot              uint32
	Experience        uint32
	Level             uint32
	NameColor         uint32
	ModelType         byte
	NameColorChecksum uint32
	SectionID         byte
	Class             byte
	V2Flags           byte
	Version           byte
	V1Flags           uint32
	Costume           uint16
	Skin              uint16
	Face              uint16
	Head              uint16
	Hair              uint16
	HairRed           uint16
	HairGreen         uint16
	HairBlue          uint16
	ProportionX       float32
	ProportionY       float32
	ReadableName      string
	Name              []byte
	Playtime          uint32
	ATP               uint16
	MST               uint16
	EVP               uint16
	HP                uint16
	DFP               uint16
	ATA               uint16
	LCK               uint16
	Meseta            uint32

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt
}

// FindCharacter returns the Character associated with the account in
// the given slot or nil if none exists.
func FindCharacter(db *gorm.DB, accountID uint, slot uint32) (*Character, error) {
	var character Character
	err := db.Where("slot = ? AND account_id = ?", slot, &accountID).First(&character).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &character, nil
}

// CreateCharacter persists a Character to the database.
func CreateCharacter(db *gorm.DB, character *Character) error {
	return db.Create(&character).Error
}

// UpsertCharacter updates an existing Character row with the contents of character.
func UpsertCharacter(db *gorm.DB, character *Character) error {
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "slot"}},
		UpdateAll: true,
	}).Create(&character).Error
}

// DeleteCharacter soft-deletes a character record from the database.
func DeleteCharacter(db *gorm.DB, accountID uint, slot uint32) error {
	character, err := FindCharacter(db, accountID, slot)
	if err != nil {
		return err
	} else if character != nil {
		return db.Delete(character).Error
	}
	return nil
}

// PermanentlyDeleteCharacter permanently deletes a character record from the database.
func PermanentlyDeleteCharacter(db *gorm.DB, character *Character) error {
	return db.Unscoped().Delete(character).Error
}
