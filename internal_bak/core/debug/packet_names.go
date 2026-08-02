package debug

import "github.com/dcrodman/archon/internal/commands"

// Janky (and simple) method of including the names of the commands as Archon
// defines them. Of course whenever new command types are defined they must also
// be added here in order for the sniffer to get the name correctly.

var patchCommandNames = map[uint16]string{
	commands.DisconnectType:          "Disconnect",
	commands.RedirectType:            "Redirect",
	commands.PatchWelcomeType:        "PatchWelcome",
	commands.PatchHandshakeType:      "PatchHandshake",
	commands.PatchMessageType:        "PatchMessage",
	commands.PatchRedirectType:       "PatchRedirect",
	commands.PatchDataAckType:        "PatchDataAck",
	commands.PatchDirAboveType:       "PatchDirAbove",
	commands.PatchChangeDirType:      "PatchChangeDir",
	commands.PatchCheckFileType:      "PatchCheckFile",
	commands.PatchFileListDoneType:   "PatchFileListDone",
	commands.PatchFileStatusType:     "PatchFileStatus",
	commands.PatchClientListDoneType: "PatchClientListDone",
	commands.PatchStartFileUpdateType:    "PatchUpdateFiles",
	commands.PatchFileHeaderType:     "PatchFileHeader",
	commands.PatchFileChunkType:      "PatchFileChunk",
	commands.PatchFileCompleteType:   "PatchFileComplete",
	commands.PatchUpdateCompleteType: "PatchUpdateComplete",
}

var commandNames = map[uint16]string{
	commands.DisconnectType:              "Disconnect",
	commands.RedirectType:                "Redirect",
	commands.MenuSelectionType:           "MenuSelect",
	commands.LoginWelcomeType:            "LoginWelcome",
	commands.LoginType:                   "Login",
	commands.LoginSecurityType:           "LoginSecurity",
	commands.LoginClientMessageType:      "LoginClientMessage",
	commands.LoginOptionsRequestType:     "LoginOptionsRequest",
	commands.LoginOptionsType:            "LoginOptions",
	commands.LoginCharSelectType:         "LoginCharSelect",
	commands.LoginCharAckType:            "LoginCharAck",
	commands.LoginCharPreviewType:        "LoginCharPreview",
	commands.LoginChecksumType:           "LoginChecksum",
	commands.LoginChecksumAckType:        "LoginChecksumAck",
	commands.LoginGuildcardReqType:       "LoginGuildcardReq",
	commands.LoginGuildcardHeaderType:    "LoginGuildcardHeader",
	commands.LoginGuildcardChunkType:     "LoginGuildcardChunk",
	commands.LoginGuildcardChunkReqType:  "LoginGuildcardChunkReq",
	commands.LoginParameterHeaderType:    "LoginParameterHeader",
	commands.LoginParameterChunkType:     "LoginParameterChunk",
	commands.LoginParameterChunkReqType:  "LoginParameterChunkReq",
	commands.LoginParameterHeaderReqType: "LoginParameterHeaderReq",
	commands.LoginSetFlagType:            "LoginSetFlag",
	commands.LoginTimestampType:          "LoginTimestamp",
	commands.LoginShipListType:           "LoginShipList",
	commands.LoginScrollMessageType:      "LoginScrollMessage",
	commands.LobbyMenuType:               "LobbyList",
	commands.BlockListType:               "BlockList",
	commands.SyncCharacterType:           "FullCharacter",
}

func getCommandName(server string, commandType uint16) string {
	if server == "PATCH:AUTH" || server == "PATCH_DATA" {
		if name, ok := patchCommandNames[commandType]; ok {
			return name
		}
	} else {
		if name, ok := commandNames[commandType]; ok {
			return name
		}
	}
	return "?"
}
