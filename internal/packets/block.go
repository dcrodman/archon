package packets

const (
	LobbyListType        = 0x83
	BlockListType        = 0x07
	FullCharacterType    = 0xE7
	FullCharacterEndType = 0x95
)

type LobbyListEntry struct {
	MenuID  uint32 // Always 0x01 0x00 0x1A 0x00
	LobbyID uint32
	Padding uint32
}

// LobbyList is the list of available lobbies in a block.
type LobbyList struct {
	Header  BBHeader
	Lobbies []LobbyListEntry
}

type InventoryItem struct {
	Data    [12]byte
	ItemID  uint32
	MagData uint32
}

type InventorySlot struct {
	InUse   uint8 // 0x01 for in use, 0xFF is unused
	Unknown [3]byte
	Flags   uint32
	Item    InventoryItem
}

type BankItem struct {
	Data      [12]byte
	ItemID    uint32
	MagData   [4]byte
	BankCount uint32
}

// FullCharacter is the full dataset for one character.
// TODO: Someday, figure out what more of these fields do.
type FullCharacter struct {
	Header                BBHeader
	NumInventoryItems     uint8
	HPMaterials           uint8
	TPMaterials           uint8
	Language              uint8
	Inventory             [30]InventorySlot
	ATP                   uint16
	MST                   uint16
	EVP                   uint16
	HP                    uint16
	DFP                   uint16
	ATA                   uint16
	LCK                   uint16
	Unknown               [30]byte
	Level                 uint16
	Unknown2              uint16
	Experience            uint32
	Meseta                uint32
	GuildcardStr          [10]byte
	Unknown3              [10]uint8
	NameColorBlue         uint8
	NameColorGreen        uint8
	NameColorRed          uint8
	NameColorTransparency uint8
	SkinID                uint16
	Unknown4              [18]byte
	SectionID             uint8
	Class                 uint8
	SkinFlag              uint8
	Unknown5              [5]byte
	Costume               uint16
	Skin                  uint16
	Face                  uint16
	Head                  uint16
	Hair                  uint16
	HairColorRed          uint16
	HairColorBlue         uint16
	HairColorGreen        uint16
	ProportionX           uint32
	ProportionY           uint32
	Name                  [24]byte
	PlayTime              uint32
	Unknown6              [4]byte
	KeyConfig             [232]uint8
	Techniques            [20]uint8
	Unknown7              [16]uint8
	Options               [4]uint8
	Reserved4             uint32
	QuestData             [512]uint8
	Reserved5             uint32
	BankUse               uint32
	BankMeseta            uint32
	BankInventory         [200]BankItem
	Guildcard             uint32
	Name2                 [24]uint8
	Unknown9              [56]byte
	GuildcardText         [176]uint8
	Reserved1             uint8
	Reserved2             uint8
	SectionID2            uint8
	Class2                uint8
	Unknown10             [4]uint8
	SymbolChats           [1248]uint8
	Shortcuts             [2624]uint8
	AutoReply             [344]uint8
	GCBoard               [172]uint8
	Unknown12             [200]uint8
	ChallengeData         [320]uint8
	TechConfig            [40]uint8
	Unknown13             [40]uint8
	QuestData2            [92]uint8
	Unknown14             [276]uint8
	KeyConfigGlobal       [364]uint8
	JoystickConfigGlobal  [56]uint8
	Guildcard2            uint32
	TeamID                uint32
	TeamInformation       [8]uint8
	PrivilegeLevel        uint16
	Reserved3             uint16
	TeamName              [28]uint8
	Unknown15             uint32
	TeamFlag              [2048]uint8
	TeamRewards           [8]uint8
}
