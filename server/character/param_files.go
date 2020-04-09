package character

import (
	"errors"
	"fmt"
	"github.com/dcrodman/archon/internal/prs"
	"github.com/dcrodman/archon/server/internal"
	"github.com/spf13/viper"
	"hash/crc32"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

const (
	NumCharacterClasses = 12
	// Amount of meseta new characters are given when created.
	StartingMeseta = 300
)

var (
	// Parameter files we're expecting. I still don't really know what they're
	// for yet, so emulating what I've seen others do.
	paramFiles = []string{
		"ItemMagEdit.prs",
		"ItemPMT.prs",
		"BattleParamEntry.dat",
		"BattleParamEntry_on.dat",
		"BattleParamEntry_lab.dat",
		"BattleParamEntry_lab_on.dat",
		"BattleParamEntry_ep4.dat",
		"BattleParamEntry_ep4_on.dat",
		"PlyLevelTbl.prs",
	}

	// Cached parameter data to avoid computing it every time.
	paramHeaderData []byte
	paramChunkData  map[int][]byte

	// Starting stats for any new character. The CharClass constants can be used
	// to index into this array to obtain the base stats for each class.
	BaseStats [NumCharacterClasses]stats

	paramInitLock sync.Once
)

// Per-character stats as stored in config files.
type stats struct {
	ATP uint16
	MST uint16
	EVP uint16
	HP  uint16
	DFP uint16
	ATA uint16
	LCK uint16
}

// Struct for caching the parameter chunk data and header so
// that the param files aren't re-read every time.
type parameterEntry struct {
	Size     uint32
	Checksum uint32
	Offset   uint32
	Filename [0x40]uint8
}

func initParameterData() {
	paramInitLock.Do(func() {
		paramFileDir := viper.GetString("character_server.parameters_dir")

		if err := loadParameterFiles(paramFileDir); err != nil {
			fmt.Println("failed to load parameter files:" + err.Error())
			os.Exit(1)
		}

		// Load the base stats for creating new characters.
		statsFile, _ := os.Open(filepath.Join(paramFileDir, "PlyLevelTbl.prs"))
		compressedStatsFile, err := ioutil.ReadAll(statsFile)
		if err != nil {
			fmt.Println("failed to load PlyLevelTbl.prs:" + err.Error())
			os.Exit(1)
		}

		decompressedStatsFile := make([]byte, prs.DecompressSize(compressedStatsFile))
		prs.Decompress(compressedStatsFile, decompressedStatsFile)

		// Base character class stats are stored sequentially, each 14 bytes long.
		for i := 0; i < NumCharacterClasses; i++ {
			internal.StructFromBytes(decompressedStatsFile[i*14:], &BaseStats[i])
		}
	})
}

// Load the PSOBB parameter files, build the parameter header,
// and init/cache the param file chunks for the EB packets.
func loadParameterFiles(paramFileDir string) error {
	fmt.Printf("loading parameters from %s...\n", paramFileDir)

	offset := 0
	var tmpChunkData []byte

	for _, paramFile := range paramFiles {
		data, err := ioutil.ReadFile(filepath.Join(paramFileDir, paramFile))
		if err != nil {
			return errors.New("Error reading parameter file: " + err.Error())
		}

		fileSize := len(data)
		offset += fileSize

		entry := &parameterEntry{
			Size:     uint32(fileSize),
			Checksum: crc32.ChecksumIEEE(data),
			Offset:   uint32(offset),
			Filename: [64]uint8{},
		}
		copy(entry.Filename[:], paramFile)

		// We don't care what the actual entries are for the packet, so just append
		// the bytes to save us having to do the conversion every time.
		bytes, _ := internal.BytesFromStruct(entry)
		paramHeaderData = append(paramHeaderData, bytes...)
		tmpChunkData = append(tmpChunkData, data...)

		if viper.GetBool("debug_mode") {
			fmt.Printf("%s (%v bytes, checksum: %v)\n", paramFile, fileSize, entry.Checksum)
		}
	}

	// Offset should at this point be the total size of the files
	// to send - break it all up into chunks for indexing.
	paramChunkData = make(map[int][]byte)
	numChunks := offset / MaxDataChunkSize
	for i := 0; i < numChunks; i++ {
		dataOff := i * MaxDataChunkSize
		paramChunkData[i] = tmpChunkData[dataOff : dataOff+MaxDataChunkSize]
		offset -= MaxDataChunkSize
	}

	// Add any remaining data.
	if offset > 0 {
		paramChunkData[numChunks] = tmpChunkData[numChunks*MaxDataChunkSize:]
	}
	return nil
}
