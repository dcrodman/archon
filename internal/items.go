package internal

import (
	"encoding/binary"

	"github.com/dcrodman/archon/internal/commands"
)

// Amount of meseta new characters are given when created.
const StartingMeseta = 300

// NewItemData creates a new ItemData representing the item literal defined by first
// and second. Since the full item data is a total of 16 bytes (12 bytes for all items
// and an additional 4 for mags and meseta), first is copied as-is into the high
// 8 bytes of item data along with the high 32 bits of second, with the remainder
// of second comprising that additional 4 bytes (all BE order).
func NewItemData(first, second uint64) commands.ItemData {
	var item commands.ItemData
	binary.BigEndian.PutUint64(item.Data[:8], first)
	binary.BigEndian.PutUint32(item.Data[8:12], uint32(second>>32))
	binary.BigEndian.PutUint32(item.Data2[:], uint32(second))
	return item
}

// DefaultInventory is the default item set for all characters, regardless of class.
// It consists of monofluids, monomates, a frame, and a mag.
var DefaultInventory = []commands.CharacterInventoryItem{
	{Item: NewItemData(0x0301000000040000, 0)},
	{Item: NewItemData(0x0300000000040000, 0)},
	{Item: NewItemData(0x0101000000000000, 0), Flags: commands.ItemFlagEquipped},
	{Item: NewItemData(0x02000500F4010000, 0x0000000028000012), Flags: commands.ItemFlagEquipped},
}

// DefaultWeaponsByClass are the class specific starting weapons, in canonical order of class.
var DefaultWeaponsByClass = [][]commands.CharacterInventoryItem{
	{{Item: NewItemData(0x0001000000000000, 0), Flags: commands.ItemFlagEquipped}},
	{{Item: NewItemData(0x0001000000000000, 0), Flags: commands.ItemFlagEquipped}},
	{{Item: NewItemData(0x0001000000000000, 0), Flags: commands.ItemFlagEquipped}},
	{{Item: NewItemData(0x0006000000000000, 0), Flags: commands.ItemFlagEquipped}},
	{{Item: NewItemData(0x0006000000000000, 0), Flags: commands.ItemFlagEquipped}},
	{{Item: NewItemData(0x0006000000000000, 0), Flags: commands.ItemFlagEquipped}},
	{{Item: NewItemData(0x000A000000000000, 0), Flags: commands.ItemFlagEquipped}},
	{{Item: NewItemData(0x000A000000000000, 0), Flags: commands.ItemFlagEquipped}},
	{{Item: NewItemData(0x000A000000000000, 0), Flags: commands.ItemFlagEquipped}},
	{{Item: NewItemData(0x0001000000000000, 0), Flags: commands.ItemFlagEquipped}},
	{{Item: NewItemData(0x000A000000000000, 0), Flags: commands.ItemFlagEquipped}},
	{{Item: NewItemData(0x0006000000000000, 0), Flags: commands.ItemFlagEquipped}},
}
