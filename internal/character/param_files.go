package character

import (
	"embed"
	"fmt"
	"hash/crc32"
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

	parametersDirName = "parameters"
)

var (
	paramInitLock sync.Once

	// Directly embedding the vanilla parameter files to make it a little easier to
	// run the server for most people.
	// TODO: Support overriding these (see config.go).
	//
	//go:embed parameters/*
	paramFiles embed.FS

	// Cached parameter data to avoid computing it every time.
	paramHeaderData []byte
	paramChunkData  map[int][]byte

	// Starting stats for any new character. The CharClass constants can be used
	// to index into this array to obtain the base stats for each class.
	BaseStats [NumCharacterClasses]stats
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

func initParameterData(logger *logrus.Logger) (int, error) {
	var (
		numFilesLoaded int
		initErr        error
	)

	paramInitLock.Do(func() {
		var err error
		if numFilesLoaded, err = loadParameterFiles(logger); err != nil {
			initErr = fmt.Errorf("error loading parameter files: %w", err)
			return
		}

		// LoadConfig the base stats for creating new characters.
		compressedStatsData, err := paramFiles.ReadFile(filepath.Join(parametersDirName, "PlyLevelTbl.prs"))
		if err != nil {
			initErr = fmt.Errorf("error loading PlyLevelTbl.prs: %w", err)
			return
		}

		decompressedSize, err := prs.DecompressSize(compressedStatsData)
		if err != nil {
			initErr = fmt.Errorf("error decompressing size of PlyLevelTbl.prs: %w", err)
			return
		}

		decompressedStatsFile, err := prs.Decompress(compressedStatsData, decompressedSize)
		if err != nil {
			initErr = fmt.Errorf("error decompressing PlyLevelTbl.prs: %w", err)
		}

		// Base character class stats are stored sequentially, each 14 bytes long.
		for i := 0; i < NumCharacterClasses; i++ {
			bytes.StructFromBytes(decompressedStatsFile[i*14:], &BaseStats[i])
		}
	})

	return numFilesLoaded, initErr
}

// LoadConfig the PSOBB parameter files, build the parameter header,
// and init/cache the param file chunks for the EB packets.
func loadParameterFiles(logger *logrus.Logger) (int, error) {
	logger.Info("loading embedded parameter files")

	offset := 0
	var tmpChunkData []byte

	defaultFiles, err := paramFiles.ReadDir(parametersDirName)
	if err != nil {
		return 0, fmt.Errorf("error loading embedded parameter files: %w", err)
	}

	var numFilesLoaded int
	for _, paramFile := range defaultFiles {
		data, err := paramFiles.ReadFile(fmt.Sprintf("%s/%s", parametersDirName, paramFile.Name()))
		if err != nil {
			return 0, fmt.Errorf("error reading parameter file: %w", err)
		}

		numFilesLoaded++
		fileSize := len(data)

		entry := &parameterEntry{
			Size:     uint32(fileSize),
			Checksum: crc32.ChecksumIEEE(data),
			Offset:   uint32(offset),
			Filename: [64]uint8{},
		}
		copy(entry.Filename[:], []uint8(paramFile.Name()))

		bytes, _ := bytes.BytesFromStruct(entry)
		paramHeaderData = append(paramHeaderData, bytes...)
		tmpChunkData = append(tmpChunkData, data...)

		offset += fileSize

		logger.Debugf("%s (%v bytes)", paramFile.Name(), fileSize)
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
	return numFilesLoaded, nil
}
