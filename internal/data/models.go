package data

import (
	"time"

	"gorm.io/gorm"
)

// Account contains the login information specific to each registered user.
type Account struct {
	ID uint64 `gorm:"primaryKey"`

	Username         string `gorm:"unique; not null"`
	Password         string `gorm:"not null"`
	Email            string `gorm:"unique"`
	RegistrationDate time.Time
	Guildcard        uint64 `gorm:"AUTO_INCREMENT"`
	GM               bool   `gorm:"default:false"`
	Banned           bool   `gorm:"default:false"`
	Active           bool   `gorm:"default:true"`
	TeamID           int
	PrivilegeLevel   uint8

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt
}

type PlayerOptions struct {
	ID uint64 `gorm:"primaryKey"`

	Account   *Account
	AccountID uint64

	// KeyConfig holds both the key and joystick config (0x1A4 bytes total); see SendOptions.
	KeyConfig []uint8
	// SystemSettings covers the client system/audio preferences section of SyncCharacter.
	SystemSettings []uint8
}

// Character is an instance of a character in one of the slots for an account.
type Character struct {
	ID uint64 `gorm:"primaryKey"`

	Account   *Account
	AccountID uint64 `gorm:"uniqueIndex:character_account_slot"`

	Slot uint32 `gorm:"uniqueIndex:character_account_slot"`

	// Data stores the CharacterData blob as received on the wire format.
	Data        []uint8
	DataVersion int

	// The following fields are extracted from Data as a convenience for querying
	// and inspecting the database.
	ReadableName string
	Level        uint32

	// Database timestamps.
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt
}

type GuildcardEntry struct {
	ID uint64 `gorm:"primaryKey"`

	Account   *Account
	AccountID uint64

	Guildcard       uint64
	FriendGuildcard int
	Name            []uint8
	TeamName        []uint8
	Description     []uint8
	Language        uint8
	SectionID       uint8
	Class           uint8
	Comment         []uint8
}
