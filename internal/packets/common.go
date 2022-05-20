// Packets used by multiple server types.
package packets

const (
	PCHeaderSize = 0x04
	BBHeaderSize = 0x08
)

// Blueburst, PC, and Gamecube clients all use a 4 byte header to
// communicate with the patch server instead of the 8 byte one used
// by Blueburst for the other servers.
type PCHeader struct {
	Size uint16
	Type uint16
}

// Packet header for every packet sent between the server and BlueBurst clients.
type BBHeader struct {
	Size  uint16
	Type  uint16
	Flags uint32
}

// Error codes used by the 0xE6 security/auth response packet.
type BBLoginError uint32

const (
	BBLoginErrorNone = iota
	BBLoginErrorUnknown
	BBLoginErrorPassword
	BBLoginErrorPassword2 // Same as password
	BBLoginErrorMaintenance
	BBLoginErrorUserInUse
	BBLoginErrorBanned
	BBLoginErrorBanned2 // Same as banned
	BBLoginErrorUnregistered
	BBLoginErrorExpiredSub
	BBLoginErrorLocked
	BBLoginErrorPatch
	BBLoginErrorDisconnect
)

// Packet types common to multiple servers.
const (
	DisconnectType = 0x05
	RedirectType   = 0x19
	MenuSelectType = 0x10
)
