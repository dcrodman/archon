package main

import (
	"time"
)

// Account contains the login information specific to each registered user.
type Account struct {
	Username         string    `json:"username"`
	Password         string    `json:"password"`
	Email            string    `json:"email"`
	RegistrationDate time.Time `json:"registration_date"`
	Guildcard        int       `json:"guildcard"`
	GM               bool      `json:"is_gm"`
	Banned           bool      `json:"banned"`
	Active           bool      `json:"active"`
	TeamID           int       `json:"team_id"`
	PrivilegeLevel   byte      `json:"privilege_level"`
}

type PlayerOptions struct {
	Guildcard uint32 `json:"guildcard"`
	KeyConfig []byte `json:"key_config"`
}

// Character is an instance of a character in one of the slots for an account.
type Character struct {
	Guildcard         int     `json:"guildcard"`
	GuildcardStr      []byte  `json:"guildcard_str"`
	Slot              uint32  `json:"slot"`
	Experience        uint32  `json:"experience"`
	Level             uint32  `json:"level"`
	NameColor         uint32  `json:"name_color"`
	Model             byte    `json:"model"`
	NameColorChecksum uint32  `json:"name_color_checksum"`
	SectionID         byte    `json:"section_id"`
	Class             byte    `json:"class"`
	V2Flags           byte    `json:"v2_flags"`
	Version           byte    `json:"version"`
	V1Flags           uint32  `json:"v1_flags"`
	Costume           uint16  `json:"costume"`
	Skin              uint16  `json:"skin"`
	Face              uint16  `json:"face"`
	Head              uint16  `json:"head"`
	Hair              uint16  `json:"hair"`
	HairRed           uint16  `json:"hair_red"`
	HairGreen         uint16  `json:"heair_green"`
	HairBlue          uint16  `json:"hair_blue"`
	ProportionX       float32 `json:"proportion_x"`
	ProportionY       float32 `json:"proportion_y"`
	Name              []uint8 `json:"name"`
	Playtime          uint32  `json:"playtime"`
	ATP               uint16  `json:"atp"`
	MST               uint16  `json:"mst"`
	EVP               uint16  `json:"evp"`
	HP                uint16  `json:"hp"`
	DFP               uint16  `json:"dfp"`
	ATA               uint16  `json:"ata"`
	LCK               uint16  `json:"lck"`
	Meseta            uint32  `json:"meseta"`
}

type GuildcardEntry struct {
	Guildcard       int      `json:"guildcard"`
	FriendGuildcard int      `json:"friendGuildcard"`
	Name            []uint16 `json:"name"`
	TeamName        []uint16 `json:"team_name"`
	Description     []uint16 `json:"description"`
	Language        byte     `json:"language"`
	SectionID       byte     `json:"section_id"`
	Class           byte     `json:"class"`
	Comment         []uint16 `json:"comment"`
}
