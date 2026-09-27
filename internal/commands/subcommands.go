package commands

// BSubcommandHeader is the header structure common to all broadcast subcommands.
type BSubcommandHeader struct {
	Type uint8
	Size uint8
}

type BSubcommandExtHeader struct {
	Type    uint8
	Size    uint8
	ExtSize uint32
}

type BSubcommandClientHeader struct {
	Type     uint8
	Size     uint8
	ClientID uint16
}

// Sent by the client when a player has changed floors.
const SubcommandChangeFloorType = 0x1F

type SubcommandChangeFloor struct {
	Header BSubcommandClientHeader
	Floor  uint32
}

const (
	// BSubcommandClientHeader only. Sets a player invisible when changing floors.
	SubcommandSetPlayerInisible = 0x22
	// BSubcommandClientHeader only. Sets a player visible when changing floors.
	SubcommandSetPlayerVisible = 0x23
)

const (
	// Player has stopped moving.
	SubcommandStopPositionType = 0x3E
	// Player updates their position.
	SubcommandSetPositionType = 0x3F
)

type SubcommandPositionChanged struct {
	Header BSubcommandClientHeader
	Unused uint16
	Angle  uint16
	Floor  uint16
	Room   uint16
	PosX   float32
	PosY   float32
	PosZ   float32
}

// Walk to a position.
const SubcommandWalkType = 0x40

type SubcommandWalk struct {
	Header BSubcommandClientHeader
	PosX   float32
	PosZ   float32
	Flags  uint32
}

// Run to a position.
const SubcommandRunType = 0x42

type SubcommandRun struct {
	Header BSubcommandClientHeader
	PosX   float32
	PosZ   float32
}

// Various sync commands sent between a Game leader and all other players.
const (
	SubcommandSyncEnemyState  = 0x6B
	SubcommandSyncObjectState = 0x6C
	SubcommandSyncItemState   = 0x6D
	SubcommandSyncFlagState   = 0x6E
)

type SubcommandSyncState struct {
	Header           BSubcommandHeader
	DecompressedSize uint32
	CompressedSize   uint32
}

// Set quest flags when loading into the game.
const SubcommandSetQuestFlags = 0x6F

// Sync a player's display and inventory as they're joining a game.
const SubcommandSyncPlayerDataType = 0x70

type SubcommandSyncPlayerData struct {
	Header BSubcommandClientHeader
	Unused uint16
	// Base is a lazily compressed representation of a complicated-ish series of structs
	// that can be expanded if/when we need them.
	Base     [200]uint8
	Visual   CharacterVisual
	Stats    CharacterStats
	NumItems uint32
	Items    [30]CharacterInventoryItem
	Floor    uint32
	Unused2  [12]uint8
}

// Resume play after the waiting screen when a new player joins a game.
const SubcommandResume = 0x72
