package main

import "github.com/dcrodman/archon/internal/packets"

// Janky (and simple) method of including the names of the packets as Archon
// defines them. Of course whenever new packet types are defined they must also
// be added here in order for the sniffer to get the name correctly.

var patchPacketNames = map[uint16]string{
	packets.DisconnectType:          "DisconnectType",
	packets.RedirectType:            "RedirectType",
	packets.PatchWelcomeType:        "PatchWelcomeType",
	packets.PatchHandshakeType:      "PatchHandshakeType",
	packets.PatchMessageType:        "PatchMessageType",
	packets.PatchRedirectType:       "PatchRedirectType",
	packets.PatchDataAckType:        "PatchDataAckType",
	packets.PatchDirAboveType:       "PatchDirAboveType",
	packets.PatchChangeDirType:      "PatchChangeDirType",
	packets.PatchCheckFileType:      "PatchCheckFileType",
	packets.PatchFileListDoneType:   "PatchFileListDoneType",
	packets.PatchFileStatusType:     "PatchFileStatusType",
	packets.PatchClientListDoneType: "PatchClientListDoneType",
	packets.PatchUpdateFilesType:    "PatchUpdateFilesType",
	packets.PatchFileHeaderType:     "PatchFileHeaderType",
	packets.PatchFileChunkType:      "PatchFileChunkType",
	packets.PatchFileCompleteType:   "PatchFileCompleteType",
	packets.PatchUpdateCompleteType: "PatchUpdateCompleteType",
}

var packetNames = map[uint16]string{
	packets.DisconnectType:              "DisconnectType",
	packets.RedirectType:                "RedirectType",
	packets.MenuSelectType:              "MenuSelectType",
	packets.LoginWelcomeType:            "LoginWelcomeType",
	packets.LoginType:                   "LoginType",
	packets.LoginSecurityType:           "LoginSecurityType",
	packets.LoginClientMessageType:      "LoginClientMessageType",
	packets.LoginOptionsRequestType:     "LoginOptionsRequestType",
	packets.LoginOptionsType:            "LoginOptionsType",
	packets.LoginCharPreviewReqType:     "LoginCharPreviewReqType",
	packets.LoginCharAckType:            "LoginCharAckType",
	packets.LoginCharPreviewType:        "LoginCharPreviewType",
	packets.LoginChecksumType:           "LoginChecksumType",
	packets.LoginChecksumAckType:        "LoginChecksumAckType",
	packets.LoginGuildcardReqType:       "LoginGuildcardReqType",
	packets.LoginGuildcardHeaderType:    "LoginGuildcardHeaderType",
	packets.LoginGuildcardChunkType:     "LoginGuildcardChunkType",
	packets.LoginGuildcardChunkReqType:  "LoginGuildcardChunkReqType",
	packets.LoginParameterHeaderType:    "LoginParameterHeaderType",
	packets.LoginParameterChunkType:     "LoginParameterChunkType",
	packets.LoginParameterChunkReqType:  "LoginParameterChunkReqType",
	packets.LoginParameterHeaderReqType: "LoginParameterHeaderReqType",
	packets.LoginSetFlagType:            "LoginSetFlagType",
	packets.LoginTimestampType:          "LoginTimestampType",
	packets.LoginShipListType:           "LoginShipListType",
	packets.LoginScrollMessageType:      "LoginScrollMessageType",
	packets.LobbyListType:               "LobbyListType",
	packets.BlockListType:               "BlockListType",
	packets.FullCharacterType:           "FullCharacterType",
	packets.FullCharacterEndType:        "FullCharacterEndType",
}

func getPacketName(server ServerType, packetType uint16) string {
	if server == PATCH_SERVER || server == DATA_SERVER {
		if name, ok := patchPacketNames[packetType]; ok {
			return name
		}
	} else {
		if name, ok := packetNames[packetType]; ok {
			return name
		}
	}
	return "?"
}
