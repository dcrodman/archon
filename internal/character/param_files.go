package character

import (
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/dcrodman/archon/internal/core/bytes"
	"github.com/dcrodman/archon/internal/core/prs"
	"github.com/sirupsen/logrus"
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

func initParameterData(logger *logrus.Logger, paramFileDir string) error {
	var initErr error

	paramInitLock.Do(func() {
		if err := loadParameterFiles(logger, paramFileDir); err != nil {
			initErr = fmt.Errorf("error loading parameter files:" + err.Error())
			return
		}

		// LoadConfig the base stats for creating new characters.
		statsFile, _ := os.Open(filepath.Join(paramFileDir, "PlyLevelTbl.prs"))
		compressedStatsFile, err := io.ReadAll(statsFile)
		if err != nil {
			initErr = fmt.Errorf("error loading PlyLevelTbl.prs:" + err.Error())
			return
		}

		decompressedSize, err := prs.DecompressSize(compressedStatsFile)
		if err != nil {
			initErr = fmt.Errorf("error decompressing size of PlyLevelTbl.prs: %v", err)
			return
		}

		decompressedStatsFile, err := prs.Decompress(compressedStatsFile, decompressedSize)
		if err != nil {
			initErr = fmt.Errorf("error decompressing PlyLevelTbl.prs: %v", err)
		}

		// Base character class stats are stored sequentially, each 14 bytes long.
		for i := 0; i < NumCharacterClasses; i++ {
			bytes.StructFromBytes(decompressedStatsFile[i*14:], &BaseStats[i])
		}
	})

	return initErr
}

// LoadConfig the PSOBB parameter files, build the parameter header,
// and init/cache the param file chunks for the EB packets.
func loadParameterFiles(logger *logrus.Logger, paramFileDir string) error {
	logger.Infof("loading parameters from %s", paramFileDir)

	offset := 0
	var tmpChunkData []byte

	for _, paramFile := range paramFiles {
		data, err := os.ReadFile(filepath.Join(paramFileDir, paramFile))
		if err != nil {
			return fmt.Errorf("error reading parameter file: %v", err)
		}

		fileSize := len(data)

		entry := &parameterEntry{
			Size:     uint32(fileSize),
			Checksum: crc32.ChecksumIEEE(data),
			Offset:   uint32(offset),
			Filename: [64]uint8{},
		}
		copy(entry.Filename[:], paramFile)

		bytes, _ := bytes.BytesFromStruct(entry)
		paramHeaderData = append(paramHeaderData, bytes...)
		tmpChunkData = append(tmpChunkData, data...)

		offset += fileSize

		logger.Infof("%s (%v bytes, checksum: 0x%x)", paramFile, fileSize, entry.Checksum)
	}

	// Offset should at this point be the total size of the files
	// to send - break it all up into chunks for indexing.
	paramChunkData = make(map[int][]byte)
	numChunks := offset / maxDataChunkSize
	for i := 0; i < numChunks; i++ {
		dataOff := i * maxDataChunkSize
		paramChunkData[i] = tmpChunkData[dataOff : dataOff+maxDataChunkSize]
		offset -= maxDataChunkSize
	}

	// Add any remaining data.
	if offset > 0 {
		paramChunkData[numChunks] = tmpChunkData[numChunks*maxDataChunkSize:]
	}
	return nil
}
