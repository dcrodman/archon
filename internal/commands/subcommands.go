package commands

// BroadcastSubcommandHeader is the header structure common to all subcommands.
type BroadcastSubcommandHeader struct {
	Type uint8
	Size uint8
}

// Sent by the client when the player moves.
const SubcommandPositionChangedType = 0x3E

type SubcommandPositionChanged struct {
	Header   BroadcastSubcommandHeader
	ClientID uint16
	Unused   uint16
	Angle    uint16
	Floor    uint16
	Room     uint16
	PosX     float32
	PosY     float32
	PosZ     float32
}
