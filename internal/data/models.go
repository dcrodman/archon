package data

import (
	"time"

	"gorm.io/gorm"
)

// Account contains the login information specific to each registered user.
type Account struct {
	ID               uint64 `gorm:"primaryKey"`
	Username         string `gorm:"unique; not null"`
	Password         string `gorm:"not null"`
	Email            string `gorm:"unique"`
	RegistrationDate time.Time
	Guildcard        uint64 `gorm:"AUTO_INCREMENT"`
	GM               bool   `gorm:"default:false"`
	Banned           bool   `gorm:"default:false"`
	Active           bool   `gorm:"default:true"`
	TeamID           int
	PrivilegeLevel   byte

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt
}

type PlayerOptions struct {
	ID uint64 `gorm:"primaryKey"`

	Account   *Account
	AccountID uint64

	KeyConfig []byte
}

// Character is an instance of a character in one of the slots for an account.
type Character struct {
	ID uint64 `gorm:"primaryKey"`

	Account   *Account
	AccountID uint64 `gorm:"uniqueIndex:character_account_slot"`

	Guildcard         uint64
	GuildcardStr      []byte
	Slot              uint32 `gorm:"uniqueIndex:character_account_slot"`
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
	HPMaterialsUsed   byte
	TPMaterialsUsed   byte

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
	Name            []byte
	TeamName        []byte
	Description     []byte
	Language        byte
	SectionID       byte
	Class           byte
	Comment         []byte
}

// Ship contains the connection information for a Ship server.
type Ship struct {
	IP   string
	Port int
	Name string
}
