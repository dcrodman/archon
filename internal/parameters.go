package internal

import (
	"fmt"
	"hash/crc32"
	"os"
)

// Relative directory containing the various parameter files such as
// BattleParamEntry*, ItemPMT.prs, etc.
const parametersDirName = "parameters"

var (
	// List of files that the client expects to receive during the character
	// server login process.
	requiredClientFiles = []string{
		"BattleParamEntry.dat",
		"BattleParamEntry_on.dat",
		"BattleParamEntry_ep4.dat",
		"BattleParamEntry_ep4_on.dat",
		"BattleParamEntry_lab.dat",
		"BattleParamEntry_lab_on.dat",
		"ItemMagEdit.prs",
		"ItemPMT.prs",
		"PlyLevelTbl.prs",
	}
	otherParamFiles = []string{
		"ItemPT.gsl",
	}
	// Cached parameter data to avoid computing it every time for the client
	// login process.
	paramHeaderData []byte
	paramChunkData  map[int][]byte
)

// Struct for caching the parameter chunk data and header so
// that the param files aren't re-read every time.
type parameterEntry struct {
	Size     uint32
	Checksum uint32
	Offset   uint32
	Filename [0x40]uint8
}

// LoadParameterFiles loads the PSOBB parameter files, build the parameter header,
// and init/cache the param file chunks for the EB commands.
func LoadParameterFiles() error {
	var (
		offset    = 0
		chunkData []byte
	)
	Logger.Info("loading embedded parameter files")

	for _, paramFile := range requiredClientFiles {
		data, err := os.ReadFile(fmt.Sprintf("%s/%s", parametersDirName, paramFile))
		if err != nil {
			return fmt.Errorf("error reading parameter file: %w", err)
		}

		// Build the header data for this file.
		fileSize := len(data)
		entry := &parameterEntry{
			Size:     uint32(fileSize),
			Checksum: crc32.ChecksumIEEE(data),
			Offset:   uint32(offset),
		}
		copy(entry.Filename[:], []uint8(paramFile))

		bytes, _ := MarshalStruct(entry)
		paramHeaderData = append(paramHeaderData, bytes...)

		// Append the contents of the file to the working set of chunk data.
		chunkData = append(chunkData, data...)
		offset += fileSize

		// Pass the contents of the file off to the appropriate init function if needed.
		initParamFile(paramFile, data)

		Logger.Debugf("%s (%v bytes)", paramFile, fileSize)
	}

	for _, paramFile := range otherParamFiles {
		data, err := os.ReadFile(fmt.Sprintf("%s/%s", parametersDirName, paramFile))
		if err != nil {
			return fmt.Errorf("error reading parameter file: %w", err)
		}
		// Pass the contents of the file off to the appropriate init function if needed.
		initParamFile(paramFile, data)

		Logger.Debugf("%s (%v bytes)", paramFile, len(data))
	}

	// Offset should at this point be the total size of the files to send;
	// break it all up into chunks for indexing.
	paramChunkData = make(map[int][]byte)
	numChunks := offset / maxDataChunkSize
	for i := range numChunks {
		dataOff := i * maxDataChunkSize
		paramChunkData[i] = chunkData[dataOff : dataOff+maxDataChunkSize]
		offset -= maxDataChunkSize
	}

	// Add any remaining data.
	if offset > 0 {
		paramChunkData[numChunks] = chunkData[numChunks*maxDataChunkSize:]
	}
	return nil
}

func initParamFile(filename string, data []byte) {
	switch filename {
	case "PlyLevelTbl.prs":
		InitLevelTable(data)
	case "ItemPT.gsl":
		InitItemPT(data)
	}
}
