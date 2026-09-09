package internal

import (
	"encoding/binary"
	"fmt"

	"github.com/dcrodman/archon/internal/commands"
)

// Amount of meseta new characters are given when created.
const StartingMeseta = 300

// NewItemData creates a new ItemData representing the item literal defined by first
// and second. Since the full item data is a total of 16 bytes (12 bytes for all items
// and an additional 4 for mags and meseta), first is copied as-is into the high
// 64 bits of item data along with the high 32 bits of second, with the remaining 32
// bits of second copied to the additional 4 bytes (all BE order).
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
}

var DefaultMag = commands.CharacterInventoryItem{
	Item: NewItemData(0x02000500F4010000, 0x0000000028000012), Flags: commands.ItemFlagEquipped,
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

type ItemPTEntry struct {
	WeaponRatio        [12]int8
	WeaponMinRank      [12]int8
	WeapinUPGFloor     [12]int8
	PowerPattern       [9][4]int8
	PercentPattern     [23][6]uint16
	AreaPattern        [3][10]int8
	PercentAttachment  [6][10]int8
	ElementRanking     [10]int8
	ElementProbability [10]int8
	ArmorRanking       [5]int8
	SlotRanking        [5]int8
	UnitLevel          [10]int8
	ToolFrequency      [28][10]uint16
	TechFrequency      [19][10]uint8
	TechLevels         [19][20]int8
	EnemyDAR           [100]int8
	EnemyMeseta        [100][2]uint16
	EnemyDrop          [100]int8
	BoxMeseta          [10][2]uint16
	BoxDrop            [7][10]uint8
	Padding            uint16
	Pointers           [18]uint32
	ArmorLevel         int32
}

var (
	// Keyed by [episode][difficulty][section ID].
	ItemTables          [][][]ItemPTEntry
	ChallengeItemTables [][][]ItemPTEntry
)

// InitItemPT loads the item drop tables.
func InitItemPT(data []byte) {
	type ptEntry struct {
		filename string
		offset   uint32
		size     uint32
	}

	// Read the headeers so that we know what the files are and where they're stored.
	entries := make(map[string]ptEntry)
	for i := 0; ; i += 48 {
		if data[i] == 0 {
			break
		}
		filename := string(StripPadding(data[i : i+32]))
		entries[filename] = ptEntry{
			filename: filename,
			offset:   binary.BigEndian.Uint32(data[i+32:i+36]) * 2048,
			size:     binary.BigEndian.Uint32(data[i+36 : i+40]),
		}
	}
	// 2 episodes * 4 difficulties * 10 section IDs = 160.
	if len(entries) != 160 {
		panic(fmt.Sprintf("expected 160 entries but found %v", len(entries)))
	}

	var (
		modes        = []string{"", "c"}
		episodes     = []string{"", "l"}
		difficulties = []string{"n", "h", "v", "u"}
		nSectionIDs  = 10
	)
	ItemTables = make([][][]ItemPTEntry, len(episodes))
	ChallengeItemTables = make([][][]ItemPTEntry, len(episodes))

	// Step through the combinations of item files we need and read the
	// contents of each file into the corresponding entry in the table.
	for episode := range episodes {
		ItemTables[episode] = make([][]ItemPTEntry, len(difficulties))
		ChallengeItemTables[episode] = make([][]ItemPTEntry, len(difficulties))

		for difficulty := range difficulties {
			ItemTables[episode][difficulty] = make([]ItemPTEntry, nSectionIDs)
			ChallengeItemTables[episode][difficulty] = make([]ItemPTEntry, nSectionIDs)

			for sectionID := range nSectionIDs {
				for _, mode := range modes {
					filename := fmt.Sprintf(
						"ItemPT%s%s%s%d.rel",
						mode,
						episodes[episode],
						difficulties[difficulty],
						sectionID,
					)
					entry := entries[filename]
					entryData := data[entry.offset : entry.offset+entry.size]

					if mode == "c" {
						UnmarshalStructWithOrder(
							entryData,
							&ChallengeItemTables[episode][difficulty][sectionID],
							binary.BigEndian,
						)
					} else {
						UnmarshalStructWithOrder(
							entryData,
							&ItemTables[episode][difficulty][sectionID],
							binary.BigEndian,
						)
					}
				}
			}
		}
	}
}
