package packets

const (
	LobbyListType = 0x83
	BlockListType = 0x07
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
