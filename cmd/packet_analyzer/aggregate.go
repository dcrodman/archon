package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const bytesPerRow = 16

var (
	// Order in which the session files will be presented in the emitted file.
	serverOrder = []string{
		"patch",
		"data",
		"login",
		"character",
		"ship",
		"block",
	}

	patchPacketNames = map[uint64]string{
		0x02: "PatchWelcomeType",
		0x04: "PatchHandshakeType",
		0x13: "PatchMessageType",
		0x14: "PatchRedirectType",
		0x0B: "PatchDataAckType",
		0x0A: "PatchDirAboveType",
		0x09: "PatchChangeDirType",
		0x0C: "PatchCheckFileType",
		0x0D: "PatchFileListDoneType",
		0x0F: "PatchFileStatusType",
		0x10: "PatchClientListDoneType",
		0x11: "PatchUpdateFilesType",
		0x06: "PatchFileHeaderType",
		0x07: "PatchFileChunkType",
		0x08: "PatchFileCompleteType",
		0x12: "PatchUpdateCompleteType",
	}

	packetNames = map[uint64]string{
		0x03:   "LoginWelcomeType            ",
		0x93:   "LoginType                   ",
		0xE6:   "LoginSecurityType           ",
		0x1A:   "LoginClientMessageType      ",
		0xE0:   "LoginOptionsRequestType     ",
		0xE2:   "LoginOptionsType            ",
		0xE3:   "LoginCharPreviewReqType     ",
		0xE4:   "LoginCharAckType            ",
		0xE5:   "LoginCharPreviewType        ",
		0x01E8: "LoginChecksumType          ",
		0x02E8: "LoginChecksumAckType       ",
		0x03E8: "LoginGuildcardReqType      ",
		0x01DC: "LoginGuildcardHeaderType   ",
		0x02DC: "LoginGuildcardChunkType    ",
		0x03DC: "LoginGuildcardChunkReqType ",
		0x01EB: "LoginParameterHeaderType   ",
		0x02EB: "LoginParameterChunkType    ",
		0x03EB: "LoginParameterChunkReqType ",
		0x04EB: "LoginParameterHeaderReqType",
		0xEC:   "LoginSetFlagType           ",
		0xB1:   "LoginTimestampType         ",
		0xA0:   "LoginShipListType          ",
		0xEE:   "LoginScrollMessageType     ",
		0x05:   "DisconnectType",
		0x19:   "RedirectType  ",
		0x10:   "MenuSelectType",
		0x83:   "LobbyListType       ",
		0x07:   "BlockListType       ",
		0xE7:   "FullCharacterType   ",
		0x95:   "FullCharacterEndType",
	}
)

func aggregateFiles() {
	if flag.NArg() < 2 {
		fmt.Println("usage: aggregate [file.session...]")
		os.Exit(1)
	}

	var serverName string
	var sessionFiles []string
	for i := 1; i < flag.NArg(); i++ {
		filename := flag.Arg(i)
		filenameParts := strings.Split(filename, "/")
		s := strings.Split(filenameParts[len(filenameParts)-1], "_")[0]

		if serverName == "" {
			serverName = s
		} else if s != serverName {
			// There's not really a technical limitation preventing two servers from
			// being processed together but it doesn't really make sense with this tool.
			fmt.Println("error: files cannot be aggregated across servers")
			os.Exit(1)
		}
		sessionFiles = append(sessionFiles, filename)
	}

	outputFilename := fmt.Sprintf("%s_aggregated.md", serverName)
	f, err := os.Create(outputFilename)
	if err != nil {
		fmt.Printf("error: unable to create markdown file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	// Write the file header.
	f.WriteString(fmt.Sprintf("# %s\n\n", strings.Title(serverName)))

	// Write each file as a subheading, with all packets in that file formatted under that.
	for _, subserver := range serverOrder {
		for _, sessionFile := range sessionFiles {
			if strings.Contains(sessionFile, subserver) {
				writeMarkdownForFile(f, subserver, sessionFile)
				break
			}
		}
	}

	fmt.Println("wrote", outputFilename)
}

func writeMarkdownForFile(f *os.File, subserver, filename string) {
	f.WriteString(fmt.Sprintf("## %s Server\n", strings.Title(subserver)))

	sessionData, err := parseSessionDataFromFile(filename)
	if err != nil {
		fmt.Printf("error parsing %v: %v\n", filename, err)
		os.Exit(1)
	}

	for _, packet := range sessionData.Packets {
		pType, _ := strconv.ParseUint(packet.Type, 16, 32)
		f.WriteString(fmt.Sprintf("### 0x%.4X\n", pType))

		f.WriteString(fmt.Sprintf("Canonical name: %s\n", getPacketName(subserver, pType)))

		f.WriteString(fmt.Sprintf("Direction: %s -> %s\n", packet.Source, packet.Destination))

		size, _ := strconv.ParseInt(packet.Size, 16, 32)
		f.WriteString(fmt.Sprintf("Size: 0x%.4X\n", size))

		f.WriteString("```\n")
		buf := bufio.NewWriter(f)
		writePacketBodyToFile(buf, &packet)
		buf.Flush()
		f.WriteString("```\n")

		f.WriteString("\n")
	}
}

func getPacketName(subserver string, ptype uint64) string {
	var name string
	var ok bool

	if subserver == "patch" || subserver == "data" {
		name, ok = patchPacketNames[ptype]
	} else {
		name, ok = packetNames[ptype]
	}

	if !ok {
		return "Unknown"
	}
	return name
}
