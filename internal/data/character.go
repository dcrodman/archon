package data

import (
	"errors"

	"gorm.io/gorm"
)

// Character is an instance of a character in one of the slots for an account.
type Character struct {
	gorm.Model

	Account   *Account
	AccountID int

	Guildcard         int
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
}

// FindCharacter returns the Character associated with the account in
// the given slot or nil if none exists.
func FindCharacter(account *Account, slot int) (*Character, error) {
	var character Character
	err := db.Where("slot = ? AND account_id = ?", slot, &account.ID).First(&character).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &character, nil
}

// CreateCharacter persists a Character to the database.
func CreateCharacter(character *Character) error {
	return db.Create(&character).Error
}

// UpdateCharacter updates an existing Character row with the contents of character.
func UpdateCharacter(character *Character) error {
	return db.Updates(&character).Error
}

// DeleteCharacter soft-deletes a character record from the database.
func DeleteCharacter(character *Character) error {
	return db.Delete(character).Error
}

// PermanentlyDeleteCharacter permanently deletes a character record from the database.
func PermanentlyDeleteCharacter(character *Character) error {
	return db.Unscoped().Delete(character).Error
}
