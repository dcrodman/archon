package internal

import (
	"encoding/binary"
	"fmt"
	"reflect"

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

type ItemTableEntry struct {
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

type RareTableEntry struct {
	// These files typically contain either 101 or 112 entries depending on whether or not
	// the files include episode 4 rates. Since it's assumed pretty much everywhere that
	// Episode 4 is available, we use the latter number.
	MonsterRares [112]RareDropEntry
	BoxRares     [30]RareDropEntry
}

type RareDropEntry struct {
	Probability uint32
	ItemData    [12]uint8
	Area        uint32 // Only used for box drops, not enemies.
}

var (
	// Keyed by [episode][difficulty][section ID].
	ItemTables          = make([][][]ItemTableEntry, len(ptEpisodes))
	ChallengeItemTables = make([][][]ItemTableEntry, len(ptEpisodes))
	RareItemTables      = make([][][]RareTableEntry, len(ptEpisodes))

	// Constants used for parsing the contents of GSL archives.
	ptModes        = []string{"", "c"}
	ptEpisodes     = []string{"", "l", "bb"}
	ptDifficulties = []string{"n", "h", "v", "u"}
	ptSectionIDs   = 10
)

// InitItemPT populates the global item drop tables from ItemPT.gsl.
func InitItemPT(data []byte) {
	// Read the headeers so that we know what the files are and where they're stored.
	entries := readGSLHeaders(data)

	// 3 episodes * 4 difficulties * 10 section IDs + (ep1/2 challenge) = 200.
	if len(entries) != 200 {
		panic(fmt.Sprintf("expected 200 entries but found %v", len(entries)))
	}

	// Step through the combinations of item files we need and read the
	// contents of each file into the corresponding entry in the table.
	for episode := range ptEpisodes {
		ItemTables[episode] = make([][]ItemTableEntry, len(ptDifficulties))
		ChallengeItemTables[episode] = make([][]ItemTableEntry, len(ptDifficulties))

		for difficulty := range ptDifficulties {
			ItemTables[episode][difficulty] = make([]ItemTableEntry, ptSectionIDs)
			ChallengeItemTables[episode][difficulty] = make([]ItemTableEntry, ptSectionIDs)

			for sectionID := range ptSectionIDs {
				for _, mode := range ptModes {
					// Episode 4 does not have a challenge mode.
					if mode == "c" && ptEpisodes[episode] == "bb" {
						continue
					}

					filename := fmt.Sprintf("ItemPT%s%s%s%d.rel", mode, ptEpisodes[episode], ptDifficulties[difficulty], sectionID)
					entry := entries[filename]
					entryData := data[entry.offset : entry.offset+entry.size]

					if mode == "c" {
						UnmarshalStruct(entryData, &ChallengeItemTables[episode][difficulty][sectionID])
					} else {
						UnmarshalStruct(entryData, &ItemTables[episode][difficulty][sectionID])
					}
				}
			}
		}
	}
}

// InitItemPT populates the global item drop tables from ItemRT.gsl.
func InitItemRT(data []byte) {
	// Read the headeers so that we know what the files are and where they're stored.
	entries := readGSLHeaders(data)

	// 2 episodes * 4 difficulties * 10 section IDs + Ep1 challenge mode = 120.
	if len(entries) != 120 {
		panic(fmt.Sprintf("expected 120 entries but found %v", len(entries)))
	}

	// Temporary struct to hold the drop rates we read from the file since we need to
	// expand the probabilities and cannot just UnmarshalStruct straight in.
	type rtEntry struct {
		Probability uint8
		ItemCode    [3]uint8
	}

	// Step through the combinations of item files we need and read the
	// contents of each file into the corresponding entry in the table.
	for episode := range ptEpisodes {
		RareItemTables[episode] = make([][]RareTableEntry, len(ptDifficulties))

		for difficulty := range ptDifficulties {
			RareItemTables[episode][difficulty] = make([]RareTableEntry, ptSectionIDs)

			for sectionID := range ptSectionIDs {
				filename := fmt.Sprintf("ItemRT%s%s%d.rel", ptEpisodes[episode], ptDifficulties[difficulty], sectionID)
				entry := entries[filename]
				entryData := data[entry.offset : entry.offset+entry.size]
				offset := 0

				rt := &RareItemTables[episode][difficulty][sectionID]

				// First parse out the monster drop rates.
				for i := range len(rt.MonsterRares) {
					var tmpEntry rtEntry
					UnmarshalStruct(entryData[offset:], &tmpEntry)
					offset += int(reflect.TypeFor[rtEntry]().Size())

					rt.MonsterRares[i].Probability = expandRareItemProbability(tmpEntry.Probability)
					rt.MonsterRares[i].ItemData[0] = tmpEntry.ItemCode[0]
					rt.MonsterRares[i].ItemData[1] = tmpEntry.ItemCode[1]
					rt.MonsterRares[i].ItemData[2] = tmpEntry.ItemCode[2]
				}

				// Now parse out the areas corresponding to each box entry.
				areas := make([]uint8, len(rt.BoxRares))
				for i := range areas {
					areas[i] = entryData[offset]
					offset++
				}

				// Finally, the box rares.
				for i := range len(rt.BoxRares) {
					var tmpEntry rtEntry
					UnmarshalStruct(entryData[offset:], &tmpEntry)
					offset += int(reflect.TypeFor[rtEntry]().Size())

					rt.BoxRares[i].Probability = expandRareItemProbability(tmpEntry.Probability)
					rt.BoxRares[i].ItemData[0] = tmpEntry.ItemCode[0]
					rt.BoxRares[i].ItemData[1] = tmpEntry.ItemCode[1]
					rt.BoxRares[i].ItemData[2] = tmpEntry.ItemCode[2]
					rt.BoxRares[i].Area = uint32(areas[i])
				}
			}
		}
	}
}

// Expand the single byte probability value into a percentage out of the max uint32.
// Taken from a combination of how sylverant and newserv handle this.
// https://github.com/fuzziqersoftware/newserv/blob/master/src/RareItemSet.cc#L27
func expandRareItemProbability(pc uint8) uint32 {
	shift := max(0, (int8(pc)>>3)-4)
	return uint32(2<<shift) * uint32((pc&7)+7)
}

type gslHeaderEntry struct {
	filename string
	offset   uint32
	size     uint32
}

func readGSLHeaders(data []byte) map[string]gslHeaderEntry {
	entries := make(map[string]gslHeaderEntry)
	for i := 0; ; i += 48 {
		if data[i] == 0 {
			break
		}
		filename := string(StripPadding(data[i : i+32]))
		entries[filename] = gslHeaderEntry{
			filename: filename,
			offset:   binary.LittleEndian.Uint32(data[i+32:i+36]) * 2048,
			size:     binary.LittleEndian.Uint32(data[i+36 : i+40]),
		}
	}
	return entries
}
