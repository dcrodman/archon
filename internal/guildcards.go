package internal

// GuildcardData is the per-player guildcard data chunk.
type GuildcardData struct {
	Unknown  [0x114]uint8
	Blocked  [0x1DE8]uint8 //This should be a struct once implemented
	Unknown2 [0x78]uint8
	Entries  [104]GuildcardDataEntry
	Unknown3 [0x1BC]uint8
}

// GuildcardDataEntry is the per-player friend guildcard entries.
type GuildcardDataEntry struct {
	Guildcard   uint32
	Name        [48]byte
	TeamName    [32]byte
	Description [176]byte
	Reserved    uint8
	Language    uint8
	SectionID   uint8
	CharClass   uint8
	padding     uint32
	Comment     [176]byte
}
