package internal

import (
	"fmt"

	"github.com/dcrodman/archon/internal/prs"
)

const (
	NumCharacterClasses = 12
	NumCharacterDeltas  = 200
)

// LevelTable contains the starting stats for each character as well as the increases
// to those stats at each level. BaseStats and Deltas are both indexable by character
// classes (see the CharClass constants) with Deltas further indexed by level.
var LevelTable = struct {
	// Starting stats for any new character.
	BaseStats [NumCharacterClasses]Stats
	// It's not clear to me what this is for and sylverant marks it as unknown, so
	// the contents of this block are populated but not used by archon.
	Unknown [48]uint8
	// Stat increases for each level for each character.
	Deltas [NumCharacterClasses][200]LevelDeltas
	// Not clear what this is either, so treat the same as Unknown.
	Unknown2 [168]uint8
}{}

const statsStructSize = 14

type Stats struct {
	ATP uint16
	MST uint16
	EVP uint16
	HP  uint16
	DFP uint16
	ATA uint16
	LCK uint16
}

const deltasStructSize = 12

type LevelDeltas struct {
	ATP uint8
	MST uint8
	EVP uint8
	HP  uint8
	DFP uint8
	ATA uint8
	LCK uint8
	TP  uint8
	EXP uint32
}

func InitLevelTable(data []byte) {
	decompressedData, err := prs.Decompress(data)
	if err != nil {
		panic(fmt.Sprintf("error decompressing PlyLevelTbl.prs: %v", err))
	}

	offset := 0

	// The base stats are the first 168 bytes of the file.
	for i := range NumCharacterClasses {
		UnmarshalStruct(decompressedData[i*statsStructSize:], &LevelTable.BaseStats[i])
	}
	offset += statsStructSize * NumCharacterClasses

	// Copy the mystery bytes for posterity, but mostly we care about advancing the offset.
	offset += copy(LevelTable.Unknown[:], decompressedData[offset:])

	// The rest of the file contains the per-class stat and experience deltas for each level.
	for class := range NumCharacterClasses {
		for delta := range NumCharacterDeltas {
			UnmarshalStruct(
				decompressedData[offset:offset+deltasStructSize],
				&LevelTable.Deltas[class][delta],
			)
			offset += deltasStructSize
		}
	}

	// Copy the last chunk of mystery data in for posterity.
	copy(LevelTable.Unknown2[:], decompressedData[offset:])
}
