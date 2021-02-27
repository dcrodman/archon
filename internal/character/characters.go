package character

// CharClass is an enumeration of the possible character classes.
type CharClass uint8

const (
	// Possible character classes as defined by the game.
	Humar CharClass = iota
	Hunewearl
	Hucast
	Ramar
	Racast
	Racaseal
	Fomarl
	Fonewm
	Fonewearl
	Hucaseal
	Fomar
	Ramarl
)

// Common intermediate representation of a Character as it gets passed around
// various servers and/or stored.
type Summary struct {
	Experience     uint32
	Level          uint32
	GuildcardStr   [16]byte
	Unknown        [2]uint32
	NameColor      uint32
	Model          byte
	Padding        [15]byte
	NameColorChksm uint32
	SectionID      byte
	Class          byte
	V2Flags        byte
	Version        byte
	V1Flags        uint32
	Costume        uint16
	Skin           uint16
	Face           uint16
	Head           uint16
	Hair           uint16
	HairRed        uint16
	HairGreen      uint16
	HairBlue       uint16
	PropX          float32
	PropY          float32
	// In reality this is [16]uint16 but []uint8 is more convenient to work with.
	Name     [32]uint8
	Playtime uint32
}
